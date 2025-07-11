# User.py
"""
Enterprise User Management Model
-------------------------------
Comprehensive user model with RBAC, audit trails, security features, and enterprise-grade authentication.
Supports multi-factor authentication, session management, privacy controls, and compliance tracking.
"""

import uuid
import bcrypt
import json
from datetime import datetime, timezone, timedelta
from typing import List, Dict, Optional, Enum, Any
from decimal import Decimal
import hashlib
import secrets
from dataclasses import dataclass, field

class UserRole(Enum):
    """System-wide user roles with hierarchical permissions."""
    SUPER_ADMIN = "SUPER_ADMIN"
    SYSTEM_ADMIN = "SYSTEM_ADMIN"
    AIRLINE_ADMIN = "AIRLINE_ADMIN"
    REVENUE_MANAGER = "REVENUE_MANAGER"
    PRICING_ADMIN = "PRICING_ADMIN"
    FORECAST_ANALYST = "FORECAST_ANALYST"
    OPERATIONS_MANAGER = "OPERATIONS_MANAGER"
    CUSTOMER_SERVICE = "CUSTOMER_SERVICE"
    FINANCIAL_ANALYST = "FINANCIAL_ANALYST"
    DATA_ANALYST = "DATA_ANALYST"
    API_USER = "API_USER"
    READONLY_USER = "READONLY_USER"
    GUEST_USER = "GUEST_USER"

class UserStatus(Enum):
    """User account status."""
    ACTIVE = "ACTIVE"
    INACTIVE = "INACTIVE"
    SUSPENDED = "SUSPENDED"
    PENDING_VERIFICATION = "PENDING_VERIFICATION"
    LOCKED = "LOCKED"
    ARCHIVED = "ARCHIVED"

class AuthenticationMethod(Enum):
    """Supported authentication methods."""
    PASSWORD = "PASSWORD"
    BIOMETRIC = "BIOMETRIC"
    SMS_OTP = "SMS_OTP"
    EMAIL_OTP = "EMAIL_OTP"
    TOTP = "TOTP"  # Time-based One-Time Password
    SAML_SSO = "SAML_SSO"
    OAUTH2 = "OAUTH2"
    LDAP = "LDAP"
    CERTIFICATE = "CERTIFICATE"

class SessionType(Enum):
    """Types of user sessions."""
    WEB_BROWSER = "WEB_BROWSER"
    MOBILE_APP = "MOBILE_APP"
    API_TOKEN = "API_TOKEN"
    DESKTOP_APP = "DESKTOP_APP"
    SYSTEM_INTEGRATION = "SYSTEM_INTEGRATION"

class PermissionScope(Enum):
    """Permission scopes for fine-grained access control."""
    GLOBAL = "GLOBAL"
    ORGANIZATION = "ORGANIZATION"
    DEPARTMENT = "DEPARTMENT"
    PROJECT = "PROJECT"
    RESOURCE = "RESOURCE"

@dataclass
class Permission:
    """Individual permission with scope and metadata."""
    permission_id: str = field(default_factory=lambda: str(uuid.uuid4()))
    resource: str = ""  # e.g., "pricing_service", "customer_data"
    action: str = ""    # e.g., "read", "write", "execute", "admin"
    scope: PermissionScope = PermissionScope.RESOURCE
    scope_value: str = ""  # Organization ID, Department ID, etc.
    conditions: Dict[str, Any] = field(default_factory=dict)
    granted_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    granted_by: str = ""
    expires_at: Optional[datetime] = None
    
    def is_valid(self) -> bool:
        """Check if permission is currently valid."""
        if self.expires_at and datetime.now(timezone.utc) > self.expires_at:
            return False
        return True
    
    def to_dict(self) -> Dict:
        return {
            "permission_id": self.permission_id,
            "resource": self.resource,
            "action": self.action,
            "scope": self.scope.value,
            "scope_value": self.scope_value,
            "conditions": self.conditions,
            "granted_at": self.granted_at.isoformat(),
            "granted_by": self.granted_by,
            "expires_at": self.expires_at.isoformat() if self.expires_at else None
        }

@dataclass
class UserSession:
    """User session with security tracking."""
    session_id: str = field(default_factory=lambda: str(uuid.uuid4()))
    user_id: str = ""
    session_type: SessionType = SessionType.WEB_BROWSER
    ip_address: str = ""
    user_agent: str = ""
    location: Dict[str, str] = field(default_factory=dict)
    created_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    last_activity: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    expires_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc) + timedelta(hours=24))
    is_active: bool = True
    security_flags: List[str] = field(default_factory=list)
    device_fingerprint: str = ""
    
    def is_valid(self) -> bool:
        """Check if session is valid and not expired."""
        return (self.is_active and 
                datetime.now(timezone.utc) < self.expires_at)
    
    def extend_session(self, hours: int = 24):
        """Extend session expiration."""
        self.expires_at = datetime.now(timezone.utc) + timedelta(hours=hours)
        self.last_activity = datetime.now(timezone.utc)
    
    def add_security_flag(self, flag: str):
        """Add security warning flag."""
        if flag not in self.security_flags:
            self.security_flags.append(flag)
    
    def to_dict(self) -> Dict:
        return {
            "session_id": self.session_id,
            "user_id": self.user_id,
            "session_type": self.session_type.value,
            "ip_address": self.ip_address,
            "user_agent": self.user_agent,
            "location": self.location,
            "created_at": self.created_at.isoformat(),
            "last_activity": self.last_activity.isoformat(),
            "expires_at": self.expires_at.isoformat(),
            "is_active": self.is_active,
            "security_flags": self.security_flags,
            "device_fingerprint": self.device_fingerprint
        }

@dataclass
class AuditEntry:
    """Audit trail entry for user actions."""
    audit_id: str = field(default_factory=lambda: str(uuid.uuid4()))
    user_id: str = ""
    action: str = ""
    resource: str = ""
    details: Dict[str, Any] = field(default_factory=dict)
    timestamp: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    ip_address: str = ""
    session_id: str = ""
    success: bool = True
    error_message: Optional[str] = None
    risk_score: float = 0.0  # 0.0 = low risk, 1.0 = high risk
    
    def to_dict(self) -> Dict:
        return {
            "audit_id": self.audit_id,
            "user_id": self.user_id,
            "action": self.action,
            "resource": self.resource,
            "details": self.details,
            "timestamp": self.timestamp.isoformat(),
            "ip_address": self.ip_address,
            "session_id": self.session_id,
            "success": self.success,
            "error_message": self.error_message,
            "risk_score": self.risk_score
        }

@dataclass
class MultiFactorAuthConfig:
    """Multi-factor authentication configuration."""
    enabled: bool = False
    primary_method: AuthenticationMethod = AuthenticationMethod.PASSWORD
    secondary_methods: List[AuthenticationMethod] = field(default_factory=list)
    backup_codes: List[str] = field(default_factory=list)
    totp_secret: Optional[str] = None
    phone_number: Optional[str] = None
    recovery_email: Optional[str] = None
    last_used_method: Optional[AuthenticationMethod] = None
    last_used_at: Optional[datetime] = None
    
    def generate_backup_codes(self, count: int = 10) -> List[str]:
        """Generate backup authentication codes."""
        self.backup_codes = [secrets.token_hex(8) for _ in range(count)]
        return self.backup_codes
    
    def use_backup_code(self, code: str) -> bool:
        """Use and consume a backup code."""
        if code in self.backup_codes:
            self.backup_codes.remove(code)
            return True
        return False
    
    def to_dict(self) -> Dict:
        return {
            "enabled": self.enabled,
            "primary_method": self.primary_method.value,
            "secondary_methods": [method.value for method in self.secondary_methods],
            "backup_codes_count": len(self.backup_codes),
            "totp_secret": "***" if self.totp_secret else None,
            "phone_number": self.phone_number,
            "recovery_email": self.recovery_email,
            "last_used_method": self.last_used_method.value if self.last_used_method else None,
            "last_used_at": self.last_used_at.isoformat() if self.last_used_at else None
        }

class User:
    """Enterprise-grade user model with comprehensive security and audit capabilities."""
    
    def __init__(self, email: str, first_name: str, last_name: str,
                 roles: List[UserRole] = None, password: Optional[str] = None,
                 organization_id: Optional[str] = None, department: Optional[str] = None):
        # Core Identity
        self.user_id = str(uuid.uuid4())
        self.email = email.lower().strip()
        self.first_name = first_name.strip()
        self.last_name = last_name.strip()
        self.display_name = f"{first_name} {last_name}"
        self.username = self.email  # Use email as username
        
        # Organization
        self.organization_id = organization_id
        self.department = department
        self.employee_id: Optional[str] = None
        self.manager_id: Optional[str] = None
        self.cost_center: Optional[str] = None
        
        # Role & Permissions
        self.roles = roles or [UserRole.GUEST_USER]
        self.permissions: List[Permission] = []
        self.effective_permissions: Dict[str, List[str]] = {}
        
        # Authentication
        self.password_hash: Optional[bytes] = None
        if password:
            self.password_hash = self._hash_password(password)
        self.password_changed_at = datetime.now(timezone.utc)
        self.password_expires_at: Optional[datetime] = None
        self.failed_login_attempts = 0
        self.last_failed_login: Optional[datetime] = None
        self.mfa_config = MultiFactorAuthConfig()
        
        # Account Status
        self.status = UserStatus.PENDING_VERIFICATION
        self.is_verified = False
        self.verification_token: Optional[str] = None
        self.created_at = datetime.now(timezone.utc)
        self.last_login: Optional[datetime] = None
        self.last_activity: Optional[datetime] = None
        self.locked_until: Optional[datetime] = None
        
        # Security & Privacy
        self.security_questions: List[Dict[str, str]] = []
        self.privacy_settings: Dict[str, bool] = {
            "analytics_tracking": True,
            "marketing_communications": False,
            "data_sharing": False,
            "activity_logging": True
        }
        self.gdpr_consent_given = False
        self.gdpr_consent_date: Optional[datetime] = None
        self.data_retention_period: Optional[datetime] = None
        
        # Sessions & Audit
        self.active_sessions: List[UserSession] = []
        self.audit_trail: List[AuditEntry] = []
        self.login_history: List[Dict[str, Any]] = []
        
        # Profile & Preferences
        self.profile_picture_url: Optional[str] = None
        self.phone_number: Optional[str] = None
        self.timezone = "UTC"
        self.language = "en"
        self.preferences: Dict[str, Any] = {}
        self.emergency_contact: Dict[str, str] = {}
        
        # System Metadata
        self.created_by: Optional[str] = None
        self.updated_at = self.created_at
        self.updated_by: Optional[str] = None
        self.version = 1
        
        # Initialize verification token
        self._generate_verification_token()
        
        # Calculate effective permissions
        self._calculate_effective_permissions()
    
    def _hash_password(self, password: str) -> bytes:
        """Generate secure password hash using bcrypt."""
        return bcrypt.hashpw(password.encode('utf-8'), bcrypt.gensalt(rounds=12))
    
    def check_password(self, password: str) -> bool:
        """Verify password against stored hash."""
        if not self.password_hash:
            return False
        return bcrypt.checkpw(password.encode('utf-8'), self.password_hash)
    
    def set_password(self, new_password: str, changed_by: Optional[str] = None):
        """Set new password with security validation."""
        # Add password strength validation here
        if len(new_password) < 8:
            raise ValueError("Password must be at least 8 characters long")
        
        self.password_hash = self._hash_password(new_password)
        self.password_changed_at = datetime.now(timezone.utc)
        self.failed_login_attempts = 0
        self.last_failed_login = None
        self.updated_at = datetime.now(timezone.utc)
        self.updated_by = changed_by
        self.version += 1
        
        # Audit password change
        self.add_audit_entry("PASSWORD_CHANGED", "user_management", 
                           {"changed_by": changed_by}, success=True)
    
    def _generate_verification_token(self):
        """Generate email verification token."""
        self.verification_token = secrets.token_urlsafe(32)
    
    def verify_email(self) -> bool:
        """Verify user email address."""
        if self.verification_token:
            self.is_verified = True
            self.verification_token = None
            self.status = UserStatus.ACTIVE
            self.updated_at = datetime.now(timezone.utc)
            self.add_audit_entry("EMAIL_VERIFIED", "user_management", {}, success=True)
            return True
        return False
    
    def add_role(self, role: UserRole, granted_by: Optional[str] = None):
        """Add role to user."""
        if role not in self.roles:
            self.roles.append(role)
            self.updated_at = datetime.now(timezone.utc)
            self.updated_by = granted_by
            self._calculate_effective_permissions()
            
            self.add_audit_entry("ROLE_ADDED", "user_management", 
                               {"role": role.value, "granted_by": granted_by}, success=True)
    
    def remove_role(self, role: UserRole, removed_by: Optional[str] = None):
        """Remove role from user."""
        if role in self.roles:
            self.roles.remove(role)
            self.updated_at = datetime.now(timezone.utc)
            self.updated_by = removed_by
            self._calculate_effective_permissions()
            
            self.add_audit_entry("ROLE_REMOVED", "user_management",
                               {"role": role.value, "removed_by": removed_by}, success=True)
    
    def add_permission(self, permission: Permission):
        """Add specific permission to user."""
        self.permissions.append(permission)
        self._calculate_effective_permissions()
        self.add_audit_entry("PERMISSION_GRANTED", "user_management",
                           permission.to_dict(), success=True)
    
    def remove_permission(self, permission_id: str, removed_by: Optional[str] = None):
        """Remove specific permission from user."""
        self.permissions = [p for p in self.permissions if p.permission_id != permission_id]
        self._calculate_effective_permissions()
        self.add_audit_entry("PERMISSION_REVOKED", "user_management",
                           {"permission_id": permission_id, "removed_by": removed_by}, success=True)
    
    def has_permission(self, resource: str, action: str) -> bool:
        """Check if user has specific permission."""
        if resource in self.effective_permissions:
            return action in self.effective_permissions[resource]
        return False
    
    def _calculate_effective_permissions(self):
        """Calculate effective permissions from roles and explicit permissions."""
        self.effective_permissions = {}
        
        # Role-based permissions
        role_permissions = {
            UserRole.SUPER_ADMIN: {"*": ["*"]},
            UserRole.SYSTEM_ADMIN: {"system": ["read", "write", "admin"], "users": ["read", "write"]},
            UserRole.AIRLINE_ADMIN: {"airline": ["read", "write", "admin"], "revenue": ["read", "write"]},
            UserRole.REVENUE_MANAGER: {"revenue": ["read", "write"], "pricing": ["read", "write"], "forecasting": ["read"]},
            UserRole.PRICING_ADMIN: {"pricing": ["read", "write", "admin"], "revenue": ["read"]},
            UserRole.FORECAST_ANALYST: {"forecasting": ["read", "write"], "analytics": ["read", "write"]},
            UserRole.OPERATIONS_MANAGER: {"operations": ["read", "write"], "network": ["read", "write"]},
            UserRole.CUSTOMER_SERVICE: {"customers": ["read", "write"], "orders": ["read", "write"]},
            UserRole.FINANCIAL_ANALYST: {"financial": ["read", "write"], "reports": ["read", "write"]},
            UserRole.DATA_ANALYST: {"analytics": ["read", "write"], "reports": ["read"]},
            UserRole.API_USER: {"api": ["read", "write"]},
            UserRole.READONLY_USER: {"*": ["read"]},
            UserRole.GUEST_USER: {"public": ["read"]}
        }
        
        # Aggregate permissions from roles
        for role in self.roles:
            if role in role_permissions:
                for resource, actions in role_permissions[role].items():
                    if resource not in self.effective_permissions:
                        self.effective_permissions[resource] = set()
                    self.effective_permissions[resource].update(actions)
        
        # Add explicit permissions
        for permission in self.permissions:
            if permission.is_valid():
                if permission.resource not in self.effective_permissions:
                    self.effective_permissions[permission.resource] = set()
                self.effective_permissions[permission.resource].add(permission.action)
        
        # Convert sets to lists
        self.effective_permissions = {k: list(v) for k, v in self.effective_permissions.items()}
    
    def create_session(self, session_type: SessionType, ip_address: str,
                      user_agent: str = "", location: Dict[str, str] = None) -> UserSession:
        """Create new user session."""
        session = UserSession(
            user_id=self.user_id,
            session_type=session_type,
            ip_address=ip_address,
            user_agent=user_agent,
            location=location or {}
        )
        
        # Generate device fingerprint
        fingerprint_data = f"{user_agent}:{ip_address}:{session_type.value}"
        session.device_fingerprint = hashlib.sha256(fingerprint_data.encode()).hexdigest()[:16]
        
        self.active_sessions.append(session)
        self.last_login = datetime.now(timezone.utc)
        self.last_activity = self.last_login
        
        # Security checks
        self._check_session_security(session)
        
        # Add to login history
        self.login_history.append({
            "timestamp": session.created_at.isoformat(),
            "ip_address": ip_address,
            "user_agent": user_agent,
            "session_type": session_type.value,
            "success": True
        })
        
        # Keep only last 100 login records
        if len(self.login_history) > 100:
            self.login_history = self.login_history[-100:]
        
        self.add_audit_entry("SESSION_CREATED", "authentication",
                           session.to_dict(), success=True)
        
        return session
    
    def _check_session_security(self, session: UserSession):
        """Perform security checks on new session."""
        # Check for suspicious activity patterns
        recent_sessions = [s for s in self.active_sessions 
                          if (datetime.now(timezone.utc) - s.created_at).total_seconds() < 3600]
        
        # Multiple sessions from different IPs
        unique_ips = set(s.ip_address for s in recent_sessions)
        if len(unique_ips) > 3:
            session.add_security_flag("MULTIPLE_IPS")
        
        # Rapid session creation
        if len(recent_sessions) > 5:
            session.add_security_flag("RAPID_SESSIONS")
        
        # Unusual user agent
        if "bot" in session.user_agent.lower() or "crawler" in session.user_agent.lower():
            session.add_security_flag("SUSPICIOUS_USER_AGENT")
    
    def end_session(self, session_id: str):
        """End user session."""
        for session in self.active_sessions:
            if session.session_id == session_id:
                session.is_active = False
                self.add_audit_entry("SESSION_ENDED", "authentication",
                                   {"session_id": session_id}, success=True)
                break
    
    def end_all_sessions(self):
        """End all active user sessions."""
        for session in self.active_sessions:
            session.is_active = False
        self.add_audit_entry("ALL_SESSIONS_ENDED", "authentication", {}, success=True)
    
    def record_failed_login(self, ip_address: str, reason: str):
        """Record failed login attempt."""
        self.failed_login_attempts += 1
        self.last_failed_login = datetime.now(timezone.utc)
        
        # Lock account after 5 failed attempts
        if self.failed_login_attempts >= 5:
            self.locked_until = datetime.now(timezone.utc) + timedelta(minutes=30)
            self.status = UserStatus.LOCKED
        
        # Add to login history
        self.login_history.append({
            "timestamp": self.last_failed_login.isoformat(),
            "ip_address": ip_address,
            "success": False,
            "reason": reason
        })
        
        self.add_audit_entry("LOGIN_FAILED", "authentication",
                           {"ip_address": ip_address, "reason": reason, 
                            "attempts": self.failed_login_attempts}, success=False)
    
    def unlock_account(self, unlocked_by: Optional[str] = None):
        """Unlock user account."""
        self.status = UserStatus.ACTIVE
        self.locked_until = None
        self.failed_login_attempts = 0
        self.last_failed_login = None
        self.updated_at = datetime.now(timezone.utc)
        self.updated_by = unlocked_by
        
        self.add_audit_entry("ACCOUNT_UNLOCKED", "user_management",
                           {"unlocked_by": unlocked_by}, success=True)
    
    def is_locked(self) -> bool:
        """Check if account is currently locked."""
        if self.status == UserStatus.LOCKED:
            if self.locked_until and datetime.now(timezone.utc) > self.locked_until:
                # Auto-unlock expired lock
                self.unlock_account("SYSTEM_AUTO_UNLOCK")
                return False
            return True
        return False
    
    def add_audit_entry(self, action: str, resource: str, details: Dict[str, Any],
                       success: bool = True, ip_address: str = "", session_id: str = ""):
        """Add entry to user audit trail."""
        audit_entry = AuditEntry(
            user_id=self.user_id,
            action=action,
            resource=resource,
            details=details,
            success=success,
            ip_address=ip_address,
            session_id=session_id
        )
        
        self.audit_trail.append(audit_entry)
        
        # Keep only last 1000 audit entries
        if len(self.audit_trail) > 1000:
            self.audit_trail = self.audit_trail[-1000:]
    
    def update_privacy_settings(self, settings: Dict[str, bool]):
        """Update user privacy preferences."""
        self.privacy_settings.update(settings)
        self.updated_at = datetime.now(timezone.utc)
        
        self.add_audit_entry("PRIVACY_SETTINGS_UPDATED", "user_management",
                           {"settings": settings}, success=True)
    
    def give_gdpr_consent(self):
        """Record GDPR consent."""
        self.gdpr_consent_given = True
        self.gdpr_consent_date = datetime.now(timezone.utc)
        self.updated_at = datetime.now(timezone.utc)
        
        self.add_audit_entry("GDPR_CONSENT_GIVEN", "privacy_compliance", {}, success=True)
    
    def revoke_gdpr_consent(self):
        """Revoke GDPR consent."""
        self.gdpr_consent_given = False
        self.gdpr_consent_date = None
        self.updated_at = datetime.now(timezone.utc)
        
        self.add_audit_entry("GDPR_CONSENT_REVOKED", "privacy_compliance", {}, success=True)
    
    def setup_mfa(self, method: AuthenticationMethod, **kwargs):
        """Setup multi-factor authentication."""
        self.mfa_config.enabled = True
        self.mfa_config.secondary_methods.append(method)
        
        if method == AuthenticationMethod.TOTP:
            self.mfa_config.totp_secret = kwargs.get('secret')
        elif method == AuthenticationMethod.SMS_OTP:
            self.mfa_config.phone_number = kwargs.get('phone_number')
        elif method == AuthenticationMethod.EMAIL_OTP:
            self.mfa_config.recovery_email = kwargs.get('recovery_email')
        
        self.mfa_config.generate_backup_codes()
        self.updated_at = datetime.now(timezone.utc)
        
        self.add_audit_entry("MFA_ENABLED", "security",
                           {"method": method.value}, success=True)
    
    def disable_mfa(self):
        """Disable multi-factor authentication."""
        self.mfa_config.enabled = False
        self.mfa_config.secondary_methods = []
        self.mfa_config.backup_codes = []
        self.mfa_config.totp_secret = None
        self.updated_at = datetime.now(timezone.utc)
        
        self.add_audit_entry("MFA_DISABLED", "security", {}, success=True)
    
    def get_risk_score(self) -> float:
        """Calculate user risk score based on behavior and security indicators."""
        risk_score = 0.0
        
        # Failed login attempts
        if self.failed_login_attempts > 0:
            risk_score += min(0.3, self.failed_login_attempts * 0.1)
        
        # Recent security flags
        recent_sessions = [s for s in self.active_sessions 
                          if (datetime.now(timezone.utc) - s.created_at).total_seconds() < 86400]
        for session in recent_sessions:
            risk_score += len(session.security_flags) * 0.1
        
        # Account status
        if self.status in [UserStatus.SUSPENDED, UserStatus.LOCKED]:
            risk_score += 0.5
        
        # Missing MFA
        if not self.mfa_config.enabled:
            risk_score += 0.2
        
        # Unverified email
        if not self.is_verified:
            risk_score += 0.1
        
        return min(1.0, risk_score)
    
    def cleanup_expired_sessions(self):
        """Remove expired sessions."""
        current_time = datetime.now(timezone.utc)
        self.active_sessions = [s for s in self.active_sessions 
                               if s.is_valid() and s.expires_at > current_time]
    
    def to_dict(self, include_sensitive: bool = False) -> Dict:
        """Convert user to dictionary representation."""
        user_dict = {
            "user_id": self.user_id,
            "email": self.email,
            "first_name": self.first_name,
            "last_name": self.last_name,
            "display_name": self.display_name,
            "username": self.username,
            "organization_id": self.organization_id,
            "department": self.department,
            "employee_id": self.employee_id,
            "manager_id": self.manager_id,
            "roles": [role.value for role in self.roles],
            "status": self.status.value,
            "is_verified": self.is_verified,
            "created_at": self.created_at.isoformat(),
            "last_login": self.last_login.isoformat() if self.last_login else None,
            "last_activity": self.last_activity.isoformat() if self.last_activity else None,
            "profile_picture_url": self.profile_picture_url,
            "phone_number": self.phone_number,
            "timezone": self.timezone,
            "language": self.language,
            "preferences": self.preferences,
            "privacy_settings": self.privacy_settings,
            "gdpr_consent_given": self.gdpr_consent_given,
            "gdpr_consent_date": self.gdpr_consent_date.isoformat() if self.gdpr_consent_date else None,
            "mfa_enabled": self.mfa_config.enabled,
            "risk_score": self.get_risk_score(),
            "active_sessions_count": len([s for s in self.active_sessions if s.is_valid()]),
            "version": self.version,
            "updated_at": self.updated_at.isoformat()
        }
        
        if include_sensitive:
            user_dict.update({
                "effective_permissions": self.effective_permissions,
                "failed_login_attempts": self.failed_login_attempts,
                "locked_until": self.locked_until.isoformat() if self.locked_until else None,
                "password_changed_at": self.password_changed_at.isoformat(),
                "mfa_config": self.mfa_config.to_dict(),
                "active_sessions": [s.to_dict() for s in self.active_sessions if s.is_valid()],
                "recent_audit_entries": [e.to_dict() for e in self.audit_trail[-10:]],
                "login_history": self.login_history[-10:]
            })
        
        return user_dict
    
    def validate(self) -> bool:
        """Validate user data integrity."""
        # Email validation
        if not self.email or "@" not in self.email:
            return False
        
        # Name validation
        if not self.first_name or not self.last_name:
            return False
        
        # Role validation
        if not self.roles:
            return False
        
        # Status validation
        if self.status not in UserStatus:
            return False
        
        return True
    
    def __repr__(self) -> str:
        return f"User(id='{self.user_id}', email='{self.email}', roles={[r.value for r in self.roles]}, status='{self.status.value}')"
