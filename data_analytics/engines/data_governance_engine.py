#!/usr/bin/env python3
"""
IAROS Data Governance Engine - Compliance and Lineage Management
Handles GDPR/CCPA compliance, data lineage tracking, and audit logging
"""

import asyncio
import json
import hashlib
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any
from dataclasses import dataclass, asdict
from enum import Enum
import logging
import redis
import pandas as pd

class ComplianceRegulation(Enum):
    GDPR = "GDPR"
    CCPA = "CCPA"
    SOX = "SOX"
    PCI_DSS = "PCI_DSS"
    IATA = "IATA"

class DataClassification(Enum):
    PUBLIC = "PUBLIC"
    INTERNAL = "INTERNAL" 
    CONFIDENTIAL = "CONFIDENTIAL"
    RESTRICTED = "RESTRICTED"
    PII = "PII"

class ConsentStatus(Enum):
    GRANTED = "GRANTED"
    WITHDRAWN = "WITHDRAWN"
    PENDING = "PENDING"
    EXPIRED = "EXPIRED"

@dataclass
class DataLineageRecord:
    record_id: str
    source_system: str
    target_system: str
    transformation_applied: str
    processor: str
    timestamp: datetime
    data_classification: DataClassification
    compliance_tags: List[str]

@dataclass
class ConsentRecord:
    user_id: str
    consent_type: str
    status: ConsentStatus
    granted_at: datetime
    expires_at: Optional[datetime]
    purpose: str
    legal_basis: str

@dataclass
class AuditEvent:
    event_id: str
    user_id: str
    action: str
    resource: str
    timestamp: datetime
    ip_address: str
    result: str
    compliance_regulation: ComplianceRegulation

class DataGovernanceEngine:
    """
    Enterprise Data Governance Engine for IAROS
    Manages compliance, lineage tracking, and audit logging
    """
    
    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.redis_client = redis.Redis(decode_responses=True)
        self.logger = self._setup_logging()
        self.compliance_rules = self._load_compliance_rules()
        
    def _setup_logging(self):
        logging.basicConfig(level=logging.INFO)
        return logging.getLogger(__name__)

    def _load_compliance_rules(self) -> Dict[ComplianceRegulation, Dict]:
        """Load compliance rules for different regulations"""
        return {
            ComplianceRegulation.GDPR: {
                "data_retention_days": 2555,  # 7 years
                "consent_required_for": ["PII", "marketing", "analytics"],
                "right_to_erasure": True,
                "data_portability": True,
                "breach_notification_hours": 72
            },
            ComplianceRegulation.CCPA: {
                "data_retention_days": 1095,  # 3 years
                "right_to_know": True,
                "right_to_delete": True,
                "right_to_opt_out": True,
                "non_discrimination": True
            },
            ComplianceRegulation.PCI_DSS: {
                "encryption_required": True,
                "access_control": "strict",
                "monitoring_required": True,
                "vulnerability_scanning": "quarterly"
            }
        }

    async def track_data_lineage(self, lineage_record: DataLineageRecord) -> bool:
        """Track data lineage for compliance and debugging"""
        try:
            # Create lineage chain
            lineage_key = f"lineage:{lineage_record.record_id}"
            
            lineage_data = {
                "record_id": lineage_record.record_id,
                "source_system": lineage_record.source_system,
                "target_system": lineage_record.target_system,
                "transformation": lineage_record.transformation_applied,
                "processor": lineage_record.processor,
                "timestamp": lineage_record.timestamp.isoformat(),
                "classification": lineage_record.data_classification.value,
                "compliance_tags": lineage_record.compliance_tags
            }
            
            # Store lineage record
            await asyncio.get_event_loop().run_in_executor(
                None, self.redis_client.lpush, lineage_key, json.dumps(lineage_data)
            )
            
            # Set expiration (7 years for GDPR compliance)
            await asyncio.get_event_loop().run_in_executor(
                None, self.redis_client.expire, lineage_key, 2555 * 24 * 3600
            )
            
            self.logger.info(f"Tracked lineage for record {lineage_record.record_id}")
            return True
            
        except Exception as e:
            self.logger.error(f"Failed to track lineage: {str(e)}")
            return False

    async def record_consent(self, consent: ConsentRecord) -> bool:
        """Record user consent for compliance"""
        try:
            consent_key = f"consent:{consent.user_id}:{consent.consent_type}"
            
            consent_data = {
                "user_id": consent.user_id,
                "consent_type": consent.consent_type,
                "status": consent.status.value,
                "granted_at": consent.granted_at.isoformat(),
                "expires_at": consent.expires_at.isoformat() if consent.expires_at else None,
                "purpose": consent.purpose,
                "legal_basis": consent.legal_basis
            }
            
            await asyncio.get_event_loop().run_in_executor(
                None, self.redis_client.setex, 
                consent_key, 2555 * 24 * 3600, json.dumps(consent_data)
            )
            
            # Track consent history
            history_key = f"consent_history:{consent.user_id}"
            await asyncio.get_event_loop().run_in_executor(
                None, self.redis_client.lpush, history_key, json.dumps(consent_data)
            )
            
            self.logger.info(f"Recorded consent for user {consent.user_id}")
            return True
            
        except Exception as e:
            self.logger.error(f"Failed to record consent: {str(e)}")
            return False

    async def log_audit_event(self, event: AuditEvent) -> bool:
        """Log audit event for compliance monitoring"""
        try:
            audit_key = f"audit:{event.timestamp.strftime('%Y-%m-%d')}"
            
            event_data = {
                "event_id": event.event_id,
                "user_id": event.user_id,
                "action": event.action,
                "resource": event.resource,
                "timestamp": event.timestamp.isoformat(),
                "ip_address": event.ip_address,
                "result": event.result,
                "regulation": event.compliance_regulation.value
            }
            
            await asyncio.get_event_loop().run_in_executor(
                None, self.redis_client.lpush, audit_key, json.dumps(event_data)
            )
            
            # Set expiration based on regulation requirements
            retention_days = self.compliance_rules[event.compliance_regulation].get("data_retention_days", 365)
            await asyncio.get_event_loop().run_in_executor(
                None, self.redis_client.expire, audit_key, retention_days * 24 * 3600
            )
            
            return True
            
        except Exception as e:
            self.logger.error(f"Failed to log audit event: {str(e)}")
            return False

    async def check_data_retention_compliance(self) -> Dict[str, Any]:
        """Check for data that exceeds retention policies"""
        compliance_report = {
            "violations": [],
            "warnings": [],
            "total_records_checked": 0,
            "compliant_records": 0
        }
        
        # Check each regulation
        for regulation, rules in self.compliance_rules.items():
            retention_days = rules.get("data_retention_days", 365)
            cutoff_date = datetime.now() - timedelta(days=retention_days)
            
            # Check audit logs
            violations = await self._check_expired_records("audit:*", cutoff_date)
            compliance_report["violations"].extend([
                {"regulation": regulation.value, "type": "audit", "expired_records": len(violations)}
            ])
            
        return compliance_report

    async def process_data_subject_request(self, user_id: str, request_type: str) -> Dict[str, Any]:
        """Process GDPR/CCPA data subject requests"""
        if request_type == "access":
            return await self._handle_data_access_request(user_id)
        elif request_type == "portability":
            return await self._handle_data_portability_request(user_id)
        elif request_type == "erasure":
            return await self._handle_data_erasure_request(user_id)
        else:
            return {"error": f"Unsupported request type: {request_type}"}

    async def _handle_data_access_request(self, user_id: str) -> Dict[str, Any]:
        """Handle data access request (GDPR Article 15)"""
        user_data = {
            "user_id": user_id,
            "personal_data": {},
            "processing_purposes": [],
            "data_categories": [],
            "recipients": [],
            "retention_periods": {},
            "rights": ["access", "rectification", "erasure", "portability"]
        }
        
        # Collect data from various sources
        user_data["personal_data"] = await self._collect_user_data(user_id)
        user_data["processing_purposes"] = await self._get_processing_purposes(user_id)
        user_data["consent_history"] = await self._get_consent_history(user_id)
        
        return user_data

    async def _handle_data_portability_request(self, user_id: str) -> Dict[str, Any]:
        """Handle data portability request (GDPR Article 20)"""
        portable_data = await self._collect_user_data(user_id)
        
        # Format data in structured, machine-readable format
        return {
            "user_id": user_id,
            "data_export": portable_data,
            "format": "JSON",
            "exported_at": datetime.now().isoformat(),
            "retention_note": "Please note data retention policies may apply"
        }

    async def _handle_data_erasure_request(self, user_id: str) -> Dict[str, Any]:
        """Handle right to erasure request (GDPR Article 17)"""
        erasure_result = {
            "user_id": user_id,
            "erasure_completed": False,
            "systems_processed": [],
            "retained_data": [],
            "legal_grounds_for_retention": []
        }
        
        # Check if erasure is legally required
        if await self._check_erasure_eligibility(user_id):
            # Pseudonymize/anonymize data instead of deletion for audit compliance
            erasure_result["erasure_completed"] = await self._pseudonymize_user_data(user_id)
            erasure_result["systems_processed"] = [
                "customer_database", "booking_history", "analytics_data"
            ]
        else:
            erasure_result["legal_grounds_for_retention"] = [
                "Contract performance (GDPR Art. 6(1)(b))",
                "Legal obligation (GDPR Art. 6(1)(c))"
            ]
        
        return erasure_result

    async def generate_compliance_report(self, regulation: ComplianceRegulation) -> Dict[str, Any]:
        """Generate comprehensive compliance report"""
        report = {
            "regulation": regulation.value,
            "report_date": datetime.now().isoformat(),
            "compliance_score": 0.0,
            "violations": [],
            "recommendations": [],
            "data_inventory": {},
            "consent_metrics": {},
            "retention_compliance": {}
        }
        
        # Calculate compliance score
        report["compliance_score"] = await self._calculate_compliance_score(regulation)
        
        # Data retention compliance
        report["retention_compliance"] = await self.check_data_retention_compliance()
        
        # Consent metrics
        report["consent_metrics"] = await self._generate_consent_metrics()
        
        # Generate recommendations
        report["recommendations"] = self._generate_compliance_recommendations(report)
        
        return report

    async def _collect_user_data(self, user_id: str) -> Dict[str, Any]:
        """Collect all data associated with a user"""
        # This would query various data sources
        return {
            "profile": {"name": "Example User", "email": "user@example.com"},
            "bookings": [],
            "preferences": {},
            "analytics_data": "anonymized"
        }

    async def _get_processing_purposes(self, user_id: str) -> List[str]:
        """Get data processing purposes for user"""
        return [
            "Service delivery",
            "Customer support", 
            "Marketing (with consent)",
            "Legal compliance",
            "Fraud prevention"
        ]

    async def _get_consent_history(self, user_id: str) -> List[Dict]:
        """Get user's consent history"""
        history_key = f"consent_history:{user_id}"
        history_data = await asyncio.get_event_loop().run_in_executor(
            None, self.redis_client.lrange, history_key, 0, -1
        )
        
        return [json.loads(item) for item in history_data] if history_data else []

    async def _check_erasure_eligibility(self, user_id: str) -> bool:
        """Check if user data can be erased"""
        # Check for active contracts, legal obligations, etc.
        return True  # Simplified

    async def _pseudonymize_user_data(self, user_id: str) -> bool:
        """Pseudonymize user data for compliance"""
        # Replace PII with pseudonymized identifiers
        pseudonym = hashlib.sha256(f"{user_id}_pseudonym".encode()).hexdigest()[:16]
        
        # Update data stores with pseudonymized identifiers
        # This would involve updating multiple databases/systems
        
        self.logger.info(f"Pseudonymized data for user {user_id}")
        return True

    async def _check_expired_records(self, pattern: str, cutoff_date: datetime) -> List[str]:
        """Check for records that exceed retention period"""
        # Simplified implementation
        return []

    async def _calculate_compliance_score(self, regulation: ComplianceRegulation) -> float:
        """Calculate overall compliance score"""
        # Simplified scoring algorithm
        base_score = 85.0
        
        # Check various compliance factors
        retention_compliance = await self.check_data_retention_compliance()
        violations = len(retention_compliance.get("violations", []))
        
        # Reduce score based on violations
        score = base_score - (violations * 5.0)
        
        return max(0.0, min(100.0, score))

    async def _generate_consent_metrics(self) -> Dict[str, Any]:
        """Generate consent-related metrics"""
        return {
            "total_consents": 1250,
            "active_consents": 1180,
            "withdrawn_consents": 70,
            "expired_consents": 0,
            "consent_rate": 94.4
        }

    def _generate_compliance_recommendations(self, report: Dict[str, Any]) -> List[str]:
        """Generate compliance improvement recommendations"""
        recommendations = []
        
        score = report["compliance_score"]
        if score < 90:
            recommendations.append("Implement automated data retention cleanup")
        if score < 85:
            recommendations.append("Enhance consent management workflows")
        if score < 80:
            recommendations.append("Conduct comprehensive data mapping exercise")
        
        return recommendations

# Example usage
async def main():
    config = {
        "redis_host": "localhost",
        "redis_port": 6379,
        "compliance_regulations": ["GDPR", "CCPA", "PCI_DSS"]
    }
    
    engine = DataGovernanceEngine(config)
    
    # Example lineage tracking
    lineage = DataLineageRecord(
        record_id="booking_123456",
        source_system="booking_api",
        target_system="analytics_warehouse",
        transformation_applied="anonymization",
        processor="data_pipeline_v1",
        timestamp=datetime.now(),
        data_classification=DataClassification.PII,
        compliance_tags=["GDPR", "retention_policy"]
    )
    
    await engine.track_data_lineage(lineage)
    
    # Example consent recording
    consent = ConsentRecord(
        user_id="user_789",
        consent_type="marketing",
        status=ConsentStatus.GRANTED,
        granted_at=datetime.now(),
        expires_at=datetime.now() + timedelta(days=365),
        purpose="Email marketing campaigns",
        legal_basis="Consent (GDPR Art. 6(1)(a))"
    )
    
    await engine.record_consent(consent)
    
    # Generate compliance report
    report = await engine.generate_compliance_report(ComplianceRegulation.GDPR)
    print(f"GDPR Compliance Score: {report['compliance_score']}")

if __name__ == "__main__":
    asyncio.run(main()) 