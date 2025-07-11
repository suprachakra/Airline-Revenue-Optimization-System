"""
Enhanced Offer Model for IAROS 2.0
----------------------------------
Advanced offer creation with AI-powered personalization, channel differentiation, and real-time optimization.
Integrates with existing pricing service (142 scenarios) and ancillary service (110+ offerings).
"""

import uuid
import json
from datetime import datetime, timezone, timedelta
from typing import List, Dict, Optional, Enum, Union
from decimal import Decimal
from dataclasses import dataclass

class OfferStatus(Enum):
    """Offer lifecycle status."""
    DRAFT = "DRAFT"
    ACTIVE = "ACTIVE"
    EXPIRED = "EXPIRED"
    BOOKED = "BOOKED"
    CANCELLED = "CANCELLED"

class OfferType(Enum):
    """Types of offers."""
    FLIGHT_ONLY = "FLIGHT_ONLY"
    FLIGHT_PLUS_ANCILLARY = "FLIGHT_PLUS_ANCILLARY"
    BUNDLE = "BUNDLE"
    UPGRADE = "UPGRADE"
    PROMOTIONAL = "PROMOTIONAL"
    CORPORATE = "CORPORATE"
    GROUP = "GROUP"

class ChannelType(Enum):
    """Distribution channels."""
    DIRECT_WEBSITE = "DIRECT_WEBSITE"
    MOBILE_APP = "MOBILE_APP"
    GDS_AMADEUS = "GDS_AMADEUS"
    GDS_SABRE = "GDS_SABRE"
    GDS_TRAVELPORT = "GDS_TRAVELPORT"
    OTA_EXPEDIA = "OTA_EXPEDIA"
    OTA_BOOKING = "OTA_BOOKING"
    NDC_PARTNER = "NDC_PARTNER"
    TRAVEL_AGENT = "TRAVEL_AGENT"
    CORPORATE_PORTAL = "CORPORATE_PORTAL"

class PersonalizationLevel(Enum):
    """Levels of offer personalization."""
    NONE = "NONE"
    BASIC = "BASIC"
    ADVANCED = "ADVANCED"
    HYPER_PERSONALIZED = "HYPER_PERSONALIZED"

@dataclass
class PriceComponent:
    """Individual price component within an offer."""
    component_id: str
    component_type: str  # BASE_FARE, TAXES, FEES, ANCILLARY
    description: str
    amount: Decimal
    currency: str
    is_refundable: bool = False
    is_changeable: bool = False
    pricing_scenario: Optional[str] = None
    
    def to_dict(self) -> Dict:
        return {
            "component_id": self.component_id,
            "component_type": self.component_type,
            "description": self.description,
            "amount": str(self.amount),
            "currency": self.currency,
            "is_refundable": self.is_refundable,
            "is_changeable": self.is_changeable,
            "pricing_scenario": self.pricing_scenario
        }

@dataclass
class AncillaryService:
    """Ancillary service within an offer."""
    service_id: str
    service_type: str  # BAGGAGE, SEAT, MEAL, LOUNGE, etc.
    name: str
    description: str
    price: Decimal
    currency: str
    is_included: bool = False
    is_optional: bool = True
    quantity: int = 1
    metadata: Optional[Dict] = None
    personalization_score: float = 0.0
    
    def to_dict(self) -> Dict:
        return {
            "service_id": self.service_id,
            "service_type": self.service_type,
            "name": self.name,
            "description": self.description,
            "price": str(self.price),
            "currency": self.currency,
            "is_included": self.is_included,
            "is_optional": self.is_optional,
            "quantity": self.quantity,
            "metadata": self.metadata or {},
            "personalization_score": self.personalization_score
        }

@dataclass
class FlightSegment:
    """Flight segment information."""
    segment_id: str
    departure_airport: str
    arrival_airport: str
    departure_date: datetime
    arrival_date: datetime
    flight_number: str
    aircraft_type: str
    cabin_class: str
    booking_class: str
    available_seats: int
    
    def to_dict(self) -> Dict:
        return {
            "segment_id": self.segment_id,
            "departure_airport": self.departure_airport,
            "arrival_airport": self.arrival_airport,
            "departure_date": self.departure_date.isoformat(),
            "arrival_date": self.arrival_date.isoformat(),
            "flight_number": self.flight_number,
            "aircraft_type": self.aircraft_type,
            "cabin_class": self.cabin_class,
            "booking_class": self.booking_class,
            "available_seats": self.available_seats
        }

class RichContent:
    """Rich multimedia content for offers."""
    
    def __init__(self):
        self.images: List[Dict] = []
        self.videos: List[Dict] = []
        self.documents: List[Dict] = []
        self.cabin_tours: List[Dict] = []
        self.amenity_details: Dict = {}
        self.route_information: Dict = {}
        
    def add_image(self, image_url: str, alt_text: str, image_type: str = "PRODUCT"):
        """Add image to rich content."""
        self.images.append({
            "url": image_url,
            "alt_text": alt_text,
            "type": image_type,
            "timestamp": datetime.now(timezone.utc).isoformat()
        })
        
    def add_video(self, video_url: str, title: str, duration: int, video_type: str = "CABIN_TOUR"):
        """Add video to rich content."""
        self.videos.append({
            "url": video_url,
            "title": title,
            "duration": duration,
            "type": video_type,
            "timestamp": datetime.now(timezone.utc).isoformat()
        })
        
    def to_dict(self) -> Dict:
        return {
            "images": self.images,
            "videos": self.videos,
            "documents": self.documents,
            "cabin_tours": self.cabin_tours,
            "amenity_details": self.amenity_details,
            "route_information": self.route_information
        }

class PersonalizationMetadata:
    """Metadata for offer personalization."""
    
    def __init__(self):
        self.customer_segment = None
        self.personalization_level = PersonalizationLevel.NONE
        self.ml_model_version = None
        self.personalization_features: Dict = {}
        self.confidence_score = 0.0
        self.ab_test_variant = None
        self.recommendation_engine_used = False
        
    def update_personalization(self, customer_data: Dict, model_version: str):
        """Update personalization metadata."""
        self.customer_segment = customer_data.get('customer_segment')
        self.personalization_level = PersonalizationLevel.ADVANCED
        self.ml_model_version = model_version
        self.personalization_features = customer_data.get('features', {})
        self.recommendation_engine_used = True
        
    def to_dict(self) -> Dict:
        return {
            "customer_segment": self.customer_segment,
            "personalization_level": self.personalization_level.value,
            "ml_model_version": self.ml_model_version,
            "personalization_features": self.personalization_features,
            "confidence_score": self.confidence_score,
            "ab_test_variant": self.ab_test_variant,
            "recommendation_engine_used": self.recommendation_engine_used
        }

class Offer:
    """
    Enhanced Offer Model for IAROS 2.0
    ----------------------------------
    Comprehensive offer creation with AI-powered personalization, channel differentiation,
    and integration with existing IAROS pricing and ancillary services.
    """
    
    def __init__(self, flight_segments: List[FlightSegment], customer_id: Optional[str] = None,
                 channel: ChannelType = ChannelType.DIRECT_WEBSITE):
        # Core Offer Identity
        self.offer_id = str(uuid.uuid4())
        self.offer_reference = self._generate_offer_reference()
        self.customer_id = customer_id
        self.channel = channel
        
        # Offer Metadata
        self.offer_type = OfferType.FLIGHT_ONLY
        self.status = OfferStatus.DRAFT
        self.created_at = datetime.now(timezone.utc)
        self.expires_at = self.created_at + timedelta(minutes=30)  # Default 30min expiry
        self.last_updated = self.created_at
        self.version = 1
        
        # Flight Information
        self.flight_segments = flight_segments
        self.total_duration = self._calculate_total_duration()
        self.routing = self._generate_routing()
        
        # Pricing Information
        self.base_price = Decimal('0.00')
        self.total_price = Decimal('0.00')
        self.currency = "USD"
        self.price_components: List[PriceComponent] = []
        self.pricing_scenario_used = None
        self.dynamic_pricing_applied = False
        
        # Ancillary Services
        self.included_services: List[AncillaryService] = []
        self.optional_services: List[AncillaryService] = []
        self.recommended_services: List[AncillaryService] = []
        self.bundled_services: List[AncillaryService] = []
        
        # Personalization
        self.personalization_metadata = PersonalizationMetadata()
        self.willingness_to_pay_score = 0.5
        self.conversion_probability = 0.0
        
        # Channel Differentiation
        self.channel_specific_content: Dict = {}
        self.channel_pricing_applied = False
        
        # Rich Content
        self.rich_content = RichContent()
        
        # Business Rules
        self.minimum_connecting_time = 0
        self.maximum_stops = 2
        self.refund_policy = "STANDARD"
        self.change_policy = "STANDARD"
        
        # Analytics & Optimization
        self.view_count = 0
        self.last_viewed: Optional[datetime] = None
        self.conversion_events: List[Dict] = []
        self.ab_test_metadata: Dict = {}
        
        # Compliance & Validation
        self.fare_rules: Dict = {}
        self.booking_conditions: Dict = {}
        self.regulatory_info: Dict = {}
        
    def _generate_offer_reference(self) -> str:
        """Generate human-readable offer reference."""
        import random
        import string
        return 'OFF-' + ''.join(random.choices(string.ascii_uppercase + string.digits, k=8))
        
    def _calculate_total_duration(self) -> int:
        """Calculate total journey duration in minutes."""
        if not self.flight_segments:
            return 0
        
        departure = self.flight_segments[0].departure_date
        arrival = self.flight_segments[-1].arrival_date
        return int((arrival - departure).total_seconds() / 60)
        
    def _generate_routing(self) -> str:
        """Generate routing string (e.g., LAX-JFK-LHR)."""
        if not self.flight_segments:
            return ""
        
        airports = [self.flight_segments[0].departure_airport]
        for segment in self.flight_segments:
            airports.append(segment.arrival_airport)
        
        return "-".join(airports)
        
    def apply_dynamic_pricing(self, pricing_scenario: str, price_adjustment: Decimal):
        """Apply dynamic pricing from existing IAROS pricing service."""
        self.pricing_scenario_used = pricing_scenario
        self.dynamic_pricing_applied = True
        
        # Apply price adjustment
        original_price = self.base_price
        self.base_price += price_adjustment
        self.total_price = self._calculate_total_price()
        
        # Add price component
        price_component = PriceComponent(
            component_id=str(uuid.uuid4()),
            component_type="DYNAMIC_ADJUSTMENT",
            description=f"Dynamic pricing adjustment - {pricing_scenario}",
            amount=price_adjustment,
            currency=self.currency,
            pricing_scenario=pricing_scenario
        )
        self.price_components.append(price_component)
        
        self.last_updated = datetime.now(timezone.utc)
        
    def apply_channel_pricing(self, channel_modifier: float):
        """Apply channel-specific pricing."""
        if not self.channel_pricing_applied:
            adjustment = self.base_price * Decimal(str(channel_modifier - 1.0))
            self.base_price += adjustment
            self.total_price = self._calculate_total_price()
            self.channel_pricing_applied = True
            
            # Add price component
            price_component = PriceComponent(
                component_id=str(uuid.uuid4()),
                component_type="CHANNEL_ADJUSTMENT",
                description=f"Channel pricing - {self.channel.value}",
                amount=adjustment,
                currency=self.currency
            )
            self.price_components.append(price_component)
            
    def add_ancillary_service(self, service: AncillaryService, service_category: str = "OPTIONAL"):
        """Add ancillary service to offer."""
        if service_category == "INCLUDED":
            self.included_services.append(service)
        elif service_category == "OPTIONAL":
            self.optional_services.append(service)
        elif service_category == "RECOMMENDED":
            self.recommended_services.append(service)
        elif service_category == "BUNDLED":
            self.bundled_services.append(service)
            
        # If bundled or included, add to total price
        if service_category in ["BUNDLED", "INCLUDED"]:
            self.total_price += service.price
            
        self.last_updated = datetime.now(timezone.utc)
        
    def apply_personalization(self, customer_data: Dict, ml_model_version: str):
        """Apply AI-powered personalization to offer."""
        self.personalization_metadata.update_personalization(customer_data, ml_model_version)
        self.willingness_to_pay_score = customer_data.get('willingness_to_pay', 0.5)
        
        # Apply personalized ancillary recommendations
        self._generate_personalized_recommendations(customer_data)
        
        # Apply personalized pricing if appropriate
        if self.willingness_to_pay_score > 0.7:
            self._apply_premium_pricing()
        elif self.willingness_to_pay_score < 0.3:
            self._apply_discount_pricing()
            
        self.personalization_metadata.personalization_level = PersonalizationLevel.HYPER_PERSONALIZED
        
    def _generate_personalized_recommendations(self, customer_data: Dict):
        """Generate personalized ancillary recommendations."""
        customer_preferences = customer_data.get('preferences', {})
        
        # Example: Recommend seat selection based on preferences
        if 'seat_preference' in customer_preferences:
            seat_service = AncillaryService(
                service_id="SEAT_SELECTION",
                service_type="SEAT",
                name="Preferred Seat Selection",
                description="Choose your preferred seat",
                price=Decimal('25.00'),
                currency=self.currency,
                personalization_score=0.8
            )
            self.recommended_services.append(seat_service)
            
        # Example: Recommend baggage based on travel pattern
        if customer_data.get('travel_pattern') == 'BUSINESS':
            baggage_service = AncillaryService(
                service_id="EXTRA_BAGGAGE",
                service_type="BAGGAGE",
                name="Extra Baggage Allowance",
                description="Additional 23kg baggage",
                price=Decimal('45.00'),
                currency=self.currency,
                personalization_score=0.7
            )
            self.recommended_services.append(baggage_service)
            
    def _apply_premium_pricing(self):
        """Apply premium pricing for high WTP customers."""
        premium_adjustment = self.base_price * Decimal('0.1')  # 10% increase
        self.base_price += premium_adjustment
        self.total_price = self._calculate_total_price()
        
    def _apply_discount_pricing(self):
        """Apply discount pricing for price-sensitive customers."""
        discount_adjustment = self.base_price * Decimal('-0.05')  # 5% decrease
        self.base_price += discount_adjustment
        self.total_price = self._calculate_total_price()
        
    def _calculate_total_price(self) -> Decimal:
        """Calculate total offer price including all components."""
        total = self.base_price
        
        # Add included and bundled services
        for service in self.included_services + self.bundled_services:
            total += service.price
            
        # Add price components
        for component in self.price_components:
            total += component.amount
            
        return total
        
    def set_expiry(self, minutes: int):
        """Set offer expiry time."""
        self.expires_at = datetime.now(timezone.utc) + timedelta(minutes=minutes)
        
    def extend_expiry(self, additional_minutes: int):
        """Extend offer expiry time."""
        self.expires_at += timedelta(minutes=additional_minutes)
        
    def is_expired(self) -> bool:
        """Check if offer has expired."""
        return datetime.now(timezone.utc) > self.expires_at
        
    def mark_as_viewed(self):
        """Mark offer as viewed by customer."""
        self.view_count += 1
        self.last_viewed = datetime.now(timezone.utc)
        
    def track_conversion_event(self, event_type: str, metadata: Dict = None):
        """Track conversion events for analytics."""
        event = {
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "event_type": event_type,
            "metadata": metadata or {}
        }
        self.conversion_events.append(event)
        
    def book_offer(self):
        """Mark offer as booked."""
        if self.is_expired():
            raise ValueError("Cannot book expired offer")
            
        self.status = OfferStatus.BOOKED
        self.track_conversion_event("BOOKED")
        
    def calculate_conversion_probability(self) -> float:
        """Calculate conversion probability based on offer characteristics."""
        # Base probability
        probability = 0.1
        
        # Adjust based on personalization
        if self.personalization_metadata.personalization_level == PersonalizationLevel.HYPER_PERSONALIZED:
            probability += 0.3
        elif self.personalization_metadata.personalization_level == PersonalizationLevel.ADVANCED:
            probability += 0.2
            
        # Adjust based on willingness to pay alignment
        wtp_alignment = 1.0 - abs(self.willingness_to_pay_score - 0.5)
        probability += wtp_alignment * 0.2
        
        # Adjust based on channel
        channel_conversion_rates = {
            ChannelType.DIRECT_WEBSITE: 0.15,
            ChannelType.MOBILE_APP: 0.12,
            ChannelType.GDS_AMADEUS: 0.08,
            ChannelType.OTA_EXPEDIA: 0.10
        }
        probability += channel_conversion_rates.get(self.channel, 0.08)
        
        self.conversion_probability = min(1.0, probability)
        return self.conversion_probability
        
    def optimize_for_conversion(self):
        """Optimize offer for maximum conversion probability."""
        # Adjust pricing based on conversion probability
        current_probability = self.calculate_conversion_probability()
        
        if current_probability < 0.1:
            # Apply discount to increase conversion
            discount = self.base_price * Decimal('0.1')
            self.base_price -= discount
            self.total_price = self._calculate_total_price()
            
        # Add high-value, low-cost inclusions
        if len(self.included_services) == 0:
            priority_boarding = AncillaryService(
                service_id="PRIORITY_BOARDING",
                service_type="PRIORITY_BOARDING",
                name="Priority Boarding",
                description="Board first for a seamless travel experience",
                price=Decimal('0.00'),
                currency=self.currency,
                is_included=True
            )
            self.included_services.append(priority_boarding)
            
    def validate(self) -> bool:
        """Validate offer completeness and business rules."""
        errors = []
        
        if not self.flight_segments:
            errors.append("At least one flight segment is required")
            
        if self.base_price <= 0:
            errors.append("Base price must be greater than zero")
            
        if self.is_expired():
            errors.append("Offer has expired")
            
        # Validate flight segments
        for segment in self.flight_segments:
            if segment.available_seats <= 0:
                errors.append(f"No available seats for segment {segment.segment_id}")
                
        if errors:
            raise ValueError(f"Offer validation failed: {'; '.join(errors)}")
            
        return True
        
    def to_dict(self) -> Dict:
        """Convert offer to dictionary for API response."""
        return {
            # Core Identity
            "offer_id": self.offer_id,
            "offer_reference": self.offer_reference,
            "customer_id": self.customer_id,
            "channel": self.channel.value,
            
            # Metadata
            "offer_type": self.offer_type.value,
            "status": self.status.value,
            "created_at": self.created_at.isoformat(),
            "expires_at": self.expires_at.isoformat(),
            "last_updated": self.last_updated.isoformat(),
            "version": self.version,
            
            # Flight Information
            "flight_segments": [segment.to_dict() for segment in self.flight_segments],
            "total_duration": self.total_duration,
            "routing": self.routing,
            
            # Pricing
            "base_price": str(self.base_price),
            "total_price": str(self.total_price),
            "currency": self.currency,
            "price_components": [component.to_dict() for component in self.price_components],
            "pricing_scenario_used": self.pricing_scenario_used,
            "dynamic_pricing_applied": self.dynamic_pricing_applied,
            
            # Services
            "included_services": [service.to_dict() for service in self.included_services],
            "optional_services": [service.to_dict() for service in self.optional_services],
            "recommended_services": [service.to_dict() for service in self.recommended_services],
            "bundled_services": [service.to_dict() for service in self.bundled_services],
            
            # Personalization
            "personalization_metadata": self.personalization_metadata.to_dict(),
            "willingness_to_pay_score": self.willingness_to_pay_score,
            "conversion_probability": self.conversion_probability,
            
            # Rich Content
            "rich_content": self.rich_content.to_dict(),
            
            # Analytics
            "view_count": self.view_count,
            "last_viewed": self.last_viewed.isoformat() if self.last_viewed else None,
            
            # Policies
            "refund_policy": self.refund_policy,
            "change_policy": self.change_policy,
            
            # Validation
            "is_expired": self.is_expired(),
            "is_valid": True  # Would call self.validate() in production
        }
        
    def get_summary(self) -> Dict:
        """Get offer summary for quick reference."""
        return {
            "offer_id": self.offer_id,
            "offer_reference": self.offer_reference,
            "routing": self.routing,
            "total_price": str(self.total_price),
            "currency": self.currency,
            "expires_at": self.expires_at.isoformat(),
            "personalization_level": self.personalization_metadata.personalization_level.value,
            "conversion_probability": self.conversion_probability,
            "channel": self.channel.value,
            "status": self.status.value
        } 