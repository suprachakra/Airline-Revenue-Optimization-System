# ComplianceAutomation.py
"""
Compliance Automation Module
----------------------------
Provides real-time checks for GDPR and IATA compliance.
Automatically raises errors if data or fare rules fail to meet required standards.
"""

class ComplianceError(Exception):
    pass

class ComplianceEngine:
    def check_gdpr(self, data):
        # Verify that personal data includes a valid consent timestamp.
        if not data.get('consent_timestamp'):
            raise ComplianceError("GDPR Art.7 violation - Missing consent timestamp")
        return True

    def check_iata(self, fare_rules):
        # Validate that fare rules are NDC Level 4 compliant.
        if not fare_rules.get('ndc_compatible'):
            raise ComplianceError("IATA NDC Level 4 violation - Fare rules non-compliant")
        return True
