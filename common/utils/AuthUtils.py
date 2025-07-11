# AuthUtils.py
"""
Enterprise Authentication & Security Utilities
---------------------------------------------
Comprehensive authentication utilities with OAuth2, SAML SSO, JWT management, 
certificate handling, multi-factor authentication, and enterprise security compliance.
"""

import jwt
import hashlib
import secrets
import base64
import os
import time
import json
from datetime import datetime, timedelta, timezone
from typing import Dict, List, Optional, Any, Tuple, Union
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import rsa, padding
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.backends import default_backend
from cryptography.x509 import load_pem_x509_certificate
import hmac
from urllib.parse import urlencode, parse_qs
from enum import Enum
import logging

class TokenType(Enum):
    """Token types for different authentication mechanisms."""
    ACCESS_TOKEN = "ACCESS_TOKEN"
    REFRESH_TOKEN = "REFRESH_TOKEN"
    ID_TOKEN = "ID_TOKEN"
    API_KEY = "API_KEY"
    SESSION_TOKEN = "SESSION_TOKEN"
    RESET_TOKEN = "RESET_TOKEN"
    VERIFICATION_TOKEN = "VERIFICATION_TOKEN"

class AuthProvider(Enum):
    """Supported authentication providers."""
    INTERNAL = "INTERNAL"
    OAUTH2_GOOGLE = "OAUTH2_GOOGLE"
    OAUTH2_MICROSOFT = "OAUTH2_MICROSOFT"
    SAML_OKTA = "SAML_OKTA"
    SAML_ADFS = "SAML_ADFS"
    LDAP = "LDAP"
    CERTIFICATE = "CERTIFICATE"

class SecurityLevel(Enum):
    """Security levels for different operations."""
    PUBLIC = "PUBLIC"
    BASIC = "BASIC"
    ELEVATED = "ELEVATED"
    ADMIN = "ADMIN"
    SUPER_ADMIN = "SUPER_ADMIN"

class AuthUtils:
    """Enterprise-grade authentication and security utilities."""
    
    # Configuration - Should be loaded from secure environment variables
    _config = {
        "JWT_SECRET_KEY": os.getenv("JWT_SECRET_KEY", "CHANGE_THIS_IN_PRODUCTION"),
        "JWT_ALGORITHM": "HS256",
        "JWT_ACCESS_TOKEN_EXPIRE_MINUTES": 60,
        "JWT_REFRESH_TOKEN_EXPIRE_DAYS": 7,
        "ENCRYPTION_KEY": os.getenv("ENCRYPTION_KEY", "").encode() or secrets.token_bytes(32),
        "OAUTH2_CLIENT_ID": os.getenv("OAUTH2_CLIENT_ID", ""),
        "OAUTH2_CLIENT_SECRET": os.getenv("OAUTH2_CLIENT_SECRET", ""),
        "SAML_CERTIFICATE_PATH": os.getenv("SAML_CERTIFICATE_PATH", ""),
        "PASSWORD_MIN_LENGTH": 8,
        "PASSWORD_REQUIRE_SPECIAL": True,
        "PASSWORD_REQUIRE_NUMBERS": True,
        "PASSWORD_REQUIRE_UPPERCASE": True,
        "MFA_ISSUER": "IAROS Airlines",
        "RATE_LIMIT_ATTEMPTS": 5,
        "RATE_LIMIT_WINDOW_MINUTES": 15
    }
    
    _failed_attempts: Dict[str, List[datetime]] = {}
    _blocked_ips: Dict[str, datetime] = {}
    
    @classmethod
    def configure(cls, config: Dict[str, Any]):
        """Update configuration with new values."""
        cls._config.update(config)
    
    @classmethod
    def generate_secure_token(cls, length: int = 32, token_type: TokenType = TokenType.API_KEY) -> str:
        """Generate cryptographically secure random token."""
        token = secrets.token_urlsafe(length)
        
        # Add timestamp and type prefix for easier identification
        timestamp = int(time.time())
        prefix = token_type.value[:3].lower()
        
        return f"{prefix}_{timestamp}_{token}"
    
    @classmethod
    def hash_password(cls, password: str, salt: Optional[bytes] = None) -> Tuple[str, str]:
        """Hash password using PBKDF2 with SHA-256."""
        if salt is None:
            salt = secrets.token_bytes(32)
        
        # Use PBKDF2 with 100,000 iterations for strong security
        key = hashlib.pbkdf2_hmac('sha256', password.encode('utf-8'), salt, 100000)
        
        # Return base64 encoded hash and salt
        return base64.b64encode(key).decode('utf-8'), base64.b64encode(salt).decode('utf-8')
    
    @classmethod
    def verify_password(cls, password: str, hashed_password: str, salt: str) -> bool:
        """Verify password against stored hash."""
        try:
            salt_bytes = base64.b64decode(salt.encode('utf-8'))
            stored_hash = base64.b64decode(hashed_password.encode('utf-8'))
            
            # Compute hash of provided password
            key = hashlib.pbkdf2_hmac('sha256', password.encode('utf-8'), salt_bytes, 100000)
            
            # Use constant time comparison to prevent timing attacks
            return hmac.compare_digest(key, stored_hash)
        except Exception:
            return False
    
    @classmethod
    def validate_password_strength(cls, password: str) -> Dict[str, Any]:
        """Validate password strength according to security policy."""
        validation = {
            "valid": True,
            "errors": [],
            "score": 0,
            "suggestions": []
        }
        
        # Length check
        if len(password) < cls._config["PASSWORD_MIN_LENGTH"]:
            validation["valid"] = False
            validation["errors"].append(f"Password must be at least {cls._config['PASSWORD_MIN_LENGTH']} characters long")
        else:
            validation["score"] += 20
        
        # Character requirements
        has_upper = any(c.isupper() for c in password)
        has_lower = any(c.islower() for c in password)
        has_digit = any(c.isdigit() for c in password)
        has_special = any(c in "!@#$%^&*()_+-=[]{}|;:,.<>?" for c in password)
        
        if cls._config["PASSWORD_REQUIRE_UPPERCASE"] and not has_upper:
            validation["valid"] = False
            validation["errors"].append("Password must contain at least one uppercase letter")
        elif has_upper:
            validation["score"] += 15
        
        if not has_lower:
            validation["valid"] = False
            validation["errors"].append("Password must contain at least one lowercase letter")
        else:
            validation["score"] += 15
        
        if cls._config["PASSWORD_REQUIRE_NUMBERS"] and not has_digit:
            validation["valid"] = False
            validation["errors"].append("Password must contain at least one number")
        elif has_digit:
            validation["score"] += 15
        
        if cls._config["PASSWORD_REQUIRE_SPECIAL"] and not has_special:
            validation["valid"] = False
            validation["errors"].append("Password must contain at least one special character")
        elif has_special:
            validation["score"] += 15
        
        # Additional security checks
        if len(password) >= 12:
            validation["score"] += 10
        if len(set(password)) >= len(password) * 0.7:  # Character diversity
            validation["score"] += 10
        
        # Common password check (simplified)
        common_passwords = ["password", "123456", "qwerty", "admin", "welcome"]
        if password.lower() in common_passwords:
            validation["valid"] = False
            validation["errors"].append("Password is too common")
            validation["score"] = max(0, validation["score"] - 30)
        
        return validation
    
    @classmethod
    def generate_jwt_token(cls, payload: Dict[str, Any], token_type: TokenType = TokenType.ACCESS_TOKEN,
                          expires_delta: Optional[timedelta] = None) -> str:
        """Generate JWT token with specified payload and expiration."""
        if expires_delta is None:
            if token_type == TokenType.REFRESH_TOKEN:
                expires_delta = timedelta(days=cls._config["JWT_REFRESH_TOKEN_EXPIRE_DAYS"])
            else:
                expires_delta = timedelta(minutes=cls._config["JWT_ACCESS_TOKEN_EXPIRE_MINUTES"])
        
        # Add standard claims
        now = datetime.now(timezone.utc)
        payload.update({
            "iat": now,
            "exp": now + expires_delta,
            "jti": secrets.token_hex(16),  # JWT ID for tracking
            "token_type": token_type.value,
            "issuer": "IAROS_AUTH_SERVICE"
        })
        
        return jwt.encode(payload, cls._config["JWT_SECRET_KEY"], algorithm=cls._config["JWT_ALGORITHM"])
    
    @classmethod
    def validate_jwt_token(cls, token: str, expected_type: Optional[TokenType] = None) -> Dict[str, Any]:
        """Validate JWT token and return payload."""
        try:
            payload = jwt.decode(token, cls._config["JWT_SECRET_KEY"], 
                               algorithms=[cls._config["JWT_ALGORITHM"]])
            
            # Verify token type if specified
            if expected_type and payload.get("token_type") != expected_type.value:
                raise jwt.InvalidTokenError("Invalid token type")
            
            return {
                "valid": True,
                "payload": payload,
                "error": None
            }
            
        except jwt.ExpiredSignatureError:
            return {"valid": False, "payload": None, "error": "Token expired"}
        except jwt.InvalidTokenError as e:
            return {"valid": False, "payload": None, "error": f"Invalid token: {str(e)}"}
        except Exception as e:
            return {"valid": False, "payload": None, "error": f"Token validation error: {str(e)}"}
    
    @classmethod
    def encrypt_data(cls, data: str) -> str:
        """Encrypt sensitive data using AES-256-GCM."""
        # Generate random nonce
        nonce = secrets.token_bytes(12)
        
        # Create cipher
        cipher = Cipher(algorithms.AES(cls._config["ENCRYPTION_KEY"]), 
                       modes.GCM(nonce), backend=default_backend())
        encryptor = cipher.encryptor()
        
        # Encrypt data
        ciphertext = encryptor.update(data.encode('utf-8')) + encryptor.finalize()
        
        # Combine nonce, ciphertext, and auth tag
        encrypted_data = nonce + ciphertext + encryptor.tag
        
        return base64.b64encode(encrypted_data).decode('utf-8')
    
    @classmethod
    def decrypt_data(cls, encrypted_data: str) -> str:
        """Decrypt data encrypted with encrypt_data."""
        try:
            # Decode from base64
            data = base64.b64decode(encrypted_data.encode('utf-8'))
            
            # Extract components
            nonce = data[:12]
            tag = data[-16:]
            ciphertext = data[12:-16]
            
            # Create cipher
            cipher = Cipher(algorithms.AES(cls._config["ENCRYPTION_KEY"]), 
                           modes.GCM(nonce, tag), backend=default_backend())
            decryptor = cipher.decryptor()
            
            # Decrypt data
            plaintext = decryptor.update(ciphertext) + decryptor.finalize()
            
            return plaintext.decode('utf-8')
            
        except Exception as e:
            raise ValueError(f"Decryption failed: {str(e)}")
    
    @classmethod
    def generate_oauth2_authorization_url(cls, provider: AuthProvider, redirect_uri: str,
                                        state: Optional[str] = None, scopes: List[str] = None) -> str:
        """Generate OAuth2 authorization URL."""
        if state is None:
            state = secrets.token_urlsafe(32)
        
        if scopes is None:
            scopes = ["openid", "profile", "email"]
        
        # Provider-specific configuration
        provider_config = {
            AuthProvider.OAUTH2_GOOGLE: {
                "auth_url": "https://accounts.google.com/o/oauth2/v2/auth",
                "default_scopes": ["openid", "profile", "email"]
            },
            AuthProvider.OAUTH2_MICROSOFT: {
                "auth_url": "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
                "default_scopes": ["openid", "profile", "email"]
            }
        }
        
        if provider not in provider_config:
            raise ValueError(f"Unsupported OAuth2 provider: {provider}")
        
        config = provider_config[provider]
        
        params = {
            "client_id": cls._config["OAUTH2_CLIENT_ID"],
            "response_type": "code",
            "redirect_uri": redirect_uri,
            "scope": " ".join(scopes),
            "state": state
        }
        
        return f"{config['auth_url']}?{urlencode(params)}"
    
    @classmethod
    def validate_oauth2_callback(cls, provider: AuthProvider, code: str, 
                                redirect_uri: str) -> Dict[str, Any]:
        """Validate OAuth2 callback and exchange code for tokens."""
        # This would typically make HTTP requests to the provider's token endpoint
        # For demo purposes, returning a mock response
        return {
            "valid": True,
            "access_token": cls.generate_secure_token(32, TokenType.ACCESS_TOKEN),
            "refresh_token": cls.generate_secure_token(32, TokenType.REFRESH_TOKEN),
            "id_token": cls.generate_jwt_token({"sub": "user123", "email": "user@example.com"}),
            "expires_in": 3600,
            "token_type": "Bearer"
        }
    
    @classmethod
    def generate_totp_secret(cls) -> str:
        """Generate TOTP secret for two-factor authentication."""
        return base64.b32encode(secrets.token_bytes(20)).decode('utf-8')
    
    @classmethod
    def generate_totp_qr_url(cls, secret: str, user_email: str) -> str:
        """Generate TOTP QR code URL for authenticator apps."""
        issuer = cls._config["MFA_ISSUER"]
        return f"otpauth://totp/{issuer}:{user_email}?secret={secret}&issuer={issuer}&algorithm=SHA1&digits=6&period=30"
    
    @classmethod
    def verify_totp_code(cls, secret: str, code: str, window: int = 1) -> bool:
        """Verify TOTP code with time window tolerance."""
        import time
        import base64
        import struct
        
        try:
            # Decode secret
            key = base64.b32decode(secret)
            
            # Get current time counter
            time_counter = int(time.time()) // 30
            
            # Check current time and window
            for i in range(-window, window + 1):
                counter = time_counter + i
                
                # Generate HOTP value
                counter_bytes = struct.pack(">Q", counter)
                hmac_hash = hmac.new(key, counter_bytes, hashlib.sha1).digest()
                
                # Dynamic truncation
                offset = hmac_hash[-1] & 0x0F
                truncated = struct.unpack(">I", hmac_hash[offset:offset + 4])[0]
                truncated &= 0x7FFFFFFF
                
                # Generate 6-digit code
                totp_code = str(truncated % 1000000).zfill(6)
                
                if hmac.compare_digest(totp_code, code):
                    return True
            
            return False
            
        except Exception:
            return False
    
    @classmethod
    def check_rate_limit(cls, identifier: str, ip_address: str = "") -> Dict[str, Any]:
        """Check rate limiting for authentication attempts."""
        now = datetime.now(timezone.utc)
        window_start = now - timedelta(minutes=cls._config["RATE_LIMIT_WINDOW_MINUTES"])
        
        # Clean old attempts
        if identifier in cls._failed_attempts:
            cls._failed_attempts[identifier] = [
                attempt for attempt in cls._failed_attempts[identifier] 
                if attempt > window_start
            ]
        
        # Check if IP is blocked
        if ip_address in cls._blocked_ips:
            if cls._blocked_ips[ip_address] > now:
                return {
                    "allowed": False,
                    "reason": "IP blocked",
                    "retry_after": (cls._blocked_ips[ip_address] - now).total_seconds()
                }
            else:
                del cls._blocked_ips[ip_address]
        
        # Check rate limit
        attempts = len(cls._failed_attempts.get(identifier, []))
        
        if attempts >= cls._config["RATE_LIMIT_ATTEMPTS"]:
            # Block for increasing duration based on attempts
            block_duration = min(2 ** (attempts - cls._config["RATE_LIMIT_ATTEMPTS"]), 3600)  # Max 1 hour
            cls._blocked_ips[ip_address] = now + timedelta(seconds=block_duration)
            
            return {
                "allowed": False,
                "reason": "Rate limit exceeded",
                "attempts": attempts,
                "retry_after": block_duration
            }
        
        return {
            "allowed": True,
            "attempts": attempts,
            "remaining": cls._config["RATE_LIMIT_ATTEMPTS"] - attempts
        }
    
    @classmethod
    def record_failed_attempt(cls, identifier: str):
        """Record failed authentication attempt."""
        now = datetime.now(timezone.utc)
        
        if identifier not in cls._failed_attempts:
            cls._failed_attempts[identifier] = []
        
        cls._failed_attempts[identifier].append(now)
    
    @classmethod
    def clear_failed_attempts(cls, identifier: str):
        """Clear failed attempts for successful authentication."""
        if identifier in cls._failed_attempts:
            del cls._failed_attempts[identifier]
    
    @classmethod
    def verify_certificate(cls, certificate_pem: str, trusted_cas: List[str] = None) -> Dict[str, Any]:
        """Verify X.509 certificate for certificate-based authentication."""
        try:
            # Load certificate
            cert = load_pem_x509_certificate(certificate_pem.encode(), default_backend())
            
            # Basic validation
            now = datetime.now(timezone.utc)
            
            validation = {
                "valid": True,
                "errors": [],
                "subject": cert.subject.rfc4514_string(),
                "issuer": cert.issuer.rfc4514_string(),
                "serial_number": str(cert.serial_number),
                "not_before": cert.not_valid_before,
                "not_after": cert.not_valid_after
            }
            
            # Check validity period
            if now < cert.not_valid_before:
                validation["valid"] = False
                validation["errors"].append("Certificate not yet valid")
            
            if now > cert.not_valid_after:
                validation["valid"] = False
                validation["errors"].append("Certificate expired")
            
            # Additional validation against trusted CAs would go here
            if trusted_cas:
                # Simplified - in production, verify full certificate chain
                pass
            
            return validation
            
        except Exception as e:
            return {
                "valid": False,
                "errors": [f"Certificate validation error: {str(e)}"],
                "subject": None,
                "issuer": None
            }
    
    @classmethod
    def generate_api_key(cls, user_id: str, permissions: List[str], 
                        expires_in_days: Optional[int] = None) -> Dict[str, Any]:
        """Generate API key with embedded permissions and expiration."""
        # Create API key payload
        payload = {
            "user_id": user_id,
            "permissions": permissions,
            "key_type": "API_KEY",
            "created_at": datetime.now(timezone.utc).isoformat()
        }
        
        if expires_in_days:
            payload["expires_at"] = (datetime.now(timezone.utc) + 
                                   timedelta(days=expires_in_days)).isoformat()
        
        # Generate secure API key
        api_key = cls.generate_secure_token(48, TokenType.API_KEY)
        
        # Encrypt payload and embed in key
        encrypted_payload = cls.encrypt_data(json.dumps(payload))
        
        return {
            "api_key": api_key,
            "payload": payload,
            "encrypted_payload": encrypted_payload,
            "created_at": payload["created_at"]
        }
    
    @classmethod
    def validate_api_key(cls, api_key: str, encrypted_payload: str) -> Dict[str, Any]:
        """Validate API key and return associated permissions."""
        try:
            # Decrypt payload
            payload_json = cls.decrypt_data(encrypted_payload)
            payload = json.loads(payload_json)
            
            # Check expiration
            if "expires_at" in payload:
                expires_at = datetime.fromisoformat(payload["expires_at"])
                if datetime.now(timezone.utc) > expires_at:
                    return {"valid": False, "error": "API key expired", "payload": None}
            
            return {
                "valid": True,
                "payload": payload,
                "user_id": payload["user_id"],
                "permissions": payload["permissions"]
            }
            
        except Exception as e:
            return {"valid": False, "error": f"API key validation failed: {str(e)}", "payload": None}
    
    @classmethod
    def create_secure_session(cls, user_id: str, user_data: Dict[str, Any], 
                            ip_address: str, user_agent: str) -> Dict[str, Any]:
        """Create secure session with comprehensive tracking."""
        session_id = secrets.token_urlsafe(32)
        
        session_data = {
            "session_id": session_id,
            "user_id": user_id,
            "user_data": user_data,
            "ip_address": ip_address,
            "user_agent": user_agent,
            "created_at": datetime.now(timezone.utc).isoformat(),
            "last_activity": datetime.now(timezone.utc).isoformat(),
            "security_flags": []
        }
        
        # Generate device fingerprint
        fingerprint_data = f"{user_agent}:{ip_address}:{user_id}"
        device_fingerprint = hashlib.sha256(fingerprint_data.encode()).hexdigest()[:16]
        session_data["device_fingerprint"] = device_fingerprint
        
        # Create encrypted session token
        session_token = cls.encrypt_data(json.dumps(session_data))
        
        return {
            "session_id": session_id,
            "session_token": session_token,
            "device_fingerprint": device_fingerprint,
            "expires_at": (datetime.now(timezone.utc) + timedelta(hours=24)).isoformat()
        }
    
    @classmethod
    def get_security_headers(cls) -> Dict[str, str]:
        """Get recommended security headers for HTTP responses."""
        return {
            "X-Content-Type-Options": "nosniff",
            "X-Frame-Options": "DENY",
            "X-XSS-Protection": "1; mode=block",
            "Strict-Transport-Security": "max-age=31536000; includeSubDomains",
            "Content-Security-Policy": "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'",
            "Referrer-Policy": "strict-origin-when-cross-origin",
            "Permissions-Policy": "geolocation=(), microphone=(), camera=()",
            "Cache-Control": "no-cache, no-store, must-revalidate",
            "Pragma": "no-cache",
            "Expires": "0"
        }
    
    @classmethod
    def audit_log_authentication(cls, event: str, user_id: str, ip_address: str,
                                user_agent: str, success: bool, details: Dict[str, Any] = None):
        """Log authentication events for security auditing."""
        log_entry = {
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "event": event,
            "user_id": user_id,
            "ip_address": ip_address,
            "user_agent": user_agent,
            "success": success,
            "details": details or {}
        }
        
        # In production, this would write to a secure audit log
        logging.info(f"AUTH_AUDIT: {json.dumps(log_entry)}")
    
    @classmethod
    def cleanup_expired_data(cls):
        """Clean up expired rate limiting and security data."""
        now = datetime.now(timezone.utc)
        window_start = now - timedelta(minutes=cls._config["RATE_LIMIT_WINDOW_MINUTES"])
        
        # Clean expired failed attempts
        for identifier in list(cls._failed_attempts.keys()):
            cls._failed_attempts[identifier] = [
                attempt for attempt in cls._failed_attempts[identifier] 
                if attempt > window_start
            ]
            if not cls._failed_attempts[identifier]:
                del cls._failed_attempts[identifier]
        
        # Clean expired IP blocks
        for ip in list(cls._blocked_ips.keys()):
            if cls._blocked_ips[ip] <= now:
                del cls._blocked_ips[ip]

# Legacy functions for backward compatibility
def generate_token(user_id: int, roles: list, expires_in: int = 3600) -> str:
    """Legacy function - use AuthUtils.generate_jwt_token instead."""
    payload = {
        "user_id": user_id,
        "roles": roles
    }
    return AuthUtils.generate_jwt_token(payload, TokenType.ACCESS_TOKEN, 
                                      timedelta(seconds=expires_in))

def validate_token(token: str) -> dict:
    """Legacy function - use AuthUtils.validate_jwt_token instead."""
    result = AuthUtils.validate_jwt_token(token)
    if result["valid"]:
        return result["payload"]
    else:
        raise Exception(result["error"])
