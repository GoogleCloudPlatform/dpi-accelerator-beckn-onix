import uuid
import asyncio
from datetime import datetime
from typing import Any
from fastapi import Body
from fastapi import FastAPI, HTTPException, BackgroundTasks, Depends, Request
from pydantic import EmailStr

from core.constants import DeploymentStatus, OperationStatus, OperationType, RoleType, ComputeType
from core.models import (
    Deployment, DeploymentConfig, Lock, OperationSnapshot, 
    Tags, Versions, AuditLog, AuditActionType,
    DeploymentCreateRequest, DeploymentCreateResponse
)
import database as db_ops
from middleware import get_current_user

app = FastAPI(title="Onix Management Portal (OMP) v2")

@app.post("/api/deployments", status_code=201, response_model=DeploymentCreateResponse)
async def create_deployment(
    req: DeploymentCreateRequest,
    user_email: str = Depends(get_current_user)
):
    """
    Initializes metadata, generates unique suffix, and persists to Firestore.
    """
    deployment_id = str(uuid.uuid4())
    
    suffix = db_ops.generate_unique_suffix()

    new_deployment = Deployment(
        deployment_id=deployment_id,
        name=req.name,
        resource_suffix=suffix,
        created_by=user_email,
        last_updated_by=user_email,
        tags=req.tags,
        versions=Versions(    # For Now Hard Coded will have to check how we can do that
            installer_version="v2.0.0",
            onix_bundle_version="google-onix-v2.1",
            adapter_version="v1.0.0"
        ),
        status=DeploymentStatus.DRAFT,
        lock=Lock(),
        config=req.config,
        health_status="UNKNOWN"
    )
    
    # Save to Firestore
    db_ops.save_new_deployment(new_deployment)
    
    # Log Audit Event
    db_ops.log_audit_event(deployment_id, AuditLog(
        audit_log_id=f"evt-{uuid.uuid4()}",
        deployment_id=deployment_id,
        actor_email=user_email,
        action_type=AuditActionType.DEPLOYMENT_CREATED,
        details={"message": "Initial DRAFT created"},
        status_result="SUCCESS"
    ))

    return DeploymentCreateResponse(
        deployment_id= deployment_id, 
        resource_suffix= suffix
        )

@app.get("/api/deployments", response_model=list[Deployment])
async def list_deployments(
    network_group: str = None,
    environment: str = None,
    role: RoleType = None
):
    return db_ops.list_all_deployments(
        network_group=network_group,
        environment=environment,
        role=role
    )

@app.get("/api/deployments/{id}", response_model=Deployment)
async def get_deployment(id: str):
    """Get details for a specific deployment."""
    doc = db_ops.get_deployment_ref(id).get()
    if not doc.exists:
        raise HTTPException(status_code=404, detail="Deployment not found")
    return doc.to_dict()

@app.get("/health")
async def health():
    return {"status": "healthy", "timestamp": datetime.utcnow().isoformat()}