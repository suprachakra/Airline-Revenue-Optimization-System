# Customer.py
"""
Customer 360° Model
------------------
Comprehensive customer profiling model for AI-powered personalization and dynamic offer creation.
Supports real-time behavioral analysis, preference learning, and GDPR-compliant data management.
"""

import uuid
import json
from datetime import datetime, timezone, date
from typing import List, Dict, Optional, Enum
from decimal import Decimal
import hashlib

class CustomerTier(Enum):
    """Customer loyalty tiers."""
    BASIC = "BASIC"
    SILVER = "SILVER"
    GOLD = "GOLD"
    PLATINUM = "PLATINUM"
    DIAMOND = "DIAMOND"

class PreferenceCategory(Enum):
    """Categories for customer preferences."""
    SEAT = "SEAT"
    MEAL = "MEAL"
    CABIN_CLASS = "CABIN_CLASS"
    AIRLINE = "AIRLINE"
    DESTINATION = "DESTINATION"
    PAYMENT = "PAYMENT"
    COMMUNICATION = "COMMUNICATION"
    ANCILLARY = "ANCILLARY"

class ChannelPreference(Enum):
    """Customer preferred channels."""
    DIRECT_WEBSITE = "DIRECT_WEBSITE"
    MOBILE_APP = "MOBILE_APP"
    PHONE = "PHONE"
    EMAIL = "EMAIL"
    CHAT = "CHAT"
    SOCIAL_MEDIA = "SOCIAL_MEDIA"

class TravelPattern(Enum):
    """Customer travel patterns."""
    LEISURE = "LEISURE"
    BUSINESS = "BUSINESS"
    MIXED = "MIXED"
    FREQUENT_SHORT_HAUL = "FREQUENT_SHORT_HAUL"
    OCCASIONAL_LONG_HAUL = "OCCASIONAL_LONG_HAUL"

class CustomerSegment(Enum):
    """AI-generated customer segments."""
    PRICE_SENSITIVE = "PRICE_SENSITIVE"
    CONVENIENCE_FOCUSED = "CONVENIENCE_FOCUSED"
    LUXURY_SEEKER = "LUXURY_SEEKER"
    FREQUENT_FLYER = "FREQUENT_FLYER"
    OCCASIONAL_TRAVELER = "OCCASIONAL_TRAVELER"
    BUSINESS_TRAVELER = "BUSINESS_TRAVELER"
    FAMILY_TRAVELER = "FAMILY_TRAVELER"

class CustomerPreference:
    """Individual customer preference with confidence scoring."""
    
    def __init__(self, category: PreferenceCategory, preference_key: str, 
                 preference_value: str, confidence_score: float = 0.5,
                 source: str = "IMPLICIT", last_updated: Optional[datetime] = None):
        self.preference_id = str(uuid.uuid4())
        self.category = category
        self.preference_key = preference_key
        self.preference_value = preference_value
        self.confidence_score = max(0.0, min(1.0, confidence_score))  # Clamp between 0-1
        self.source = source  # EXPLICIT, IMPLICIT, INFERRED
        self.created_at = datetime.now(timezone.utc)
        self.last_updated = last_updated or self.created_at
        self.update_count = 1
        
    def update_preference(self, new_value: str, confidence_score: float, source: str):
        """Update preference with new information."""
        self.preference_value = new_value
        self.confidence_score = max(0.0, min(1.0, confidence_score))
        self.source = source
        self.last_updated = datetime.now(timezone.utc)
        self.update_count += 1
        
    def to_dict(self) -> Dict:
        return {
            "preference_id": self.preference_id,
            "category": self.category.value,
            "preference_key": self.preference_key,
            "preference_value": self.preference_value,
            "confidence_score": self.confidence_score,
            "source": self.source,
            "created_at": self.created_at.isoformat(),
            "last_updated": self.last_updated.isoformat(),
            "update_count": self.update_count
        }

class BehavioralData:
    """Customer behavioral analytics data."""
    
    def __init__(self):
        self.session_data: List[Dict] = []
        self.click_patterns: Dict = {}
        self.search_history: List[Dict] = []
        self.conversion_funnel: Dict = {}
        self.abandonment_points: List[str] = []
        self.device_preferences: Dict = {}
        self.time_preferences: Dict = {}
        self.last_analyzed = datetime.now(timezone.utc)
        
    def add_session_data(self, session_data: Dict):
        """Add new session behavioral data."""
        session_data['timestamp'] = datetime.now(timezone.utc).isoformat()
        self.session_data.append(session_data)
        # Keep only last 100 sessions for performance
        if len(self.session_data) > 100:
            self.session_data = self.session_data[-100:]
            
    def update_click_patterns(self, element: str, action: str):
        """Track click patterns for UI optimization."""
        key = f"{element}_{action}"
        self.click_patterns[key] = self.click_patterns.get(key, 0) + 1
        
    def add_search_query(self, query_data: Dict):
        """Add search query to history."""
        query_data['timestamp'] = datetime.now(timezone.utc).isoformat()
        self.search_history.append(query_data)
        # Keep only last 50 searches
        if len(self.search_history) > 50:
            self.search_history = self.search_history[-50:]
            
    def to_dict(self) -> Dict:
        return {
            "session_data": self.session_data[-10:],  # Return only recent sessions
            "click_patterns": self.click_patterns,
            "search_history": self.search_history[-20:],  # Return recent searches
            "conversion_funnel": self.conversion_funnel,
            "abandonment_points": self.abandonment_points,
            "device_preferences": self.device_preferences,
            "time_preferences": self.time_preferences,
            "last_analyzed": self.last_analyzed.isoformat()
        }

class LoyaltyProfile:
    """Customer loyalty program profile."""
    
    def __init__(self, program_id: str, member_number: str, tier: CustomerTier,
                 points_balance: int = 0, tier_credits: int = 0):
        self.program_id = program_id
        self.member_number = member_number
        self.tier = tier
        self.points_balance = points_balance
        self.tier_credits = tier_credits
        self.joined_date = datetime.now(timezone.utc)
        self.last_activity = self.joined_date
        self.tier_expiry: Optional[datetime] = None
        self.lifetime_value = Decimal('0.00')
        self.redemption_history: List[Dict] = []
        
    def add_points(self, points: int, source: str, transaction_ref: str):
        """Add loyalty points."""
        self.points_balance += points
        self.last_activity = datetime.now(timezone.utc)
        
        transaction = {
            "timestamp": self.last_activity.isoformat(),
            "points": points,
            "source": source,
            "transaction_ref": transaction_ref,
            "balance_after": self.points_balance
        }
        self.redemption_history.append(transaction)
        
    def redeem_points(self, points: int, description: str, transaction_ref: str) -> bool:
        """Redeem loyalty points."""
        if self.points_balance >= points:
            self.points_balance -= points
            self.last_activity = datetime.now(timezone.utc)
            
            transaction = {
                "timestamp": self.last_activity.isoformat(),
                "points": -points,
                "description": description,
                "transaction_ref": transaction_ref,
                "balance_after": self.points_balance
            }
            self.redemption_history.append(transaction)
            return True
        return False
        
    def to_dict(self) -> Dict:
        return {
            "program_id": self.program_id,
            "member_number": self.member_number,
            "tier": self.tier.value,
            "points_balance": self.points_balance,
            "tier_credits": self.tier_credits,
            "joined_date": self.joined_date.isoformat(),
            "last_activity": self.last_activity.isoformat(),
            "tier_expiry": self.tier_expiry.isoformat() if self.tier_expiry else None,
            "lifetime_value": str(self.lifetime_value),
            "redemption_history": self.redemption_history[-10:]  # Recent transactions only
        }

class Customer:
    """
    Customer 360° Profile Model
    --------------------------
    Comprehensive customer model supporting AI-powered personalization, behavioral analysis,
    and GDPR-compliant data management for modern airline retailing.
    """
    
    def __init__(self, email: str, first_name: str, last_name: str,
                 phone: Optional[str] = None, date_of_birth: Optional[date] = None):
        # Core Identity
        self.customer_id = str(uuid.uuid4())
        self.email = email.lower().strip()
        self.first_name = first_name.strip()
        self.last_name = last_name.strip()
        self.phone = phone
        self.date_of_birth = date_of_birth
        
        # Profile Metadata
        self.created_at = datetime.now(timezone.utc)
        self.last_updated = self.created_at
        self.last_activity = self.created_at
        self.profile_completion = 0.0
        self.data_quality_score = 0.0
        
        # Customer Segmentation & Analysis
        self.customer_segment = CustomerSegment.OCCASIONAL_TRAVELER
        self.travel_pattern = TravelPattern.LEISURE
        self.customer_tier = CustomerTier.BASIC
        self.lifetime_value = Decimal('0.00')
        self.predicted_clv = Decimal('0.00')  # Customer Lifetime Value prediction
        self.churn_risk_score = 0.5  # 0 = low risk, 1 = high risk
        
        # Preferences & Personalization
        self.preferences: Dict[str, CustomerPreference] = {}
        self.communication_preferences: List[ChannelPreference] = [ChannelPreference.EMAIL]
        self.language_preference = "EN"
        self.currency_preference = "USD"
        self.timezone = "UTC"
        
        # Behavioral Analytics
        self.behavioral_data = BehavioralData()
        self.willingness_to_pay_score = 0.5  # 0 = price sensitive, 1 = price insensitive
        self.brand_loyalty_score = 0.5
        self.service_sensitivity_score = 0.5
        
        # Travel History & Patterns
        self.total_bookings = 0
        self.total_revenue = Decimal('0.00')
        self.average_trip_value = Decimal('0.00')
        self.booking_frequency = 0.0  # bookings per year
        self.preferred_destinations: List[str] = []
        self.seasonal_patterns: Dict = {}
        
        # Loyalty Programs
        self.loyalty_profiles: Dict[str, LoyaltyProfile] = {}
        
        # Privacy & Compliance
        self.consent_marketing = False
        self.consent_analytics = False
        self.consent_personalization = False
        self.gdpr_consent_date: Optional[datetime] = None
        self.data_retention_date: Optional[datetime] = None
        self.privacy_preferences: Dict = {}
        
        # Fraud & Risk
        self.fraud_score = 0.0  # 0 = no risk, 1 = high risk
        self.risk_flags: List[str] = []
        self.verification_status = "UNVERIFIED"  # UNVERIFIED, VERIFIED, BLOCKED
        
        # Customer Service
        self.support_tier = "STANDARD"  # STANDARD, PREMIUM, VIP
        self.support_history: List[Dict] = []
        self.satisfaction_score = 0.0
        self.nps_score: Optional[int] = None
        
        # AI Model Predictions
        self.ml_features: Dict = {}  # Features for ML models
        self.prediction_cache: Dict = {}  # Cached ML predictions
        self.last_model_update = self.created_at
        
    def add_preference(self, preference: CustomerPreference):
        """Add or update customer preference."""
        key = f"{preference.category.value}_{preference.preference_key}"
        
        if key in self.preferences:
            # Update existing preference with weighted confidence
            existing = self.preferences[key]
            new_confidence = (existing.confidence_score + preference.confidence_score) / 2
            existing.update_preference(preference.preference_value, new_confidence, preference.source)
        else:
            self.preferences[key] = preference
            
        self._update_profile_completion()
        self.last_updated = datetime.now(timezone.utc)
        
    def get_preference(self, category: PreferenceCategory, preference_key: str) -> Optional[CustomerPreference]:
        """Get customer preference by category and key."""
        key = f"{category.value}_{preference_key}"
        return self.preferences.get(key)
        
    def add_loyalty_profile(self, loyalty_profile: LoyaltyProfile):
        """Add loyalty program profile."""
        self.loyalty_profiles[loyalty_profile.program_id] = loyalty_profile
        
        # Update customer tier based on highest loyalty tier
        if loyalty_profile.tier.value in [tier.value for tier in CustomerTier]:
            current_tier_value = list(CustomerTier).index(self.customer_tier)
            new_tier_value = list(CustomerTier).index(loyalty_profile.tier)
            if new_tier_value > current_tier_value:
                self.customer_tier = loyalty_profile.tier
                
    def update_behavioral_data(self, session_data: Dict):
        """Update behavioral analytics data."""
        self.behavioral_data.add_session_data(session_data)
        self.last_activity = datetime.now(timezone.utc)
        
        # Trigger ML model updates if significant behavioral change
        if len(self.behavioral_data.session_data) % 10 == 0:
            self._schedule_ml_update()
            
    def calculate_willingness_to_pay(self, base_price: Decimal, 
                                   historical_purchases: List[Decimal]) -> float:
        """Calculate willingness to pay score based on historical data."""
        if not historical_purchases:
            return 0.5
            
        avg_historical = sum(historical_purchases) / len(historical_purchases)
        price_ratio = float(base_price / avg_historical) if avg_historical > 0 else 1.0
        
        # Adjust WTP based on customer tier and historical behavior
        tier_multiplier = {
            CustomerTier.BASIC: 0.8,
            CustomerTier.SILVER: 0.9,
            CustomerTier.GOLD: 1.1,
            CustomerTier.PLATINUM: 1.3,
            CustomerTier.DIAMOND: 1.5
        }
        
        adjusted_wtp = self.willingness_to_pay_score * tier_multiplier[self.customer_tier]
        return max(0.0, min(1.0, adjusted_wtp / price_ratio))
        
    def predict_churn_risk(self) -> float:
        """Predict customer churn risk based on behavioral patterns."""
        risk_factors = []
        
        # Recency of activity
        days_since_activity = (datetime.now(timezone.utc) - self.last_activity).days
        if days_since_activity > 365:
            risk_factors.append(0.3)
        elif days_since_activity > 180:
            risk_factors.append(0.2)
            
        # Booking frequency decline
        if self.booking_frequency < 1.0:  # Less than once per year
            risk_factors.append(0.2)
            
        # Support issues
        recent_support = [h for h in self.support_history 
                         if (datetime.now(timezone.utc) - 
                             datetime.fromisoformat(h.get('timestamp', '2020-01-01T00:00:00+00:00'))).days < 30]
        if len(recent_support) > 2:
            risk_factors.append(0.2)
            
        # Calculate weighted risk score
        self.churn_risk_score = min(1.0, sum(risk_factors))
        return self.churn_risk_score
        
    def update_customer_segment(self, new_segment: CustomerSegment, confidence: float = 0.8):
        """Update customer segment with ML model prediction."""
        if confidence > 0.7:  # Only update if confident
            self.customer_segment = new_segment
            self.ml_features['segment_confidence'] = confidence
            self.last_updated = datetime.now(timezone.utc)
            
    def _update_profile_completion(self):
        """Calculate profile completion percentage."""
        fields = [
            self.first_name, self.last_name, self.email, self.phone,
            self.date_of_birth, len(self.preferences) > 0,
            len(self.loyalty_profiles) > 0, self.consent_marketing
        ]
        completed_fields = sum(1 for field in fields if field)
        self.profile_completion = completed_fields / len(fields)
        
    def _schedule_ml_update(self):
        """Schedule ML model update for this customer."""
        self.prediction_cache.clear()  # Clear cached predictions
        self.last_model_update = datetime.now(timezone.utc)
        
    def update_privacy_consent(self, marketing: bool = False, 
                             analytics: bool = False, personalization: bool = False):
        """Update privacy consent preferences."""
        self.consent_marketing = marketing
        self.consent_analytics = analytics
        self.consent_personalization = personalization
        self.gdpr_consent_date = datetime.now(timezone.utc)
        
        # Set data retention date (7 years for airline data)
        from dateutil.relativedelta import relativedelta
        self.data_retention_date = self.gdpr_consent_date + relativedelta(years=7)
        
    def anonymize_data(self):
        """Anonymize customer data for GDPR compliance."""
        # Hash identifiable information
        self.email = hashlib.sha256(self.email.encode()).hexdigest()
        self.first_name = "ANONYMIZED"
        self.last_name = "ANONYMIZED"
        self.phone = None
        self.date_of_birth = None
        
        # Clear behavioral data
        self.behavioral_data = BehavioralData()
        self.support_history = []
        
        # Mark as anonymized
        self.verification_status = "ANONYMIZED"
        self.last_updated = datetime.now(timezone.utc)
        
    def get_personalization_features(self) -> Dict:
        """Get features for personalization algorithms."""
        return {
            "customer_id": self.customer_id,
            "customer_segment": self.customer_segment.value,
            "travel_pattern": self.travel_pattern.value,
            "customer_tier": self.customer_tier.value,
            "lifetime_value": float(self.lifetime_value),
            "willingness_to_pay": self.willingness_to_pay_score,
            "churn_risk": self.churn_risk_score,
            "booking_frequency": self.booking_frequency,
            "average_trip_value": float(self.average_trip_value),
            "profile_completion": self.profile_completion,
            "preferences_count": len(self.preferences),
            "loyalty_programs": len(self.loyalty_profiles),
            "communication_channels": [cp.value for cp in self.communication_preferences],
            "preferred_destinations": self.preferred_destinations[:5],  # Top 5
            "last_activity_days": (datetime.now(timezone.utc) - self.last_activity).days
        }
        
    def to_dict(self, include_sensitive: bool = False) -> Dict:
        """Convert customer to dictionary with privacy controls."""
        base_data = {
            # Core Identity (conditionally included)
            "customer_id": self.customer_id,
            "email": self.email if include_sensitive else "***@***.***",
            "first_name": self.first_name if include_sensitive else "***",
            "last_name": self.last_name if include_sensitive else "***",
            "phone": self.phone if include_sensitive else None,
            
            # Profile Metadata
            "created_at": self.created_at.isoformat(),
            "last_updated": self.last_updated.isoformat(),
            "last_activity": self.last_activity.isoformat(),
            "profile_completion": self.profile_completion,
            
            # Segmentation
            "customer_segment": self.customer_segment.value,
            "travel_pattern": self.travel_pattern.value,
            "customer_tier": self.customer_tier.value,
            "lifetime_value": str(self.lifetime_value),
            "churn_risk_score": self.churn_risk_score,
            
            # Preferences
            "preferences": {k: v.to_dict() for k, v in self.preferences.items()},
            "communication_preferences": [cp.value for cp in self.communication_preferences],
            "language_preference": self.language_preference,
            "currency_preference": self.currency_preference,
            
            # Travel Analytics
            "total_bookings": self.total_bookings,
            "booking_frequency": self.booking_frequency,
            "preferred_destinations": self.preferred_destinations,
            
            # Loyalty
            "loyalty_profiles": {k: v.to_dict() for k, v in self.loyalty_profiles.items()},
            
            # Privacy
            "consent_marketing": self.consent_marketing,
            "consent_analytics": self.consent_analytics,
            "consent_personalization": self.consent_personalization,
            
            # Service
            "support_tier": self.support_tier,
            "satisfaction_score": self.satisfaction_score,
            "verification_status": self.verification_status
        }
        
        # Include behavioral data only if analytics consent given
        if self.consent_analytics:
            base_data["behavioral_data"] = self.behavioral_data.to_dict()
            
        return base_data
        
    def validate(self) -> bool:
        """Validate customer data integrity."""
        errors = []
        
        if not self.email or '@' not in self.email:
            errors.append("Valid email is required")
            
        if not self.first_name or not self.last_name:
            errors.append("First name and last name are required")
            
        if self.willingness_to_pay_score < 0 or self.willingness_to_pay_score > 1:
            errors.append("Willingness to pay score must be between 0 and 1")
            
        if self.churn_risk_score < 0 or self.churn_risk_score > 1:
            errors.append("Churn risk score must be between 0 and 1")
            
        if errors:
            raise ValueError(f"Customer validation failed: {'; '.join(errors)}")
            
        return True 