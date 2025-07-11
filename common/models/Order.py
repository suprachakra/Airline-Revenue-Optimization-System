# Order.py
"""
IATA ONE Order Model
-------------------
Implements IATA ONE Order standard for unified order records, replacing traditional PNR/ET/EMD complexity.
Supports complete order lifecycle: creation, modification, cancellation, refund, and audit trail.
"""

import uuid
import json
from datetime import datetime, timezone
from typing import List, Dict, Optional, Enum
from decimal import Decimal
import re

class OrderStatus(Enum):
    """Order status following IATA ONE Order standard."""
    CREATED = "CREATED"
    CONFIRMED = "CONFIRMED"
    TICKETED = "TICKETED"
    FULFILLED = "FULFILLED"
    MODIFIED = "MODIFIED"
    CANCELLED = "CANCELLED"
    REFUNDED = "REFUNDED"
    EXPIRED = "EXPIRED"

class PaymentStatus(Enum):
    """Payment status for order financial tracking."""
    PENDING = "PENDING"
    AUTHORIZED = "AUTHORIZED"
    CAPTURED = "CAPTURED"
    FAILED = "FAILED"
    REFUNDED = "REFUNDED"
    PARTIALLY_REFUNDED = "PARTIALLY_REFUNDED"

class CurrencyCode(Enum):
    """ISO 4217 currency codes."""
    USD = "USD"
    EUR = "EUR"
    GBP = "GBP"
    AED = "AED"
    SGD = "SGD"
    JPY = "JPY"
    CAD = "CAD"
    AUD = "AUD"

class ServiceType(Enum):
    """Service types within an order."""
    FLIGHT = "FLIGHT"
    SEAT = "SEAT"
    BAGGAGE = "BAGGAGE"
    MEAL = "MEAL"
    LOUNGE = "LOUNGE"
    UPGRADE = "UPGRADE"
    INSURANCE = "INSURANCE"
    WIFI = "WIFI"
    PRIORITY_BOARDING = "PRIORITY_BOARDING"
    FAST_TRACK = "FAST_TRACK"

class OrderItem:
    """Individual item within an order."""
    
    def __init__(self, service_type: ServiceType, description: str, 
                 quantity: int, unit_price: Decimal, currency: CurrencyCode,
                 service_id: Optional[str] = None, metadata: Optional[Dict] = None):
        self.item_id = str(uuid.uuid4())
        self.service_type = service_type
        self.service_id = service_id
        self.description = description
        self.quantity = quantity
        self.unit_price = unit_price
        self.currency = currency
        self.total_price = unit_price * quantity
        self.metadata = metadata or {}
        self.created_at = datetime.now(timezone.utc)
        
    def to_dict(self) -> Dict:
        """Convert order item to dictionary for serialization."""
        return {
            "item_id": self.item_id,
            "service_type": self.service_type.value,
            "service_id": self.service_id,
            "description": self.description,
            "quantity": self.quantity,
            "unit_price": str(self.unit_price),
            "currency": self.currency.value,
            "total_price": str(self.total_price),
            "metadata": self.metadata,
            "created_at": self.created_at.isoformat()
        }

class PaymentMethod:
    """Payment method information."""
    
    def __init__(self, payment_type: str, last_four: str, 
                 expiry_month: Optional[int] = None, expiry_year: Optional[int] = None,
                 cardholder_name: Optional[str] = None):
        self.payment_type = payment_type  # e.g., "VISA", "MASTERCARD", "PAYPAL"
        self.last_four = last_four
        self.expiry_month = expiry_month
        self.expiry_year = expiry_year
        self.cardholder_name = cardholder_name
        
    def to_dict(self) -> Dict:
        """Convert payment method to dictionary."""
        return {
            "payment_type": self.payment_type,
            "last_four": self.last_four,
            "expiry_month": self.expiry_month,
            "expiry_year": self.expiry_year,
            "cardholder_name": self.cardholder_name
        }

class ContactInfo:
    """Customer contact information."""
    
    def __init__(self, email: str, phone: Optional[str] = None, 
                 address: Optional[Dict] = None):
        self.email = email
        self.phone = phone
        self.address = address or {}
        self.validate()
        
    def validate(self):
        """Validate contact information."""
        # Email validation
        email_pattern = r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
        if not re.match(email_pattern, self.email):
            raise ValueError(f"Invalid email format: {self.email}")
            
        # Phone validation (basic)
        if self.phone and not re.match(r'^\+?[\d\s\-\(\)]{10,}$', self.phone):
            raise ValueError(f"Invalid phone format: {self.phone}")
            
    def to_dict(self) -> Dict:
        """Convert contact info to dictionary."""
        return {
            "email": self.email,
            "phone": self.phone,
            "address": self.address
        }

class PassengerInfo:
    """Passenger information within an order."""
    
    def __init__(self, first_name: str, last_name: str, date_of_birth: str,
                 gender: str, nationality: str, passport_number: Optional[str] = None,
                 passenger_type: str = "ADULT", loyalty_number: Optional[str] = None):
        self.passenger_id = str(uuid.uuid4())
        self.first_name = first_name
        self.last_name = last_name
        self.date_of_birth = date_of_birth
        self.gender = gender
        self.nationality = nationality
        self.passport_number = passport_number
        self.passenger_type = passenger_type  # ADULT, CHILD, INFANT
        self.loyalty_number = loyalty_number
        self.validate()
        
    def validate(self):
        """Validate passenger information."""
        if not self.first_name or not self.last_name:
            raise ValueError("First name and last name are required")
            
        if self.passenger_type not in ["ADULT", "CHILD", "INFANT"]:
            raise ValueError(f"Invalid passenger type: {self.passenger_type}")
            
    def to_dict(self) -> Dict:
        """Convert passenger info to dictionary."""
        return {
            "passenger_id": self.passenger_id,
            "first_name": self.first_name,
            "last_name": self.last_name,
            "date_of_birth": self.date_of_birth,
            "gender": self.gender,
            "nationality": self.nationality,
            "passport_number": self.passport_number,
            "passenger_type": self.passenger_type,
            "loyalty_number": self.loyalty_number
        }

class Order:
    """
    IATA ONE Order Implementation
    ----------------------------
    Complete order record supporting the full customer journey from booking to fulfillment.
    Replaces traditional PNR/ET/EMD with a unified, versioned order record.
    """
    
    def __init__(self, customer_id: str, contact_info: ContactInfo,
                 passengers: List[PassengerInfo], channel: str = "DIRECT"):
        # Core Order Identifiers
        self.order_id = self._generate_order_id()
        self.customer_id = customer_id
        self.order_reference = self._generate_order_reference()
        
        # Order Metadata
        self.status = OrderStatus.CREATED
        self.channel = channel  # DIRECT, GDS, OTA, NDC
        self.created_at = datetime.now(timezone.utc)
        self.modified_at = self.created_at
        self.version = 1
        
        # Customer Information
        self.contact_info = contact_info
        self.passengers = passengers
        
        # Order Content
        self.items: List[OrderItem] = []
        self.total_amount = Decimal('0.00')
        self.currency = CurrencyCode.USD
        self.taxes = Decimal('0.00')
        self.fees = Decimal('0.00')
        
        # Payment Information
        self.payment_status = PaymentStatus.PENDING
        self.payment_method: Optional[PaymentMethod] = None
        self.payment_reference: Optional[str] = None
        
        # Fulfillment Information
        self.pnr_reference: Optional[str] = None
        self.ticket_numbers: List[str] = []
        self.booking_reference: Optional[str] = None
        
        # Audit Trail
        self.audit_trail: List[Dict] = []
        self.expires_at: Optional[datetime] = None
        
        # Additional Metadata
        self.metadata: Dict = {}
        
        # Initialize audit trail
        self._add_audit_entry("ORDER_CREATED", "Order created in IAROS")
        
    def _generate_order_id(self) -> str:
        """Generate unique order ID."""
        return f"ORD-{uuid.uuid4().hex[:12].upper()}"
        
    def _generate_order_reference(self) -> str:
        """Generate human-readable order reference."""
        import random
        import string
        return ''.join(random.choices(string.ascii_uppercase + string.digits, k=6))
        
    def _add_audit_entry(self, action: str, description: str, 
                        user_id: Optional[str] = None, metadata: Optional[Dict] = None):
        """Add entry to audit trail."""
        entry = {
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "action": action,
            "description": description,
            "user_id": user_id,
            "version": self.version,
            "metadata": metadata or {}
        }
        self.audit_trail.append(entry)
        
    def add_item(self, item: OrderItem) -> None:
        """Add item to order and recalculate totals."""
        self.items.append(item)
        self._recalculate_totals()
        self._add_audit_entry("ITEM_ADDED", f"Added {item.service_type.value}: {item.description}")
        
    def remove_item(self, item_id: str) -> bool:
        """Remove item from order and recalculate totals."""
        for i, item in enumerate(self.items):
            if item.item_id == item_id:
                removed_item = self.items.pop(i)
                self._recalculate_totals()
                self._add_audit_entry("ITEM_REMOVED", 
                                    f"Removed {removed_item.service_type.value}: {removed_item.description}")
                return True
        return False
        
    def _recalculate_totals(self) -> None:
        """Recalculate order totals."""
        self.total_amount = sum(item.total_price for item in self.items)
        self.modified_at = datetime.now(timezone.utc)
        
    def add_payment(self, payment_method: PaymentMethod, payment_reference: str) -> None:
        """Add payment information to order."""
        self.payment_method = payment_method
        self.payment_reference = payment_reference
        self.payment_status = PaymentStatus.AUTHORIZED
        self._add_audit_entry("PAYMENT_ADDED", f"Payment method added: {payment_method.payment_type}")
        
    def confirm_payment(self) -> None:
        """Confirm payment and update order status."""
        if self.payment_status == PaymentStatus.AUTHORIZED:
            self.payment_status = PaymentStatus.CAPTURED
            self.status = OrderStatus.CONFIRMED
            self._add_audit_entry("PAYMENT_CONFIRMED", "Payment confirmed and captured")
        else:
            raise ValueError(f"Cannot confirm payment. Current status: {self.payment_status}")
            
    def add_fulfillment_info(self, pnr_reference: str, 
                           ticket_numbers: List[str] = None) -> None:
        """Add fulfillment information (PNR, tickets)."""
        self.pnr_reference = pnr_reference
        if ticket_numbers:
            self.ticket_numbers.extend(ticket_numbers)
        self.status = OrderStatus.TICKETED
        self._add_audit_entry("FULFILLMENT_ADDED", 
                            f"PNR: {pnr_reference}, Tickets: {len(ticket_numbers or [])}")
        
    def cancel_order(self, reason: str, user_id: Optional[str] = None) -> None:
        """Cancel the order."""
        if self.status in [OrderStatus.CANCELLED, OrderStatus.REFUNDED]:
            raise ValueError(f"Order already {self.status.value}")
            
        self.status = OrderStatus.CANCELLED
        self.modified_at = datetime.now(timezone.utc)
        self.version += 1
        self._add_audit_entry("ORDER_CANCELLED", reason, user_id)
        
    def refund_order(self, refund_amount: Decimal, reason: str, 
                    user_id: Optional[str] = None) -> None:
        """Process order refund."""
        if self.status != OrderStatus.CANCELLED:
            raise ValueError("Order must be cancelled before refund")
            
        self.status = OrderStatus.REFUNDED
        self.modified_at = datetime.now(timezone.utc)
        self.version += 1
        
        if self.payment_status == PaymentStatus.CAPTURED:
            if refund_amount >= self.total_amount:
                self.payment_status = PaymentStatus.REFUNDED
            else:
                self.payment_status = PaymentStatus.PARTIALLY_REFUNDED
                
        self._add_audit_entry("ORDER_REFUNDED", 
                            f"Refund amount: {refund_amount} {self.currency.value}. Reason: {reason}", 
                            user_id)
        
    def modify_order(self, modifications: Dict, user_id: Optional[str] = None) -> None:
        """Modify order with version control."""
        self.version += 1
        self.modified_at = datetime.now(timezone.utc)
        self.status = OrderStatus.MODIFIED
        
        modification_details = json.dumps(modifications, default=str)
        self._add_audit_entry("ORDER_MODIFIED", modification_details, user_id)
        
    def validate(self) -> bool:
        """Validate order completeness and consistency."""
        errors = []
        
        # Basic validation
        if not self.customer_id:
            errors.append("Customer ID is required")
            
        if not self.passengers:
            errors.append("At least one passenger is required")
            
        if not self.items:
            errors.append("At least one order item is required")
            
        # Business rule validation
        if self.total_amount <= 0:
            errors.append("Order total must be greater than zero")
            
        # Payment validation for confirmed orders
        if self.status in [OrderStatus.CONFIRMED, OrderStatus.TICKETED, OrderStatus.FULFILLED]:
            if not self.payment_method or self.payment_status == PaymentStatus.PENDING:
                errors.append("Payment required for confirmed orders")
                
        if errors:
            raise ValueError(f"Order validation failed: {'; '.join(errors)}")
            
        return True
        
    def to_dict(self) -> Dict:
        """Convert order to dictionary for serialization."""
        return {
            # Core Identifiers
            "order_id": self.order_id,
            "customer_id": self.customer_id,
            "order_reference": self.order_reference,
            
            # Status and Metadata
            "status": self.status.value,
            "channel": self.channel,
            "created_at": self.created_at.isoformat(),
            "modified_at": self.modified_at.isoformat(),
            "version": self.version,
            
            # Customer Information
            "contact_info": self.contact_info.to_dict(),
            "passengers": [p.to_dict() for p in self.passengers],
            
            # Order Content
            "items": [item.to_dict() for item in self.items],
            "total_amount": str(self.total_amount),
            "currency": self.currency.value,
            "taxes": str(self.taxes),
            "fees": str(self.fees),
            
            # Payment Information
            "payment_status": self.payment_status.value,
            "payment_method": self.payment_method.to_dict() if self.payment_method else None,
            "payment_reference": self.payment_reference,
            
            # Fulfillment Information
            "pnr_reference": self.pnr_reference,
            "ticket_numbers": self.ticket_numbers,
            "booking_reference": self.booking_reference,
            
            # Audit and Metadata
            "audit_trail": self.audit_trail,
            "expires_at": self.expires_at.isoformat() if self.expires_at else None,
            "metadata": self.metadata
        }
        
    @classmethod
    def from_dict(cls, data: Dict) -> 'Order':
        """Create order from dictionary."""
        # This would be a complex deserialization method
        # Implementation would reconstruct all nested objects
        # For brevity, showing structure only
        pass
        
    def get_summary(self) -> Dict:
        """Get order summary for quick reference."""
        return {
            "order_id": self.order_id,
            "order_reference": self.order_reference,
            "status": self.status.value,
            "total_amount": str(self.total_amount),
            "currency": self.currency.value,
            "passenger_count": len(self.passengers),
            "item_count": len(self.items),
            "created_at": self.created_at.isoformat(),
            "channel": self.channel
        } 