# User.py
"""
User Model
----------
Defines the user data model used for authentication and personalization.
Incorporates secure password hashing and multi-factor authentication readiness.
"""

import bcrypt

class User:
    def __init__(self, user_id: int, email: str, password: str, roles: list):
        self.user_id = user_id
        self.email = email
        self.roles = roles  # E.g., ["pricing_admin", "revenue_manager"]
        self.password_hash = self.hash_password(password)

    def hash_password(self, password: str) -> bytes:
        # Generate a bcrypt hash for secure password storage.
        return bcrypt.hashpw(password.encode('utf-8'), bcrypt.gensalt())

    def check_password(self, password: str) -> bool:
        # Verify that the provided password matches the stored hash.
        return bcrypt.checkpw(password.encode('utf-8'), self.password_hash)
