import json
import uuid
import logging
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any
from dataclasses import dataclass, asdict
from enum import Enum
import asyncio
from pymongo import MongoClient
import hashlib
import jwt
from cryptography.fernet import Fernet

class ConsentType(Enum):
    MARKETING = "marketing"
    ANALYTICS = "analytics"
    FUNCTIONAL = "functional"
    PERSONALIZATION = "personalization"
    THIRD_PARTY = "third_party"
    RESEARCH = "research"
    PROFILING = "profiling"

class ConsentStatus(Enum):
    GRANTED = "granted"
    DENIED = "denied"
    WITHDRAWN = "withdrawn"
    EXPIRED = "expired"
    PENDING = "pending"

class DataSubjectRightType(Enum):
    ACCESS = "access"
    RECTIFICATION = "rectification"
    ERASURE = "erasure"
    RESTRICTION = "restriction"
    PORTABILITY = "portability"
    OBJECTION = "objection"
    AUTOMATED_DECISION = "automated_decision"

@dataclass
class ConsentRecord:
    id: str
    user_id: str
    consent_type: ConsentType
    status: ConsentStatus
    purpose: str
    legal_basis: str
    granted_at: Optional[datetime]
    withdrawn_at: Optional[datetime]
    expires_at: Optional[datetime]
    version: str
    channel: str
    ip_address: str
    user_agent: str
    evidence: Dict[str, Any]
    metadata: Dict[str, Any]
    created_at: datetime
    updated_at: datetime

@dataclass
class DataSubjectRequest:
    id: str
    user_id: str
    request_type: DataSubjectRightType
    status: str
    description: str
    requested_data: List[str]
    legal_basis: str
    identity_verified: bool
    verification_method: str
    processing_notes: List[str]
    response_data: Optional[Dict[str, Any]]
    deadline: datetime
    created_at: datetime
    updated_at: datetime
    completed_at: Optional[datetime]

@dataclass
class PreferenceProfile:
    user_id: str
    communication_preferences: Dict[str, bool]
    data_processing_preferences: Dict[str, str]
    marketing_preferences: Dict[str, bool]
    privacy_settings: Dict[str, Any]
    notification_settings: Dict[str, bool]
    language_preference: str
    timezone: str
    accessibility_settings: Dict[str, Any]
    last_updated: datetime

class PrivacyConsentManager:
    def __init__(self, db_connection: str, encryption_key: bytes):
        self.db = MongoClient(db_connection)
        self.consent_db = self.db.privacy_consent
        self.cipher_suite = Fernet(encryption_key)
        self.logger = logging.getLogger(__name__)
        
        # Initialize components
        self.consent_engine = ConsentEngine(self.consent_db, self.cipher_suite)
        self.dsr_processor = DataSubjectRequestProcessor(self.consent_db, self.cipher_suite)
        self.preference_manager = PreferenceManager(self.consent_db, self.cipher_suite)
        self.analytics_engine = ConsentAnalyticsEngine(self.consent_db)
        self.compliance_monitor = ComplianceMonitor(self.consent_db)
        self.automation_engine = PrivacyAutomationEngine(self.consent_db)

class ConsentEngine:
    def __init__(self, db, cipher_suite):
        self.db = db
        self.cipher_suite = cipher_suite
        self.logger = logging.getLogger(__name__)

    async def capture_consent(self, user_id: str, consent_data: Dict[str, Any]) -> ConsentRecord:
        """Capture and record user consent with comprehensive evidence"""
        try:
            consent_id = str(uuid.uuid4())
            
            # Create consent record
            consent_record = ConsentRecord(
                id=consent_id,
                user_id=user_id,
                consent_type=ConsentType(consent_data['type']),
                status=ConsentStatus.GRANTED,
                purpose=consent_data['purpose'],
                legal_basis=consent_data.get('legal_basis', 'consent'),
                granted_at=datetime.utcnow(),
                withdrawn_at=None,
                expires_at=self._calculate_expiry(consent_data.get('duration')),
                version=consent_data.get('version', '1.0'),
                channel=consent_data.get('channel', 'web'),
                ip_address=consent_data.get('ip_address', ''),
                user_agent=consent_data.get('user_agent', ''),
                evidence=self._capture_evidence(consent_data),
                metadata=consent_data.get('metadata', {}),
                created_at=datetime.utcnow(),
                updated_at=datetime.utcnow()
            )

            # Encrypt sensitive data
            encrypted_record = self._encrypt_consent_data(consent_record)
            
            # Store in database
            await self.db.consent_records.insert_one(asdict(encrypted_record))
            
            # Update consent analytics
            await self._update_consent_analytics(consent_record)
            
            # Trigger downstream processing
            await self._trigger_consent_processing(consent_record)
            
            self.logger.info(f"Consent captured: {consent_id} for user {user_id}")
            return consent_record
            
        except Exception as e:
            self.logger.error(f"Error capturing consent: {e}")
            raise

    async def withdraw_consent(self, user_id: str, consent_type: ConsentType, reason: str = "") -> bool:
        """Withdraw user consent and trigger data processing restrictions"""
        try:
            # Find active consent
            filter_query = {
                'user_id': user_id,
                'consent_type': consent_type.value,
                'status': ConsentStatus.GRANTED.value
            }
            
            consent_record = await self.db.consent_records.find_one(filter_query)
            if not consent_record:
                raise ValueError(f"No active consent found for {consent_type.value}")
            
            # Update consent record
            update_data = {
                'status': ConsentStatus.WITHDRAWN.value,
                'withdrawn_at': datetime.utcnow(),
                'updated_at': datetime.utcnow(),
                'withdrawal_reason': reason
            }
            
            await self.db.consent_records.update_one(
                {'id': consent_record['id']}, 
                {'$set': update_data}
            )
            
            # Trigger data processing restrictions
            await self._trigger_withdrawal_processing(user_id, consent_type, reason)
            
            # Update analytics
            await self._update_withdrawal_analytics(user_id, consent_type)
            
            self.logger.info(f"Consent withdrawn: {consent_type.value} for user {user_id}")
            return True
            
        except Exception as e:
            self.logger.error(f"Error withdrawing consent: {e}")
            raise

    async def get_consent_status(self, user_id: str, consent_type: Optional[ConsentType] = None) -> Dict[str, Any]:
        """Get current consent status for user"""
        try:
            query = {'user_id': user_id}
            if consent_type:
                query['consent_type'] = consent_type.value
                
            consents = await self.db.consent_records.find(query).to_list(length=None)
            
            # Decrypt and process consent data
            consent_status = {}
            for consent in consents:
                decrypted_consent = self._decrypt_consent_data(consent)
                consent_status[consent['consent_type']] = {
                    'status': consent['status'],
                    'granted_at': consent.get('granted_at'),
                    'expires_at': consent.get('expires_at'),
                    'version': consent.get('version', '1.0')
                }
            
            return consent_status
            
        except Exception as e:
            self.logger.error(f"Error getting consent status: {e}")
            raise

    def _calculate_expiry(self, duration: Optional[str]) -> Optional[datetime]:
        """Calculate consent expiry date based on duration"""
        if not duration:
            return None
            
        duration_map = {
            '1year': timedelta(days=365),
            '2years': timedelta(days=730),
            '3years': timedelta(days=1095),
            'indefinite': None
        }
        
        delta = duration_map.get(duration)
        return datetime.utcnow() + delta if delta else None

    def _capture_evidence(self, consent_data: Dict[str, Any]) -> Dict[str, Any]:
        """Capture comprehensive evidence of consent"""
        return {
            'consent_text': consent_data.get('consent_text', ''),
            'privacy_policy_version': consent_data.get('privacy_policy_version', ''),
            'opt_in_method': consent_data.get('opt_in_method', ''),
            'form_data': consent_data.get('form_data', {}),
            'session_id': consent_data.get('session_id', ''),
            'timestamp': datetime.utcnow().isoformat(),
            'checksum': self._generate_checksum(consent_data)
        }

    def _generate_checksum(self, data: Dict[str, Any]) -> str:
        """Generate checksum for data integrity"""
        data_str = json.dumps(data, sort_keys=True, default=str)
        return hashlib.sha256(data_str.encode()).hexdigest()

    def _encrypt_consent_data(self, consent_record: ConsentRecord) -> ConsentRecord:
        """Encrypt sensitive consent data"""
        # Encrypt PII fields
        if consent_record.ip_address:
            consent_record.ip_address = self.cipher_suite.encrypt(
                consent_record.ip_address.encode()
            ).decode()
        
        return consent_record

    def _decrypt_consent_data(self, consent_data: Dict[str, Any]) -> Dict[str, Any]:
        """Decrypt sensitive consent data"""
        if consent_data.get('ip_address'):
            try:
                consent_data['ip_address'] = self.cipher_suite.decrypt(
                    consent_data['ip_address'].encode()
                ).decode()
            except Exception:
                pass  # Handle decryption errors gracefully
        
        return consent_data

    async def _update_consent_analytics(self, consent_record: ConsentRecord):
        """Update consent analytics"""
        analytics_data = {
            'event_type': 'consent_granted',
            'consent_type': consent_record.consent_type.value,
            'user_id': consent_record.user_id,
            'channel': consent_record.channel,
            'timestamp': datetime.utcnow()
        }
        
        await self.db.consent_analytics.insert_one(analytics_data)

    async def _trigger_consent_processing(self, consent_record: ConsentRecord):
        """Trigger downstream consent processing"""
        # Notify data processing systems
        processing_event = {
            'event_type': 'consent_granted',
            'user_id': consent_record.user_id,
            'consent_type': consent_record.consent_type.value,
            'legal_basis': consent_record.legal_basis,
            'timestamp': datetime.utcnow()
        }
        
        await self.db.processing_events.insert_one(processing_event)

    async def _trigger_withdrawal_processing(self, user_id: str, consent_type: ConsentType, reason: str):
        """Trigger consent withdrawal processing"""
        processing_event = {
            'event_type': 'consent_withdrawn',
            'user_id': user_id,
            'consent_type': consent_type.value,
            'reason': reason,
            'timestamp': datetime.utcnow(),
            'actions_required': [
                'stop_data_processing',
                'delete_derived_data',
                'update_user_profile'
            ]
        }
        
        await self.db.processing_events.insert_one(processing_event)

    async def _update_withdrawal_analytics(self, user_id: str, consent_type: ConsentType):
        """Update withdrawal analytics"""
        analytics_data = {
            'event_type': 'consent_withdrawn',
            'consent_type': consent_type.value,
            'user_id': user_id,
            'timestamp': datetime.utcnow()
        }
        
        await self.db.consent_analytics.insert_one(analytics_data)

class DataSubjectRequestProcessor:
    def __init__(self, db, cipher_suite):
        self.db = db
        self.cipher_suite = cipher_suite
        self.logger = logging.getLogger(__name__)

    async def submit_request(self, user_id: str, request_data: Dict[str, Any]) -> DataSubjectRequest:
        """Submit a new data subject request"""
        try:
            request_id = str(uuid.uuid4())
            
            # Create DSR
            dsr = DataSubjectRequest(
                id=request_id,
                user_id=user_id,
                request_type=DataSubjectRightType(request_data['type']),
                status='pending_verification',
                description=request_data.get('description', ''),
                requested_data=request_data.get('requested_data', []),
                legal_basis=request_data.get('legal_basis', 'gdpr_article_15'),
                identity_verified=False,
                verification_method='',
                processing_notes=[],
                response_data=None,
                deadline=datetime.utcnow() + timedelta(days=30),  # GDPR 30-day requirement
                created_at=datetime.utcnow(),
                updated_at=datetime.utcnow(),
                completed_at=None
            )

            # Store request
            await self.db.data_subject_requests.insert_one(asdict(dsr))
            
            # Trigger identity verification
            await self._initiate_identity_verification(dsr)
            
            # Schedule automated processing
            await self._schedule_automated_processing(dsr)
            
            self.logger.info(f"DSR submitted: {request_id} for user {user_id}")
            return dsr
            
        except Exception as e:
            self.logger.error(f"Error submitting DSR: {e}")
            raise

    async def process_access_request(self, request_id: str) -> Dict[str, Any]:
        """Process data access request automatically"""
        try:
            # Get request
            dsr = await self.db.data_subject_requests.find_one({'id': request_id})
            if not dsr:
                raise ValueError("Request not found")

            # Collect user data from all systems
            user_data = await self._collect_user_data(dsr['user_id'])
            
            # Apply data minimization
            filtered_data = await self._apply_data_minimization(user_data, dsr)
            
            # Generate response package
            response_package = {
                'request_id': request_id,
                'user_id': dsr['user_id'],
                'data_collected': filtered_data,
                'collection_timestamp': datetime.utcnow(),
                'format': 'json',
                'encryption': 'aes_256',
                'retention_info': await self._get_retention_info(dsr['user_id'])
            }

            # Encrypt response
            encrypted_response = self._encrypt_response(response_package)
            
            # Update request status
            await self.db.data_subject_requests.update_one(
                {'id': request_id},
                {
                    '$set': {
                        'status': 'completed',
                        'response_data': encrypted_response,
                        'completed_at': datetime.utcnow(),
                        'updated_at': datetime.utcnow()
                    }
                }
            )

            # Send response to user
            await self._send_response_to_user(dsr['user_id'], encrypted_response)
            
            return response_package
            
        except Exception as e:
            self.logger.error(f"Error processing access request: {e}")
            raise

    async def process_erasure_request(self, request_id: str) -> bool:
        """Process data erasure request with verification"""
        try:
            # Get request
            dsr = await self.db.data_subject_requests.find_one({'id': request_id})
            if not dsr:
                raise ValueError("Request not found")

            # Verify erasure conditions
            can_erase = await self._verify_erasure_conditions(dsr['user_id'])
            if not can_erase:
                await self._update_request_status(request_id, 'denied', 'Legal basis prevents erasure')
                return False

            # Execute erasure across all systems
            erasure_results = await self._execute_data_erasure(dsr['user_id'])
            
            # Verify erasure completion
            verification_results = await self._verify_erasure_completion(dsr['user_id'])
            
            # Update request
            await self.db.data_subject_requests.update_one(
                {'id': request_id},
                {
                    '$set': {
                        'status': 'completed',
                        'response_data': {
                            'erasure_results': erasure_results,
                            'verification': verification_results
                        },
                        'completed_at': datetime.utcnow(),
                        'updated_at': datetime.utcnow()
                    }
                }
            )

            # Log compliance action
            await self._log_compliance_action('data_erasure', dsr['user_id'], erasure_results)
            
            return True
            
        except Exception as e:
            self.logger.error(f"Error processing erasure request: {e}")
            raise

    async def _collect_user_data(self, user_id: str) -> Dict[str, Any]:
        """Collect comprehensive user data from all systems"""
        user_data = {
            'profile_data': await self._get_profile_data(user_id),
            'booking_data': await self._get_booking_data(user_id),
            'consent_data': await self._get_consent_data(user_id),
            'analytics_data': await self._get_analytics_data(user_id),
            'communication_data': await self._get_communication_data(user_id),
            'system_logs': await self._get_system_logs(user_id),
            'preferences': await self._get_preferences(user_id)
        }
        
        return user_data

    async def _apply_data_minimization(self, user_data: Dict[str, Any], dsr: Dict[str, Any]) -> Dict[str, Any]:
        """Apply data minimization principles to response"""
        # Filter data based on request scope
        if dsr.get('requested_data'):
            filtered_data = {}
            for data_type in dsr['requested_data']:
                if data_type in user_data:
                    filtered_data[data_type] = user_data[data_type]
            return filtered_data
        
        return user_data

    async def _execute_data_erasure(self, user_id: str) -> Dict[str, Any]:
        """Execute data erasure across all systems"""
        erasure_results = {
            'user_profile': await self._erase_user_profile(user_id),
            'booking_history': await self._erase_booking_history(user_id),
            'analytics_data': await self._erase_analytics_data(user_id),
            'consent_records': await self._erase_consent_records(user_id),
            'communication_logs': await self._erase_communication_logs(user_id),
            'backups': await self._mark_backups_for_erasure(user_id)
        }
        
        return erasure_results

    async def _initiate_identity_verification(self, dsr: DataSubjectRequest):
        """Initiate identity verification process"""
        verification_token = str(uuid.uuid4())
        
        verification_data = {
            'request_id': dsr.id,
            'user_id': dsr.user_id,
            'token': verification_token,
            'method': 'email_verification',
            'expires_at': datetime.utcnow() + timedelta(hours=24),
            'created_at': datetime.utcnow()
        }
        
        await self.db.identity_verifications.insert_one(verification_data)
        
        # Send verification email
        await self._send_verification_email(dsr.user_id, verification_token)

    async def _send_verification_email(self, user_id: str, token: str):
        """Send identity verification email"""
        # Mock email sending
        self.logger.info(f"Verification email sent to user {user_id} with token {token}")

    async def _schedule_automated_processing(self, dsr: DataSubjectRequest):
        """Schedule automated request processing"""
        processing_schedule = {
            'request_id': dsr.id,
            'scheduled_for': datetime.utcnow() + timedelta(hours=1),  # Process after verification
            'auto_process': True,
            'created_at': datetime.utcnow()
        }
        
        await self.db.processing_schedule.insert_one(processing_schedule)

class PreferenceManager:
    def __init__(self, db, cipher_suite):
        self.db = db
        self.cipher_suite = cipher_suite
        self.logger = logging.getLogger(__name__)

    async def create_preference_profile(self, user_id: str, preferences: Dict[str, Any]) -> PreferenceProfile:
        """Create comprehensive user preference profile"""
        try:
            profile = PreferenceProfile(
                user_id=user_id,
                communication_preferences={
                    'email_marketing': preferences.get('email_marketing', False),
                    'sms_notifications': preferences.get('sms_notifications', False),
                    'push_notifications': preferences.get('push_notifications', True),
                    'phone_calls': preferences.get('phone_calls', False),
                    'postal_mail': preferences.get('postal_mail', False)
                },
                data_processing_preferences={
                    'analytics': preferences.get('analytics_processing', 'minimal'),
                    'personalization': preferences.get('personalization_level', 'basic'),
                    'profiling': preferences.get('profiling_consent', 'denied'),
                    'third_party_sharing': preferences.get('third_party_sharing', 'denied')
                },
                marketing_preferences={
                    'flight_deals': preferences.get('flight_deals', False),
                    'destination_offers': preferences.get('destination_offers', False),
                    'loyalty_updates': preferences.get('loyalty_updates', True),
                    'partner_offers': preferences.get('partner_offers', False)
                },
                privacy_settings={
                    'data_retention': preferences.get('data_retention', 'minimum'),
                    'cookie_settings': preferences.get('cookie_settings', 'essential_only'),
                    'tracking_consent': preferences.get('tracking_consent', False),
                    'data_export_format': preferences.get('data_export_format', 'json')
                },
                notification_settings={
                    'booking_confirmations': preferences.get('booking_confirmations', True),
                    'flight_updates': preferences.get('flight_updates', True),
                    'service_announcements': preferences.get('service_announcements', True),
                    'policy_updates': preferences.get('policy_updates', True)
                },
                language_preference=preferences.get('language', 'en'),
                timezone=preferences.get('timezone', 'UTC'),
                accessibility_settings={
                    'screen_reader': preferences.get('screen_reader', False),
                    'high_contrast': preferences.get('high_contrast', False),
                    'large_text': preferences.get('large_text', False)
                },
                last_updated=datetime.utcnow()
            )

            # Store encrypted profile
            encrypted_profile = self._encrypt_preferences(profile)
            await self.db.preference_profiles.insert_one(asdict(encrypted_profile))
            
            # Update consent records based on preferences
            await self._sync_preferences_with_consent(profile)
            
            self.logger.info(f"Preference profile created for user {user_id}")
            return profile
            
        except Exception as e:
            self.logger.error(f"Error creating preference profile: {e}")
            raise

    async def update_preferences(self, user_id: str, updates: Dict[str, Any]) -> bool:
        """Update user preferences with consent implications"""
        try:
            # Get current profile
            current_profile = await self.db.preference_profiles.find_one({'user_id': user_id})
            if not current_profile:
                raise ValueError("Preference profile not found")

            # Apply updates
            update_data = {'last_updated': datetime.utcnow()}
            
            for category, settings in updates.items():
                if category in current_profile:
                    if isinstance(current_profile[category], dict):
                        update_data[f"{category}"] = {**current_profile[category], **settings}
                    else:
                        update_data[category] = settings

            # Update database
            await self.db.preference_profiles.update_one(
                {'user_id': user_id},
                {'$set': update_data}
            )

            # Check for consent implications
            await self._handle_consent_implications(user_id, updates)
            
            # Log preference change
            await self._log_preference_change(user_id, updates)
            
            return True
            
        except Exception as e:
            self.logger.error(f"Error updating preferences: {e}")
            raise

    def _encrypt_preferences(self, profile: PreferenceProfile) -> PreferenceProfile:
        """Encrypt sensitive preference data"""
        # Encrypt PII and sensitive settings
        return profile  # Simplified for brevity

    async def _sync_preferences_with_consent(self, profile: PreferenceProfile):
        """Sync preferences with consent records"""
        # Update consent based on preference changes
        consent_updates = []
        
        if not profile.communication_preferences['email_marketing']:
            consent_updates.append({
                'type': ConsentType.MARKETING,
                'action': 'withdraw'
            })
        
        if profile.data_processing_preferences['analytics'] == 'denied':
            consent_updates.append({
                'type': ConsentType.ANALYTICS,
                'action': 'withdraw'
            })

        # Process consent updates
        for update in consent_updates:
            await self._process_consent_update(profile.user_id, update)

    async def _handle_consent_implications(self, user_id: str, updates: Dict[str, Any]):
        """Handle consent implications of preference changes"""
        # Analyze preference changes for consent impact
        consent_changes = self._analyze_consent_impact(updates)
        
        for change in consent_changes:
            await self._process_consent_change(user_id, change)

    def _analyze_consent_impact(self, updates: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Analyze preference updates for consent implications"""
        consent_changes = []
        
        # Check for marketing opt-outs
        if 'communication_preferences' in updates:
            comm_prefs = updates['communication_preferences']
            if not comm_prefs.get('email_marketing', True):
                consent_changes.append({
                    'consent_type': ConsentType.MARKETING,
                    'action': 'withdraw',
                    'reason': 'user_preference_change'
                })

        return consent_changes

class ConsentAnalyticsEngine:
    def __init__(self, db):
        self.db = db
        self.logger = logging.getLogger(__name__)

    async def generate_consent_analytics(self, time_range: str = '30d') -> Dict[str, Any]:
        """Generate comprehensive consent analytics"""
        try:
            analytics = {
                'summary': await self._get_consent_summary(time_range),
                'trends': await self._get_consent_trends(time_range),
                'breakdown': await self._get_consent_breakdown(time_range),
                'conversion_rates': await self._get_conversion_rates(time_range),
                'withdrawal_analysis': await self._get_withdrawal_analysis(time_range),
                'compliance_metrics': await self._get_compliance_metrics(time_range),
                'recommendations': await self._generate_recommendations()
            }
            
            return analytics
            
        except Exception as e:
            self.logger.error(f"Error generating consent analytics: {e}")
            raise

    async def _get_consent_summary(self, time_range: str) -> Dict[str, Any]:
        """Get consent summary statistics"""
        # Calculate time range
        end_date = datetime.utcnow()
        start_date = end_date - timedelta(days=int(time_range.rstrip('d')))
        
        # Aggregate consent data
        pipeline = [
            {'$match': {'created_at': {'$gte': start_date, '$lte': end_date}}},
            {'$group': {
                '_id': '$consent_type',
                'total_granted': {'$sum': {'$cond': [{'$eq': ['$status', 'granted']}, 1, 0]}},
                'total_denied': {'$sum': {'$cond': [{'$eq': ['$status', 'denied']}, 1, 0]}},
                'total_withdrawn': {'$sum': {'$cond': [{'$eq': ['$status', 'withdrawn']}, 1, 0]}}
            }}
        ]
        
        results = await self.db.consent_records.aggregate(pipeline).to_list(length=None)
        
        summary = {
            'total_consent_requests': sum(r['total_granted'] + r['total_denied'] for r in results),
            'total_granted': sum(r['total_granted'] for r in results),
            'total_denied': sum(r['total_denied'] for r in results),
            'total_withdrawn': sum(r['total_withdrawn'] for r in results),
            'consent_rate': 0.0,
            'by_type': {r['_id']: r for r in results}
        }
        
        if summary['total_consent_requests'] > 0:
            summary['consent_rate'] = summary['total_granted'] / summary['total_consent_requests'] * 100
        
        return summary

    async def _generate_recommendations(self) -> List[str]:
        """Generate actionable recommendations based on consent data"""
        recommendations = [
            "Optimize consent flow to reduce abandonment",
            "Implement granular consent options for better user control",
            "Add consent renewal reminders before expiration",
            "Improve consent withdrawal process visibility",
            "Implement progressive consent collection",
            "Add consent analytics dashboard for stakeholders"
        ]
        
        return recommendations

class ComplianceMonitor:
    def __init__(self, db):
        self.db = db
        self.logger = logging.getLogger(__name__)

    async def run_compliance_audit(self) -> Dict[str, Any]:
        """Run comprehensive privacy compliance audit"""
        try:
            audit_results = {
                'gdpr_compliance': await self._audit_gdpr_compliance(),
                'pdpa_compliance': await self._audit_pdpa_compliance(),
                'ccpa_compliance': await self._audit_ccpa_compliance(),
                'consent_management': await self._audit_consent_management(),
                'data_subject_rights': await self._audit_dsr_processing(),
                'data_retention': await self._audit_data_retention(),
                'recommendations': await self._generate_compliance_recommendations()
            }
            
            # Calculate overall compliance score
            audit_results['overall_score'] = self._calculate_overall_score(audit_results)
            audit_results['audit_timestamp'] = datetime.utcnow()
            
            # Store audit results
            await self.db.compliance_audits.insert_one(audit_results)
            
            return audit_results
            
        except Exception as e:
            self.logger.error(f"Error running compliance audit: {e}")
            raise

    async def _audit_gdpr_compliance(self) -> Dict[str, Any]:
        """Audit GDPR compliance"""
        compliance_checks = {
            'consent_management': await self._check_consent_requirements(),
            'data_subject_rights': await self._check_dsr_compliance(),
            'data_protection_officer': await self._check_dpo_requirements(),
            'privacy_by_design': await self._check_privacy_by_design(),
            'breach_notification': await self._check_breach_procedures()
        }
        
        score = sum(1 for check in compliance_checks.values() if check['compliant']) / len(compliance_checks) * 100
        
        return {
            'score': score,
            'checks': compliance_checks,
            'compliant': score >= 90
        }

class PrivacyAutomationEngine:
    def __init__(self, db):
        self.db = db
        self.logger = logging.getLogger(__name__)

    async def automate_consent_renewal(self):
        """Automatically handle consent renewals"""
        try:
            # Find expiring consents
            expiry_threshold = datetime.utcnow() + timedelta(days=30)
            expiring_consents = await self.db.consent_records.find({
                'expires_at': {'$lte': expiry_threshold},
                'status': 'granted'
            }).to_list(length=None)

            for consent in expiring_consents:
                await self._send_renewal_reminder(consent)
                
            self.logger.info(f"Processed {len(expiring_consents)} consent renewals")
            
        except Exception as e:
            self.logger.error(f"Error in consent renewal automation: {e}")

    async def automate_data_retention(self):
        """Automatically enforce data retention policies"""
        try:
            # Get retention policies
            policies = await self.db.retention_policies.find({}).to_list(length=None)
            
            for policy in policies:
                await self._enforce_retention_policy(policy)
                
            self.logger.info("Data retention policies enforced")
            
        except Exception as e:
            self.logger.error(f"Error in data retention automation: {e}")

    async def _send_renewal_reminder(self, consent):
        """Send consent renewal reminder"""
        reminder_data = {
            'user_id': consent['user_id'],
            'consent_type': consent['consent_type'],
            'expires_at': consent['expires_at'],
            'reminder_sent_at': datetime.utcnow()
        }
        
        await self.db.renewal_reminders.insert_one(reminder_data)
        # Mock email sending
        self.logger.info(f"Renewal reminder sent for consent {consent['id']}")

# API Interface Functions
async def capture_user_consent(user_id: str, consent_data: Dict[str, Any]) -> Dict[str, Any]:
    """API function to capture user consent"""
    manager = PrivacyConsentManager("mongodb://localhost:27017", Fernet.generate_key())
    
    try:
        consent_record = await manager.consent_engine.capture_consent(user_id, consent_data)
        return {
            'success': True,
            'consent_id': consent_record.id,
            'status': consent_record.status.value,
            'expires_at': consent_record.expires_at.isoformat() if consent_record.expires_at else None
        }
    except Exception as e:
        return {
            'success': False,
            'error': str(e)
        }

async def submit_data_subject_request(user_id: str, request_data: Dict[str, Any]) -> Dict[str, Any]:
    """API function to submit data subject request"""
    manager = PrivacyConsentManager("mongodb://localhost:27017", Fernet.generate_key())
    
    try:
        dsr = await manager.dsr_processor.submit_request(user_id, request_data)
        return {
            'success': True,
            'request_id': dsr.id,
            'status': dsr.status,
            'deadline': dsr.deadline.isoformat()
        }
    except Exception as e:
        return {
            'success': False,
            'error': str(e)
        }

async def get_consent_analytics(time_range: str = '30d') -> Dict[str, Any]:
    """API function to get consent analytics"""
    manager = PrivacyConsentManager("mongodb://localhost:27017", Fernet.generate_key())
    
    try:
        analytics = await manager.analytics_engine.generate_consent_analytics(time_range)
        return {
            'success': True,
            'analytics': analytics
        }
    except Exception as e:
        return {
            'success': False,
            'error': str(e)
        }

# Initialize logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

if __name__ == "__main__":
    logger.info("Privacy Consent Manager initialized successfully") 