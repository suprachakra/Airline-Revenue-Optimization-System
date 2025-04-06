# AuthUtils.py
"""
Authentication Helpers
----------------------
Provides shared functions for secure token validation, session management, and multi-factor auth.
"""

import jwt
from datetime import datetime, timedelta

SECRET_KEY = "YOUR_SUPER_SECRET_KEY"  # This should be loaded from secure configuration

def generate_token(user_id: int, roles: list, expires_in: int = 3600) -> str:
    payload = {
        "user_id": user_id,
        "roles": roles,
        "exp": datetime.utcnow() + timedelta(seconds=expires_in),
        "iat": datetime.utcnow()
    }
    return jwt.encode(payload, SECRET_KEY, algorithm="HS256")

def validate_token(token: str) -> dict:
    try:
        decoded = jwt.decode(token, SECRET_KEY, algorithms=["HS256"])
        return decoded
    except jwt.ExpiredSignatureError:
        raise Exception("Token expired")
    except jwt.InvalidTokenError:
        raise Exception("Invalid token")
