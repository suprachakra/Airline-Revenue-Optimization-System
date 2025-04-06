# roles.py
"""
User Roles Enumeration
----------------------
Defines user roles and their permissions for RBAC across IAROS.
"""

ROLES = {
    "pricing_admin": "Can modify pricing rules and view sensitive data",
    "revenue_manager": "Can access revenue analytics and override pricing under emergency",
    "pricing_analyst": "Can view pricing scenarios and generate reports",
    "user": "General user with limited access"
}
