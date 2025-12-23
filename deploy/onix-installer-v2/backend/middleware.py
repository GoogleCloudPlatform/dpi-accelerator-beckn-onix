from fastapi import Request, HTTPException, status

def get_current_user(request: Request) -> str:
    """
    Extracts the authenticated user's email from the IAP header.
    Enforces security by raising a 401 error if the header is missing.
    """
    email_header = request.headers.get("X-Goog-Authenticated-User-Email")
    
    if not email_header:
        # Raise 401 Unauthorized for production-level security
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Authentication required: Missing X-Goog-Authenticated-User-Email header"
        )
    
    # Format is typically 'accounts.google.com:email@example.com'
    return email_header.split(":")[-1]