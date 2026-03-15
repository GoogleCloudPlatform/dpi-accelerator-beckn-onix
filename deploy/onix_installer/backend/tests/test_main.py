import pytest
import httpx
from unittest.mock import patch, MagicMock, AsyncMock
from main import app

RealAsyncClient = httpx.AsyncClient

@pytest.fixture
def mock_httpx_client():
    mock_instance = AsyncMock()
    mock_response = MagicMock()
    mock_response.status_code = 200
    mock_response.content = b"Success"
    mock_response.text = "Success"
    mock_response.raise_for_status = MagicMock()
    mock_instance.post.return_value = mock_response

    def client_factory(*args, **kwargs):
        if "app" in kwargs:
            return RealAsyncClient(*args, **kwargs)
        mock_client = MagicMock()
        mock_client.__aenter__.return_value = mock_instance
        return mock_client

    with patch('main.httpx.AsyncClient', side_effect=client_factory):
        yield mock_instance

@pytest.fixture
def mock_google_auth():
    with patch('main.google.auth.default') as mock_default, \
         patch('main.impersonated_credentials.Credentials') as mock_credentials, \
         patch('main.impersonated_credentials.IDTokenCredentials') as mock_id_token_credentials, \
         patch('main.google_auth_requests.Request') as mock_request:
        
        mock_default.return_value = (MagicMock(), "project-id")
        
        mock_id_token_instance = MagicMock()
        mock_id_token_instance.token = "mock_oidc_token"
        mock_id_token_credentials.return_value = mock_id_token_instance

        yield {
            "default": mock_default,
            "credentials": mock_credentials,
            "id_token_credentials": mock_id_token_credentials,
            "request": mock_request
        }

@pytest.mark.asyncio
async def test_dynamic_proxy_success(mock_httpx_client):
    async with httpx.AsyncClient(app=app, base_url="http://test") as client:
        payload = {
            "target_url": "https://target.api/data",
            "payload": {"key": "value"}
        }
        response = await client.post("/api/dynamic-proxy", json=payload)
        assert response.status_code == 200
        assert mock_httpx_client.post.call_count == 1
        call_args = mock_httpx_client.post.call_args
        assert call_args[0][0] == "https://target.api/data"
        assert call_args[1]["json"] == {"key": "value"}
        assert "Authorization" not in call_args[1].get("headers", {})

@pytest.mark.asyncio
async def test_dynamic_proxy_impersonation_success(mock_httpx_client, mock_google_auth):
  async with httpx.AsyncClient(app=app, base_url="http://test") as client:
    payload = {
        "target_url": "https://target.api/data",
        "payload": {"key": "value"},
        "impersonate_service_account": (
            "test-sa@project.iam.gserviceaccount.com"
        ),
        "audience": "https://audience.api",
    }
    response = await client.post("/api/dynamic-proxy", json=payload)

    assert response.status_code == 200
    assert mock_google_auth["default"].call_count == 1
    assert mock_google_auth["credentials"].call_count == 1
    assert mock_google_auth["id_token_credentials"].call_count == 1

    # Verify the target credentials config
    cred_args = mock_google_auth["credentials"].call_args
    assert (
        cred_args[1]["target_principal"]
        == "test-sa@project.iam.gserviceaccount.com"
    )
    assert cred_args[1]["target_scopes"] == [
        "https://www.googleapis.com/auth/cloud-platform"
    ]

    # Verify ID token args
    id_token_args = mock_google_auth["id_token_credentials"].call_args
    assert id_token_args[1]["target_audience"] == "https://audience.api"

    assert mock_httpx_client.post.call_count == 1
    call_args = mock_httpx_client.post.call_args
    assert call_args[1]["headers"]["Authorization"] == "Bearer mock_oidc_token"

@pytest.mark.asyncio
async def test_dynamic_proxy_impersonation_failure(mock_httpx_client, mock_google_auth):
  mock_google_auth["id_token_credentials"].side_effect = Exception(
      "Impersonation failed"
  )

  async with httpx.AsyncClient(app=app, base_url="http://test") as client:
    payload = {
        "target_url": "https://target.api/data",
        "payload": {"key": "value"},
        "impersonate_service_account": (
            "test-sa@project.iam.gserviceaccount.com"
        ),
        "audience": "https://audience.api",
    }
    response = await client.post("/api/dynamic-proxy", json=payload)

    assert response.status_code == 500
    assert "Impersonation failed" in response.text
    assert mock_httpx_client.post.call_count == 0
