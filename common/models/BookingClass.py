# BookingClass.py
"""
Enterprise Booking Class Management Model
----------------------------------------
Comprehensive booking class model with dynamic pricing, revenue optimization, inventory management,
fare rules, and advanced airline revenue management capabilities.
"""

import uuid
import json
from datetime import datetime, timezone, timedelta, date
from typing import List, Dict, Optional, Enum, Any, Tuple
from decimal import Decimal
from dataclasses import dataclass, field
import math
from enum import Enum

class FareType(Enum):
    """Fare type classifications."""
    PUBLISHED = "PUBLISHED"          # Published fares (public)
    PRIVATE = "PRIVATE"              # Private negotiated fares
    CORPORATE = "CORPORATE"          # Corporate contract fares
    PROMOTIONAL = "PROMOTIONAL"      # Promotional/discount fares
    GROUP = "GROUP"                  # Group booking fares
    TOUR = "TOUR"                    # Tour operator fares
    CONSOLIDATOR = "CONSOLIDATOR"    # Wholesale fares
    STUDENT = "STUDENT"              # Student discount fares
    SENIOR = "SENIOR"                # Senior citizen fares
    MILITARY = "MILITARY"            # Military fares

class CabinClass(Enum):
    """Aircraft cabin classes."""
    ECONOMY = "ECONOMY"
    PREMIUM_ECONOMY = "PREMIUM_ECONOMY"
    BUSINESS = "BUSINESS"
    FIRST = "FIRST"

class BookingChannel(Enum):
    """Distribution channels for bookings."""
    DIRECT_WEBSITE = "DIRECT_WEBSITE"
    MOBILE_APP = "MOBILE_APP"
    CALL_CENTER = "CALL_CENTER"
    TRAVEL_AGENT = "TRAVEL_AGENT"
    OTA = "OTA"                      # Online Travel Agency
    GDS = "GDS"                      # Global Distribution System
    NDC = "NDC"                      # New Distribution Capability
    PARTNER_AIRLINE = "PARTNER_AIRLINE"
    CORPORATE_BOOKING = "CORPORATE_BOOKING"

class RefundType(Enum):
    """Refund policy types."""
    NON_REFUNDABLE = "NON_REFUNDABLE"
    REFUNDABLE = "REFUNDABLE"
    PARTIALLY_REFUNDABLE = "PARTIALLY_REFUNDABLE"
    FLEXIBLE = "FLEXIBLE"

class ChangePolicy(Enum):
    """Change policy types."""
    NON_CHANGEABLE = "NON_CHANGEABLE"
    CHANGEABLE_WITH_FEE = "CHANGEABLE_WITH_FEE"
    FLEXIBLE_CHANGES = "FLEXIBLE_CHANGES"
    SAME_DAY_CHANGE = "SAME_DAY_CHANGE"

class SeasonType(Enum):
    """Seasonal pricing periods."""
    LOW_SEASON = "LOW_SEASON"
    SHOULDER_SEASON = "SHOULDER_SEASON"
    HIGH_SEASON = "HIGH_SEASON"
    PEAK_SEASON = "PEAK_SEASON"
    HOLIDAY_SEASON = "HOLIDAY_SEASON"

class DemandLevel(Enum):
    """Current demand levels for pricing."""
    VERY_LOW = "VERY_LOW"
    LOW = "LOW"
    MODERATE = "MODERATE"
    HIGH = "HIGH"
    VERY_HIGH = "VERY_HIGH"
    EXTREME = "EXTREME"

@dataclass
class FareRule:
    """Individual fare rule with conditions and penalties."""
    rule_id: str = field(default_factory=lambda: str(uuid.uuid4()))
    rule_type: str = ""              # REFUND, CHANGE, CANCELLATION, etc.
    description: str = ""
    penalty_amount: Decimal = field(default_factory=lambda: Decimal('0.00'))
    penalty_percentage: float = 0.0  # Percentage of fare
    minimum_penalty: Decimal = field(default_factory=lambda: Decimal('0.00'))
    maximum_penalty: Decimal = field(default_factory=lambda: Decimal('999999.00'))
    advance_purchase_required: int = 0  # days
    minimum_stay: Optional[int] = None  # days
    maximum_stay: Optional[int] = None  # days
    blackout_dates: List[date] = field(default_factory=list)
    valid_days_of_week: List[int] = field(default_factory=list)  # 0=Monday, 6=Sunday
    combinable_with_other_fares: bool = True
    endorsement_required: bool = False
    
    def calculate_penalty(self, base_fare: Decimal) -> Decimal:
        """Calculate penalty amount based on fare and rules."""
        if self.penalty_percentage > 0:
            percentage_penalty = base_fare * Decimal(str(self.penalty_percentage / 100))
            penalty = max(percentage_penalty, self.minimum_penalty)
        else:
            penalty = self.penalty_amount
        
        return min(penalty, self.maximum_penalty)

@dataclass
class InventoryControl:
    """Inventory management and availability control."""
    authorized_inventory: int = 0      # Total authorized seats
    available_inventory: int = 0       # Currently available seats
    sold_inventory: int = 0            # Seats sold
    blocked_inventory: int = 0         # Seats blocked for groups/holds
    waitlist_count: int = 0            # Waitlisted passengers
    overbooking_limit: int = 0         # Maximum overbooking allowed
    overbooking_current: int = 0       # Current overbooking level
    nest_level: int = 1                # Nested inventory level
    can_steal_from: List[str] = field(default_factory=list)  # Lower fare classes
    can_steal_to: List[str] = field(default_factory=list)    # Higher fare classes
    
    @property
    def load_factor(self) -> float:
        """Calculate load factor percentage."""
        return (self.sold_inventory / self.authorized_inventory * 100) if self.authorized_inventory > 0 else 0
    
    @property
    def availability_rate(self) -> float:
        """Calculate availability rate percentage."""
        return (self.available_inventory / self.authorized_inventory * 100) if self.authorized_inventory > 0 else 0
    
    def can_sell(self, requested_seats: int = 1) -> bool:
        """Check if requested seats can be sold."""
        return (self.available_inventory >= requested_seats and 
                (self.sold_inventory + self.overbooking_current + requested_seats) <= 
                (self.authorized_inventory + self.overbooking_limit))

@dataclass
class DynamicPricingRule:
    """Dynamic pricing rules and triggers."""
    rule_id: str = field(default_factory=lambda: str(uuid.uuid4()))
    name: str = ""
    trigger_condition: str = ""        # LOAD_FACTOR, TIME_TO_DEPARTURE, DEMAND_LEVEL, etc.
    trigger_threshold: float = 0.0     # Threshold value for trigger
    price_adjustment_type: str = "PERCENTAGE"  # PERCENTAGE, FIXED_AMOUNT
    price_adjustment_value: float = 0.0
    minimum_fare: Decimal = field(default_factory=lambda: Decimal('0.00'))
    maximum_fare: Decimal = field(default_factory=lambda: Decimal('999999.00'))
    active: bool = True
    priority: int = 1                  # Rule priority (1 = highest)
    valid_from: Optional[datetime] = None
    valid_to: Optional[datetime] = None
    
    def should_trigger(self, current_value: float) -> bool:
        """Check if rule should trigger based on current conditions."""
        if not self.active:
            return False
        
        now = datetime.now(timezone.utc)
        if self.valid_from and now < self.valid_from:
            return False
        if self.valid_to and now > self.valid_to:
            return False
        
        return current_value >= self.trigger_threshold
    
    def calculate_adjusted_price(self, base_price: Decimal) -> Decimal:
        """Calculate adjusted price based on rule."""
        if self.price_adjustment_type == "PERCENTAGE":
            adjusted_price = base_price * (1 + Decimal(str(self.price_adjustment_value / 100)))
        else:  # FIXED_AMOUNT
            adjusted_price = base_price + Decimal(str(self.price_adjustment_value))
        
        # Apply bounds
        adjusted_price = max(adjusted_price, self.minimum_fare)
        adjusted_price = min(adjusted_price, self.maximum_fare)
        
        return adjusted_price

@dataclass
class RevenueMetrics:
    """Revenue tracking and analytics for booking class."""
    total_revenue: Decimal = field(default_factory=lambda: Decimal('0.00'))
    average_fare: Decimal = field(default_factory=lambda: Decimal('0.00'))
    revenue_per_seat: Decimal = field(default_factory=lambda: Decimal('0.00'))
    yield_per_mile: Decimal = field(default_factory=lambda: Decimal('0.00'))
    passenger_revenue: Decimal = field(default_factory=lambda: Decimal('0.00'))
    ancillary_revenue: Decimal = field(default_factory=lambda: Decimal('0.00'))
    commission_paid: Decimal = field(default_factory=lambda: Decimal('0.00'))
    taxes_collected: Decimal = field(default_factory=lambda: Decimal('0.00'))
    net_revenue: Decimal = field(default_factory=lambda: Decimal('0.00'))
    bookings_count: int = 0
    cancellations_count: int = 0
    no_shows_count: int = 0
    
    def calculate_metrics(self):
        """Recalculate derived metrics."""
        if self.bookings_count > 0:
            self.average_fare = self.passenger_revenue / self.bookings_count
        
        self.net_revenue = (self.total_revenue - self.commission_paid)

@dataclass
class CompetitiveData:
    """Competitive intelligence for pricing optimization."""
    competitor_prices: Dict[str, Decimal] = field(default_factory=dict)  # airline_code: price
    market_position: str = "NEUTRAL"     # PREMIUM, COMPETITIVE, DISCOUNT
    price_index: float = 1.0             # Relative to market average (1.0 = at market)
    market_share: float = 0.0            # Percentage of market share
    last_updated: Optional[datetime] = None
    
    def get_market_average(self) -> Decimal:
        """Calculate market average price."""
        if not self.competitor_prices:
            return Decimal('0.00')
        return sum(self.competitor_prices.values()) / len(self.competitor_prices)
    
    def get_position_vs_market(self, our_price: Decimal) -> str:
        """Determine price position relative to market."""
        market_avg = self.get_market_average()
        if market_avg == 0:
            return "UNKNOWN"
        
        ratio = float(our_price / market_avg)
        if ratio <= 0.9:
            return "DISCOUNT"
        elif ratio <= 1.1:
            return "COMPETITIVE"
        else:
            return "PREMIUM"

class BookingClass:
    """Enterprise-grade booking class with comprehensive revenue optimization capabilities."""
    
    def __init__(self, class_code: str, class_name: str, cabin_class: CabinClass,
                 base_fare: Decimal, fare_type: FareType = FareType.PUBLISHED):
        
        # Core Identity
        self.booking_class_id = str(uuid.uuid4())
        self.class_code = class_code.upper()  # e.g., "Y", "B", "M", "H"
        self.class_name = class_name
        self.cabin_class = cabin_class
        self.fare_type = fare_type
        self.created_at = datetime.now(timezone.utc)
        self.updated_at = self.created_at
        
        # Pricing Core
        self.base_fare = base_fare
        self.current_fare = base_fare
        self.published_fare = base_fare
        self.net_fare = base_fare
        self.selling_fare = base_fare
        
        # Fare Rules & Policies
        self.fare_rules: List[FareRule] = []
        self.refund_type = RefundType.NON_REFUNDABLE
        self.change_policy = ChangePolicy.CHANGEABLE_WITH_FEE
        self.advance_purchase_days = 0
        self.minimum_stay_days: Optional[int] = None
        self.maximum_stay_days: Optional[int] = None
        self.saturday_night_stay_required = False
        
        # Inventory Management
        self.inventory = InventoryControl()
        self.nest_group: Optional[str] = None
        self.fare_basis = class_code
        self.booking_designator = class_code
        
        # Dynamic Pricing
        self.dynamic_pricing_enabled = True
        self.pricing_rules: List[DynamicPricingRule] = []
        self.price_elasticity = 1.0  # Price sensitivity factor
        self.demand_level = DemandLevel.MODERATE
        self.season_type = SeasonType.SHOULDER_SEASON
        
        # Distribution & Channel Management
        self.available_channels: List[BookingChannel] = [BookingChannel.DIRECT_WEBSITE]
        self.channel_markups: Dict[BookingChannel, float] = {}
        self.gds_enabled = True
        self.ndc_enabled = False
        
        # Revenue & Analytics
        self.revenue_metrics = RevenueMetrics()
        self.competitive_data = CompetitiveData()
        
        # Operational Data
        self.active = True
        self.display_priority = 1
        self.marketing_description = ""
        self.internal_notes = ""
        
        # Time-based Controls
        self.sale_start_date: Optional[datetime] = None
        self.sale_end_date: Optional[datetime] = None
        self.travel_start_date: Optional[date] = None
        self.travel_end_date: Optional[date] = None
        self.blackout_dates: List[date] = []
        
        # Yield Management
        self.yield_score = 0.0
        self.optimization_weight = 1.0
        self.revenue_contribution = 0.0
        self.demand_forecast = 0.0
        
        # Bundling & Ancillary
        self.included_services: List[str] = []
        self.excluded_services: List[str] = []
        self.ancillary_bundling_enabled = True
        self.upsell_target_classes: List[str] = []
        
        # Historical Performance
        self.historical_performance: Dict[str, Any] = {}
        self.pricing_history: List[Dict[str, Any]] = []
        
        # Initialize default fare rules
        self._initialize_default_fare_rules()
    
    def _initialize_default_fare_rules(self):
        """Initialize default fare rules based on cabin class and fare type."""
        # Refund rule
        if self.refund_type == RefundType.NON_REFUNDABLE:
            refund_rule = FareRule(
                rule_type="REFUND",
                description="Non-refundable fare - no refund allowed",
                penalty_percentage=100.0
            )
            self.fare_rules.append(refund_rule)
        
        # Change rule
        if self.change_policy == ChangePolicy.CHANGEABLE_WITH_FEE:
            change_fee = Decimal('150.00') if self.cabin_class == CabinClass.ECONOMY else Decimal('200.00')
            change_rule = FareRule(
                rule_type="CHANGE",
                description="Changes allowed with fee",
                penalty_amount=change_fee,
                minimum_penalty=change_fee
            )
            self.fare_rules.append(change_rule)
    
    def update_fare(self, new_fare: Decimal, reason: str = "", updated_by: str = "SYSTEM"):
        """Update current fare with audit trail."""
        old_fare = self.current_fare
        self.current_fare = new_fare
        self.selling_fare = new_fare
        self.updated_at = datetime.now(timezone.utc)
        
        # Record pricing history
        price_change = {
            "timestamp": self.updated_at.isoformat(),
            "old_fare": str(old_fare),
            "new_fare": str(new_fare),
            "change_amount": str(new_fare - old_fare),
            "change_percentage": float((new_fare - old_fare) / old_fare * 100) if old_fare > 0 else 0,
            "reason": reason,
            "updated_by": updated_by,
            "demand_level": self.demand_level.value,
            "load_factor": self.inventory.load_factor
        }
        
        self.pricing_history.append(price_change)
        
        # Keep only last 100 price changes
        if len(self.pricing_history) > 100:
            self.pricing_history = self.pricing_history[-100:]
    
    def apply_dynamic_pricing(self, flight_data: Dict[str, Any]) -> Decimal:
        """Apply dynamic pricing rules based on current conditions."""
        if not self.dynamic_pricing_enabled:
            return self.current_fare
        
        adjusted_fare = self.current_fare
        applied_rules = []
        
        # Sort rules by priority
        active_rules = [rule for rule in self.pricing_rules if rule.active]
        active_rules.sort(key=lambda x: x.priority)
        
        for rule in active_rules:
            trigger_value = self._get_trigger_value(rule.trigger_condition, flight_data)
            
            if rule.should_trigger(trigger_value):
                adjusted_fare = rule.calculate_adjusted_price(adjusted_fare)
                applied_rules.append({
                    "rule_name": rule.name,
                    "trigger_condition": rule.trigger_condition,
                    "trigger_value": trigger_value,
                    "adjustment": rule.price_adjustment_value
                })
        
        # Update fare if changes applied
        if applied_rules:
            reason = f"Dynamic pricing applied: {', '.join([r['rule_name'] for r in applied_rules])}"
            self.update_fare(adjusted_fare, reason, "DYNAMIC_PRICING_ENGINE")
        
        return adjusted_fare
    
    def _get_trigger_value(self, condition: str, flight_data: Dict[str, Any]) -> float:
        """Get current value for trigger condition."""
        if condition == "LOAD_FACTOR":
            return self.inventory.load_factor
        elif condition == "TIME_TO_DEPARTURE":
            departure_time = flight_data.get("scheduled_departure")
            if departure_time:
                if isinstance(departure_time, str):
                    departure_time = datetime.fromisoformat(departure_time.replace('Z', '+00:00'))
                hours_to_departure = (departure_time - datetime.now(timezone.utc)).total_seconds() / 3600
                return max(0, hours_to_departure)
        elif condition == "DEMAND_LEVEL":
            return float(list(DemandLevel).index(self.demand_level))
        elif condition == "AVAILABLE_SEATS":
            return float(self.inventory.available_inventory)
        elif condition == "DAYS_TO_DEPARTURE":
            departure_time = flight_data.get("scheduled_departure")
            if departure_time:
                if isinstance(departure_time, str):
                    departure_time = datetime.fromisoformat(departure_time.replace('Z', '+00:00'))
                days_to_departure = (departure_time.date() - datetime.now(timezone.utc).date()).days
                return max(0, days_to_departure)
        
        return 0.0
    
    def add_pricing_rule(self, rule: DynamicPricingRule):
        """Add dynamic pricing rule."""
        self.pricing_rules.append(rule)
        self.pricing_rules.sort(key=lambda x: x.priority)
        self.updated_at = datetime.now(timezone.utc)
    
    def remove_pricing_rule(self, rule_id: str):
        """Remove dynamic pricing rule."""
        self.pricing_rules = [rule for rule in self.pricing_rules if rule.rule_id != rule_id]
        self.updated_at = datetime.now(timezone.utc)
    
    def set_inventory(self, authorized: int, available: int = None, 
                     overbooking_limit: int = 0):
        """Set inventory levels."""
        self.inventory.authorized_inventory = authorized
        self.inventory.available_inventory = available if available is not None else authorized
        self.inventory.overbooking_limit = overbooking_limit
        self.updated_at = datetime.now(timezone.utc)
    
    def sell_seats(self, seats: int, channel: BookingChannel = BookingChannel.DIRECT_WEBSITE,
                   fare_paid: Optional[Decimal] = None) -> bool:
        """Process seat sale and update inventory."""
        if not self.inventory.can_sell(seats):
            return False
        
        self.inventory.available_inventory -= seats
        self.inventory.sold_inventory += seats
        
        # Update revenue metrics
        fare_amount = fare_paid or self.current_fare
        revenue = fare_amount * seats
        
        self.revenue_metrics.total_revenue += revenue
        self.revenue_metrics.passenger_revenue += revenue
        self.revenue_metrics.bookings_count += seats
        self.revenue_metrics.calculate_metrics()
        
        # Apply channel markup if applicable
        if channel in self.channel_markups:
            markup_amount = revenue * Decimal(str(self.channel_markups[channel] / 100))
            self.revenue_metrics.commission_paid += markup_amount
        
        self.updated_at = datetime.now(timezone.utc)
        return True
    
    def cancel_booking(self, seats: int, refund_amount: Decimal = None):
        """Process booking cancellation."""
        self.inventory.available_inventory += seats
        self.inventory.sold_inventory = max(0, self.inventory.sold_inventory - seats)
        
        self.revenue_metrics.cancellations_count += seats
        
        if refund_amount:
            self.revenue_metrics.total_revenue -= refund_amount
            self.revenue_metrics.passenger_revenue -= refund_amount
        
        self.revenue_metrics.calculate_metrics()
        self.updated_at = datetime.now(timezone.utc)
    
    def add_to_waitlist(self, seats: int):
        """Add passengers to waitlist."""
        self.inventory.waitlist_count += seats
        self.updated_at = datetime.now(timezone.utc)
    
    def clear_waitlist_seat(self):
        """Clear one seat from waitlist (passenger confirmed)."""
        if self.inventory.waitlist_count > 0:
            self.inventory.waitlist_count -= 1
            self.updated_at = datetime.now(timezone.utc)
    
    def calculate_yield(self, distance_miles: float) -> Decimal:
        """Calculate yield (revenue per passenger mile)."""
        if distance_miles > 0:
            yield_value = self.revenue_metrics.average_fare / Decimal(str(distance_miles))
            self.revenue_metrics.yield_per_mile = yield_value
            return yield_value
        return Decimal('0.00')
    
    def update_competitive_data(self, competitor_prices: Dict[str, Decimal]):
        """Update competitive pricing intelligence."""
        self.competitive_data.competitor_prices = competitor_prices
        self.competitive_data.last_updated = datetime.now(timezone.utc)
        self.competitive_data.market_position = self.competitive_data.get_position_vs_market(self.current_fare)
        
        # Update price index
        market_avg = self.competitive_data.get_market_average()
        if market_avg > 0:
            self.competitive_data.price_index = float(self.current_fare / market_avg)
    
    def optimize_pricing(self, market_conditions: Dict[str, Any]) -> Decimal:
        """Optimize pricing based on market conditions and revenue goals."""
        # Basic optimization algorithm
        target_load_factor = market_conditions.get("target_load_factor", 80.0)
        current_load_factor = self.inventory.load_factor
        price_elasticity = self.price_elasticity
        
        # If load factor is below target, consider price reduction
        if current_load_factor < target_load_factor * 0.8:
            # Reduce price to stimulate demand
            price_adjustment = -0.05 * (target_load_factor - current_load_factor) / target_load_factor
        elif current_load_factor > target_load_factor * 1.2:
            # Increase price due to high demand
            price_adjustment = 0.1 * (current_load_factor - target_load_factor) / target_load_factor
        else:
            price_adjustment = 0.0
        
        # Apply elasticity
        price_adjustment *= price_elasticity
        
        # Calculate optimized price
        optimized_price = self.current_fare * (1 + Decimal(str(price_adjustment)))
        
        # Apply bounds (minimum 50% of base fare, maximum 300% of base fare)
        min_price = self.base_fare * Decimal('0.5')
        max_price = self.base_fare * Decimal('3.0')
        optimized_price = max(min_price, min(max_price, optimized_price))
        
        return optimized_price
    
    def get_fare_quote(self, channel: BookingChannel = BookingChannel.DIRECT_WEBSITE,
                      passenger_type: str = "ADULT") -> Dict[str, Any]:
        """Get comprehensive fare quote with all fees and taxes."""
        base_fare = self.current_fare
        
        # Apply channel markup
        channel_markup = 0.0
        if channel in self.channel_markups:
            channel_markup = float(base_fare) * self.channel_markups[channel] / 100
        
        # Calculate taxes (simplified)
        tax_rate = 0.075  # 7.5% average tax rate
        taxes = base_fare * Decimal(str(tax_rate))
        
        # Calculate total fare
        total_fare = base_fare + Decimal(str(channel_markup)) + taxes
        
        return {
            "booking_class": self.class_code,
            "cabin_class": self.cabin_class.value,
            "base_fare": str(base_fare),
            "channel_markup": channel_markup,
            "taxes": str(taxes),
            "total_fare": str(total_fare),
            "fare_type": self.fare_type.value,
            "refund_type": self.refund_type.value,
            "change_policy": self.change_policy.value,
            "available_seats": self.inventory.available_inventory,
            "advance_purchase_required": self.advance_purchase_days,
            "fare_rules_count": len(self.fare_rules),
            "channel": channel.value,
            "valid_until": (datetime.now(timezone.utc) + timedelta(hours=24)).isoformat()
        }
    
    def calculate_change_fee(self, new_fare: Decimal) -> Decimal:
        """Calculate change fee for rebooking."""
        change_rules = [rule for rule in self.fare_rules if rule.rule_type == "CHANGE"]
        
        if not change_rules:
            return Decimal('0.00')
        
        change_rule = change_rules[0]  # Use first change rule
        return change_rule.calculate_penalty(self.current_fare)
    
    def calculate_cancellation_refund(self) -> Decimal:
        """Calculate refund amount for cancellation."""
        if self.refund_type == RefundType.NON_REFUNDABLE:
            return Decimal('0.00')
        
        refund_rules = [rule for rule in self.fare_rules if rule.rule_type == "REFUND"]
        
        if refund_rules:
            penalty = refund_rules[0].calculate_penalty(self.current_fare)
            return max(Decimal('0.00'), self.current_fare - penalty)
        
        return self.current_fare
    
    def get_performance_metrics(self) -> Dict[str, Any]:
        """Get comprehensive performance metrics."""
        return {
            "inventory": {
                "authorized": self.inventory.authorized_inventory,
                "available": self.inventory.available_inventory,
                "sold": self.inventory.sold_inventory,
                "load_factor": round(self.inventory.load_factor, 2),
                "waitlist": self.inventory.waitlist_count
            },
            "revenue": {
                "total_revenue": str(self.revenue_metrics.total_revenue),
                "average_fare": str(self.revenue_metrics.average_fare),
                "yield_per_mile": str(self.revenue_metrics.yield_per_mile),
                "bookings_count": self.revenue_metrics.bookings_count,
                "cancellation_rate": (self.revenue_metrics.cancellations_count / 
                                    max(1, self.revenue_metrics.bookings_count)) * 100
            },
            "pricing": {
                "current_fare": str(self.current_fare),
                "base_fare": str(self.base_fare),
                "price_changes_count": len(self.pricing_history),
                "dynamic_pricing_enabled": self.dynamic_pricing_enabled,
                "pricing_rules_count": len(self.pricing_rules)
            },
            "competitive": {
                "market_position": self.competitive_data.market_position,
                "price_index": round(self.competitive_data.price_index, 3),
                "competitors_tracked": len(self.competitive_data.competitor_prices)
            }
        }
    
    def to_dict(self, include_sensitive: bool = False) -> Dict[str, Any]:
        """Convert booking class to dictionary representation."""
        booking_class_dict = {
            "booking_class_id": self.booking_class_id,
            "class_code": self.class_code,
            "class_name": self.class_name,
            "cabin_class": self.cabin_class.value,
            "fare_type": self.fare_type.value,
            "current_fare": str(self.current_fare),
            "base_fare": str(self.base_fare),
            "refund_type": self.refund_type.value,
            "change_policy": self.change_policy.value,
            "active": self.active,
            "available_seats": self.inventory.available_inventory,
            "load_factor": round(self.inventory.load_factor, 2),
            "dynamic_pricing_enabled": self.dynamic_pricing_enabled,
            "demand_level": self.demand_level.value,
            "created_at": self.created_at.isoformat(),
            "updated_at": self.updated_at.isoformat()
        }
        
        if include_sensitive:
            booking_class_dict.update({
                "inventory": self.inventory.__dict__,
                "revenue_metrics": self.revenue_metrics.__dict__,
                "competitive_data": {
                    "market_position": self.competitive_data.market_position,
                    "price_index": self.competitive_data.price_index,
                    "competitor_count": len(self.competitive_data.competitor_prices)
                },
                "fare_rules": [rule.__dict__ for rule in self.fare_rules],
                "pricing_rules": [rule.__dict__ for rule in self.pricing_rules],
                "pricing_history": self.pricing_history[-10:],  # Last 10 changes
                "channel_markups": self.channel_markups,
                "available_channels": [ch.value for ch in self.available_channels]
            })
        
        return booking_class_dict
    
    def validate(self) -> bool:
        """Validate booking class data integrity."""
        # Basic validation
        if not self.class_code or not self.class_name:
            return False
        
        # Fare validation
        if self.current_fare < 0 or self.base_fare < 0:
            return False
        
        # Inventory validation
        if (self.inventory.authorized_inventory < 0 or 
            self.inventory.available_inventory < 0 or
            self.inventory.sold_inventory < 0):
            return False
        
        # Logical consistency
        if (self.inventory.sold_inventory + self.inventory.available_inventory + 
            self.inventory.blocked_inventory) > self.inventory.authorized_inventory:
            return False
        
        return True
    
    def __repr__(self) -> str:
        return (f"BookingClass(code='{self.class_code}', cabin='{self.cabin_class.value}', "
                f"fare={self.current_fare}, load_factor={self.inventory.load_factor:.1f}%)")
