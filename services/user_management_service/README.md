## Auth Core v5.2
This module handles user authentication, authorization, and role-based access control (RBAC) for IAROS. It employs OAuth 2.1/OpenID Connect standards, supports FIDO2 WebAuthn, and enforces a strict RBAC model with seven permission tiers. Fallback mechanisms ensure that, during OAuth provider outages, cached credentials are used to maintain uninterrupted service.

### Key Capabilities
- **Authentication:**  
  Secure JWT token generation and validation with auto‑rotation.
- **Authorization:**  
  Fine-grained RBAC with just‑in‑time privilege assignment.
- **Fallback Mechanisms:**  
  Caches credentials and enables read‑only mode during service degradation.
- **Compliance:**  
  SOC 2 Type II, ISO 27001, and GDPR Article 32 compliant.

For detailed security architecture, see [Auth_Security_Model_v4.pdf](../../design-docs/auth_security_model_v4.pdf).
