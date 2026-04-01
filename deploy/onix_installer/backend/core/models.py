# Copyright 2025 Google LLC
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
from typing import Annotated, Dict, Optional, Any
from pydantic.main import BaseModel
from pydantic.networks import HttpUrl
from pydantic.fields import Field


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
    """
    project_id: NonEmptyStr
    region: NonEmptyStr
    app_name: NonEmptyStr
    type: DeploymentType
    components: Dict[str, bool]
    # Expected keys for components: "gateway", "registry", "bap", "bpp"

class AdapterConfig(BaseModel):
    """
    Configuration specific to the Adapter service.
    """
    enable_schema_validation: Optional[bool] = False

class RegistryConfig(BaseModel):
    """
    Configuration specific to the Registry service.
    """
    subscriber_id: NonEmptyStr
    key_id: NonEmptyStr
    enable_auto_approver: Optional[bool] = False

class GatewayConfig(BaseModel):
    """
    Configuration specific to the Gateway service.
    """
    subscriber_id: NonEmptyStr

class DomainConfig(BaseModel):
    domainType: NonEmptyStr
    baseDomain: str
    dnsZone: str
    
class AppDeploymentRequest(BaseModel):
    """
    Pydantic model for incoming application deployment requests.
    """
    app_name: NonEmptyStr
    components: Dict[str, bool]
    # Expected keys for components: "gateway", "registry", "bap", "bpp"
    domain_names: Dict[str, NonEmptyStr]
    # Expected keys for domain_names: "registry", "registry_admin", "subscriber", "gateway", "adapter"
    image_urls: Dict[str, NonEmptyStr]
    # Expected keys for image_urls: "registry", "registry_admin", "subscriber", "gateway", "adapter"

    registry_url: HttpUrl

    registry_config: RegistryConfig
    domain_config: DomainConfig
    adapter_config: Optional[AdapterConfig] = None
    gateway_config: Optional[GatewayConfig] = None


class ProxyRequest(BaseModel):
    targetUrl: str
    payload: Dict[Any, Any]