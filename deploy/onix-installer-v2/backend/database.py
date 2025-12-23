import uuid
import os
from datetime import datetime
from typing import Optional, Any
from google.cloud import firestore
from fastapi import HTTPException
from dotenv import load_dotenv

import secrets
import string


from core.constants import OperationStatus, OperationType, DeploymentStatus
from core.models import Deployment, Operation, AuditLog, OperationSnapshot

load_dotenv()
PROJECT_ID = os.getenv("GOOGLE_CLOUD_PROJECT")
DATABASE_NAME = os.getenv("FIRESTORE_DATABASE_NAME")



def get_db_client():
    """
    Returns a Firestore client. 
    If database_id is provided, it connects to that specific database instance.
    """
    print(DATABASE_NAME)
    return firestore.Client(project=PROJECT_ID, database=DATABASE_NAME)

# Initialize the default client for the application
db = get_db_client()

COLLECTION_DEPLOYMENTS = "deployments"
SUB_COLLECTION_OPERATIONS = "operations"
SUB_COLLECTION_AUDIT = "audit_logs"

def generate_unique_suffix(max_attempts: int = 3) -> str:
    """
    Generates a unique 6-character alphanumeric resource suffix.
    Performs a uniqueness check loop as per Section 6.1.1 of the design doc.
    """
    alphabet = string.ascii_lowercase + string.digits
    
    for attempt in range(max_attempts):
        # Generate a candidate 6-char suffix
        candidate = ''.join(secrets.choice(alphabet) for _ in range(6))
        
        # Query Firestore to check for collisions
        existing = (
            db.collection(COLLECTION_DEPLOYMENTS)
            .where("resource_suffix", "==", candidate)
            .limit(1)
            .get()
        )
        
        if not existing:
            return candidate
            
        print(f"DEBUG: Suffix collision detected for '{candidate}'. Retrying...")

    # If all attempts fail, raise a 500 error as per the spec
    raise HTTPException(
        status_code=500,
        detail="Failed to generate a unique resource suffix after multiple attempts."
    )

def get_deployment_ref(deployment_id: str):
    """Returns a DocumentReference for a specific deployment."""
    return db.collection(COLLECTION_DEPLOYMENTS).document(deployment_id)

def get_operation_ref(deployment_id: str, op_id: str):
    """Returns a DocumentReference for an operation in the sub-collection."""
    return (
        db.collection(COLLECTION_DEPLOYMENTS)
        .document(deployment_id)
        .collection(SUB_COLLECTION_OPERATIONS)
        .document(op_id)
    )

def save_new_deployment(deployment: Deployment):
    """Saves a new DRAFT deployment document."""
    doc_ref = get_deployment_ref(deployment.deployment_id)
    # model_dump() is the Pydantic v2 way to get a dict
    doc_ref.set(deployment.model_dump())
    return deployment.deployment_id

def list_all_deployments(
    network_group: Optional[str] = None,
    environment: Optional[str] = None,
    role: Optional[str] = None
) -> list[dict[str, Any]]:
    """Lists all deployments with optional filtering (Section 6.1.2)."""
    query = db.collection(COLLECTION_DEPLOYMENTS)
    
    # Apply optional equality filters
    if network_group:
        print("inside if")
        query = query.where("tags.network_group", "==", network_group)
    if environment:
        query = query.where("tags.environment", "==", environment)
    if role:
        query = query.where("tags.role", "==", role)
    
    
    # Firestore allows ordering by a field after equality filters
    docs = query.order_by("last_updated_at", direction=firestore.Query.DESCENDING).stream()
    return [doc.to_dict() for doc in docs]

def log_audit_event(deployment_id: str, audit: AuditLog):
    """Records a business audit event in the sub-collection (Section 3.3)."""
    evt_id = f"evt-{uuid.uuid4()}"
    audit_ref = (
        db.collection(COLLECTION_DEPLOYMENTS)
        .document(deployment_id)
        .collection(SUB_COLLECTION_AUDIT)
        .document(evt_id)
    )
    audit_ref.set(audit.model_dump())