from typing import Optional, Any
from pydantic import BaseModel, Field, EmailStr
from datetime import datetime
from core.constants import (
    DeploymentStatus, OperationStatus, OperationType, 
    RoleType, ComputeType, SubscriptionStatus, AuditActionType
)

# --- Sub-Structures ---

class Tags(BaseModel):
    network_group: str
    environment: str
    role: RoleType
    custom_tags: dict[str, Any] = Field(default_factory=dict)

class Versions(BaseModel):
    installer_version: str
    onix_bundle_version: str
    adapter_version: str

class OperationSnapshot(BaseModel):
    operation_id: str
    type: OperationType
    status: OperationStatus
    stage: Optional[str] = None
    error_summary: Optional[str] = None

class Lock(BaseModel):
    is_locked: bool = False
    locked_by_operation_id: Optional[str] = None
    locked_at: Optional[datetime] = None

class DeploymentConfig(BaseModel):
    project_id: str  # Immutable
    region: str      # Immutable
    compute_type: ComputeType = ComputeType.GKE
    deployment_size: str = "small"
    components: dict[str, bool] = Field(default_factory=dict)
    domain_config: dict[str, Any] = Field(default_factory=dict)
    image_urls: Optional[dict[str, str]] = Field(default_factory=dict)
    adapter_config: Optional[dict[str, Any]] = Field(default_factory=dict)
    registry_config: Optional[dict[str, Any]] = Field(default_factory=dict)
    gateway_config: Optional[dict[str, Any]] = Field(default_factory=dict)

class Artifacts(BaseModel):
    gcs_bucket: str
    tf_state_path: str
    tf_state_generation: str
    live_config_snapshot_id: str

# --- Main Collections ---

class Deployment(BaseModel):
    deployment_id: str
    name: str
    resource_suffix: str
    created_by: EmailStr
    created_at: datetime = Field(default_factory=datetime.utcnow)
    last_updated_by: EmailStr
    last_updated_at: datetime = Field(default_factory=datetime.utcnow)
    tags: Tags
    versions: Versions
    status: DeploymentStatus
    lock: Lock
    latest_operation: Optional[OperationSnapshot] = None
    config: DeploymentConfig
    infra_details: dict[str, str] = Field(default_factory=dict)
    artifacts: Optional[Artifacts] = None
    health_status: str = "UNKNOWN"

class Operation(BaseModel):
    operation_id: str
    deployment_id: str
    target_config: dict[str, Any]
    type: OperationType
    status: OperationStatus
    started_at: datetime = Field(default_factory=datetime.utcnow)
    completed_at: Optional[datetime] = None
    triggered_by: EmailStr
    worker_id: Optional[str] = None
    error_message: Optional[str] = None
    stage_failed: Optional[str] = None
    cloud_logging_link: Optional[str] = None

class AuditLog(BaseModel):
    audit_log_id: str
    deployment_id: str
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    actor_email: EmailStr
    action_type: AuditActionType
    details: dict[str, Any]
    status_result: str

class Subscription(BaseModel):
    subscriber_id: str
    registry_url: str
    status: SubscriptionStatus
    created_at: datetime = Field(default_factory=datetime.utcnow)
    error_detail: Optional[str] = None