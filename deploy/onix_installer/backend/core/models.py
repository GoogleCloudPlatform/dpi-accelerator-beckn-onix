# Copyright 2026 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from enum import Enum
from typing import Annotated, Any, Dict, Optional

from pydantic.fields import Field
from pydantic.functional_validators import model_validator
from pydantic.main import BaseModel
from pydantic.networks import HttpUrl


# Define a type alias for non-empty strings
NonEmptyStr = Annotated[str, Field(min_length=1)]

# Define an Enum for the allowed deployment types
class DeploymentType(str, Enum):
    """
    Defines the allowed types for infrastructure deployments.
    """
    SMALL = "small"
    MEDIUM = "medium"
    LARGE = "large"

class InfraDeploymentRequest(BaseModel):
    """
    Pydantic model for incoming infrastructure deployment requests.

    Attributes:
        project_id: The Google Cloud Project ID.
        region: The Google Cloud region for deployment.
        app_name: The name of the application.
        type: The deployment size type (e.g., "small", "medium", "large").
        components: A dictionary indicating which components to deploy.
        enable_cloud_armor: Whether to enable Cloud Armor. Defaults to False.
        allowed_regions: A tuple of regions. Defaults to ("IN",).
        rate_limit_count: Max requests per time frame. Defaults to 100.
    """
    project_id: NonEmptyStr
    region: NonEmptyStr
    app_name: NonEmptyStr
    type: DeploymentType
    components: dict[str, bool]
    # Expected keys for components: "gateway", "registry", "bap", "bpp"
    enable_cloud_armor: bool | None = False
    rate_limit_count: int | None = 100

class AdapterConfig(BaseModel):
    """
    Configuration specific to the Adapter service.
    """
    enable_schema_validation: bool | None = False

class RegistryConfig(BaseModel):
    """
    Configuration specific to the Registry service.
    """
    subscriber_id: NonEmptyStr
    key_id: NonEmptyStr
    enable_auto_approver: bool | None = False

class GatewayConfig(BaseModel):
    """
    Configuration specific to the Gateway service.
    """
    subscriber_id: NonEmptyStr

class SecurityConfig(BaseModel):
    """
    Configuration specific to security policies.
    """
    enable_inbound_auth: bool | None = False
    issuer_url: str | None = None
    idclaim: str | None = None
    allowed_values: list[str] | None = None
    jwks_content: str | None = None
    enable_outbound_auth: bool | None = False
    aud_overrides: str | None = None

    @model_validator(mode='after')
    def validate_inbound_auth_requirements(self) -> 'SecurityConfig':
        # Only run this check if inbound auth is explicitly set to True
        if self.enable_inbound_auth:
            missing_fields = []
            # Check if fields are either None or empty strings
            if not self.issuer_url:
                missing_fields.append("issuer_url")
            if not self.idclaim:
                missing_fields.append("idclaim")
            # This checks for both None and an empty list []
            if not self.allowed_values:
                missing_fields.append("allowed_values")

            if missing_fields:
              raise ValueError(
                  f"When enable_inbound_auth is True, the following fields cannot be empty: {', '.join(missing_fields)}"
                )
        return self


class DomainConfig(BaseModel):
    
    domainType: NonEmptyStr
    baseDomain: str
    dnsZone: str


class ConfigGenerationRequest(BaseModel):
    """
    Model for generating application configuration files (YAMLs).
    Contains only the fields necessary for rendering service configs.
    """
    app_name: NonEmptyStr
    components: dict[str, bool]
    registry_url: HttpUrl
    registry_config: RegistryConfig
    adapter_config: AdapterConfig| None = None
    gateway_config: GatewayConfig | None = None
    security_config: SecurityConfig | None = None
    domain_names: dict[str, NonEmptyStr] = {}


    
class AppDeploymentRequest(BaseModel):
    """
    Pydantic model for incoming application deployment requests.
    """
    app_name: NonEmptyStr
    components: dict[str, bool]
    # Expected keys for components: "gateway", "registry", "bap", "bpp"
    domain_names: dict[str, NonEmptyStr]
    # Expected keys for domain_names: "registry", "registry_admin", "subscriber", "gateway", "adapter"
    image_urls: dict[str, NonEmptyStr]
    # Expected keys for image_urls: "registry", "registry_admin", "subscriber", "gateway", "adapter"

    registry_url: HttpUrl

    registry_config: RegistryConfig
    domain_config: DomainConfig
    adapter_config: AdapterConfig | None = None
    gateway_config: GatewayConfig | None = None
    security_config: SecurityConfig | None = None


class ProxyRequest(BaseModel):
    target_url: str
    payload: Dict[Any, Any]
    impersonate_service_account: Optional[str] = None
    audience: Optional[str] = None

class ConfigUpdateRequest(BaseModel):
    path: NonEmptyStr
    content: str
