# Flight.py
"""
Enterprise Flight Management Model
---------------------------------
Comprehensive flight model with network optimization, revenue management, operational tracking,
and real-time data integration for airline industry operations and revenue optimization.
"""

import uuid
import json
from datetime import datetime, timezone, timedelta, date, time
from typing import List, Dict, Optional, Enum, Any, Tuple
from decimal import Decimal
from dataclasses import dataclass, field
import math
from enum import Enum

class FlightStatus(Enum):
    """Flight operational status."""
    SCHEDULED = "SCHEDULED"
    BOARDING = "BOARDING"
    DEPARTED = "DEPARTED"
    EN_ROUTE = "EN_ROUTE"
    APPROACHING = "APPROACHING"
    LANDED = "LANDED"
    ARRIVED = "ARRIVED"
    DELAYED = "DELAYED"
    CANCELLED = "CANCELLED"
    DIVERTED = "DIVERTED"
    MAINTENANCE = "MAINTENANCE"

class AircraftType(Enum):
    """Common aircraft types with capacity data."""
    B737_800 = "B737_800"  # Boeing 737-800
    B737_MAX = "B737_MAX"  # Boeing 737 MAX
    A320 = "A320"          # Airbus A320
    A321 = "A321"          # Airbus A321
    B777_300 = "B777_300"  # Boeing 777-300
    B787_9 = "B787_9"      # Boeing 787-9
    A350_900 = "A350_900"  # Airbus A350-900
    B747_8 = "B747_8"      # Boeing 747-8
    A380 = "A380"          # Airbus A380
    E190 = "E190"          # Embraer E190

class CabinClass(Enum):
    """Aircraft cabin classes."""
    ECONOMY = "ECONOMY"
    PREMIUM_ECONOMY = "PREMIUM_ECONOMY"
    BUSINESS = "BUSINESS"
    FIRST = "FIRST"

class FlightType(Enum):
    """Flight operation types."""
    DOMESTIC = "DOMESTIC"
    INTERNATIONAL = "INTERNATIONAL"
    REGIONAL = "REGIONAL"
    CHARTER = "CHARTER"
    CARGO = "CARGO"
    FERRY = "FERRY"

class WeatherCondition(Enum):
    """Weather impact levels."""
    CLEAR = "CLEAR"
    LIGHT_TURBULENCE = "LIGHT_TURBULENCE"
    MODERATE_TURBULENCE = "MODERATE_TURBULENCE"
    SEVERE_TURBULENCE = "SEVERE_TURBULENCE"
    THUNDERSTORM = "THUNDERSTORM"
    FOG = "FOG"
    SNOW = "SNOW"
    WIND_SHEAR = "WIND_SHEAR"

class DelayCategory(Enum):
    """Delay categorization for analytics."""
    WEATHER = "WEATHER"
    AIR_TRAFFIC_CONTROL = "AIR_TRAFFIC_CONTROL"
    AIRCRAFT_TECHNICAL = "AIRCRAFT_TECHNICAL"
    CREW = "CREW"
    PASSENGER = "PASSENGER"
    AIRPORT_OPERATIONS = "AIRPORT_OPERATIONS"
    SECURITY = "SECURITY"
    FUELING = "FUELING"
    CATERING = "CATERING"
    BAGGAGE = "BAGGAGE"

@dataclass
class Airport:
    """Airport information with operational data."""
    code: str  # IATA code (e.g., "JFK")
    icao_code: str  # ICAO code (e.g., "KJFK")
    name: str
    city: str
    country: str
    timezone: str
    latitude: float
    longitude: float
    elevation: int  # feet above sea level
    runway_count: int
    terminal_count: int
    gate_count: int
    slot_controlled: bool = False  # Airport slot coordination required
    hub_airline: Optional[str] = None
    
    def calculate_distance_to(self, other_airport: 'Airport') -> float:
        """Calculate great circle distance to another airport in nautical miles."""
        lat1, lon1 = math.radians(self.latitude), math.radians(self.longitude)
        lat2, lon2 = math.radians(other_airport.latitude), math.radians(other_airport.longitude)
        
        dlat = lat2 - lat1
        dlon = lon2 - lon1
        
        a = math.sin(dlat/2)**2 + math.cos(lat1) * math.cos(lat2) * math.sin(dlon/2)**2
        c = 2 * math.asin(math.sqrt(a))
        
        # Earth radius in nautical miles
        earth_radius_nm = 3440.065
        return earth_radius_nm * c

@dataclass
class Aircraft:
    """Aircraft information and operational capabilities."""
    tail_number: str
    aircraft_type: AircraftType
    manufacturer: str
    model: str
    year_manufactured: int
    seat_configuration: Dict[CabinClass, int]
    total_seats: int
    max_range: int  # nautical miles
    cruise_speed: int  # knots
    fuel_capacity: int  # gallons
    max_takeoff_weight: int  # pounds
    maintenance_cycles: int = 0
    last_maintenance: Optional[datetime] = None
    next_maintenance_due: Optional[datetime] = None
    operational_status: str = "ACTIVE"
    wifi_enabled: bool = True
    entertainment_system: bool = True
    
    def calculate_fuel_consumption(self, distance_nm: float, load_factor: float = 0.8) -> float:
        """Calculate estimated fuel consumption for flight distance."""
        # Simplified fuel consumption model
        base_consumption = 0.8  # gallons per nautical mile per passenger
        efficiency_factor = {
            AircraftType.A320: 1.0,
            AircraftType.B737_800: 1.05,
            AircraftType.B787_9: 0.8,
            AircraftType.A350_900: 0.75,
            AircraftType.B777_300: 1.2,
            AircraftType.A380: 1.4
        }.get(self.aircraft_type, 1.0)
        
        passenger_count = self.total_seats * load_factor
        return distance_nm * passenger_count * base_consumption * efficiency_factor

@dataclass
class SeatMap:
    """Seat map configuration and availability."""
    cabin_class: CabinClass
    total_seats: int
    available_seats: int
    blocked_seats: int
    occupied_seats: int
    seat_pitch: int  # inches
    seat_width: int  # inches
    configuration: str  # e.g., "3-3" for economy, "2-2" for business
    premium_seats: List[str] = field(default_factory=list)  # seat numbers
    blocked_seat_list: List[str] = field(default_factory=list)
    
    @property
    def occupancy_rate(self) -> float:
        """Calculate cabin occupancy rate."""
        return (self.occupied_seats / self.total_seats) * 100 if self.total_seats > 0 else 0
    
    @property
    def availability_rate(self) -> float:
        """Calculate available seat percentage."""
        return (self.available_seats / self.total_seats) * 100 if self.total_seats > 0 else 0

@dataclass
class FlightLeg:
    """Individual flight leg in multi-segment journey."""
    leg_number: int
    departure_airport: Airport
    arrival_airport: Airport
    scheduled_departure: datetime
    scheduled_arrival: datetime
    actual_departure: Optional[datetime] = None
    actual_arrival: Optional[datetime] = None
    flight_time: timedelta = field(default_factory=timedelta)
    distance: float = 0.0  # nautical miles
    fuel_required: float = 0.0  # gallons
    
    def __post_init__(self):
        if self.distance == 0.0:
            self.distance = self.departure_airport.calculate_distance_to(self.arrival_airport)
        
        if self.flight_time == timedelta():
            self.flight_time = self.scheduled_arrival - self.scheduled_departure

@dataclass
class CrewAssignment:
    """Flight crew assignment information."""
    crew_id: str
    role: str  # CAPTAIN, FIRST_OFFICER, FLIGHT_ATTENDANT, etc.
    name: str
    certification_level: str
    flight_hours: int
    duty_start: datetime
    duty_end: datetime
    previous_flight: Optional[str] = None
    next_flight: Optional[str] = None

@dataclass
class WeatherData:
    """Weather information affecting flight operations."""
    location: str  # Airport code
    timestamp: datetime
    condition: WeatherCondition
    temperature: float  # Celsius
    wind_speed: int  # knots
    wind_direction: int  # degrees
    visibility: float  # miles
    ceiling: Optional[int] = None  # feet
    pressure: Optional[float] = None  # inches of mercury
    impact_score: float = 0.0  # 0.0 = no impact, 1.0 = severe impact

@dataclass
class OperationalDelay:
    """Delay tracking and categorization."""
    delay_id: str = field(default_factory=lambda: str(uuid.uuid4()))
    category: DelayCategory = DelayCategory.AIRCRAFT_TECHNICAL
    duration_minutes: int = 0
    description: str = ""
    cost_impact: Decimal = field(default_factory=lambda: Decimal('0.00'))
    passenger_impact: int = 0  # number of affected passengers
    timestamp: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    resolved: bool = False
    resolution_notes: str = ""

@dataclass
class RevenueData:
    """Flight revenue and profitability metrics."""
    total_revenue: Decimal = field(default_factory=lambda: Decimal('0.00'))
    passenger_revenue: Decimal = field(default_factory=lambda: Decimal('0.00'))
    cargo_revenue: Decimal = field(default_factory=lambda: Decimal('0.00'))
    ancillary_revenue: Decimal = field(default_factory=lambda: Decimal('0.00'))
    fuel_cost: Decimal = field(default_factory=lambda: Decimal('0.00'))
    crew_cost: Decimal = field(default_factory=lambda: Decimal('0.00'))
    airport_fees: Decimal = field(default_factory=lambda: Decimal('0.00'))
    maintenance_cost: Decimal = field(default_factory=lambda: Decimal('0.00'))
    total_cost: Decimal = field(default_factory=lambda: Decimal('0.00'))
    profit_margin: float = 0.0
    revenue_per_seat: Decimal = field(default_factory=lambda: Decimal('0.00'))
    cost_per_seat: Decimal = field(default_factory=lambda: Decimal('0.00'))
    break_even_load_factor: float = 0.0

class Flight:
    """Enterprise-grade flight model with comprehensive operational and revenue tracking."""
    
    def __init__(self, flight_number: str, airline_code: str, 
                 departure_airport: Airport, arrival_airport: Airport,
                 scheduled_departure: datetime, scheduled_arrival: datetime,
                 aircraft: Aircraft, flight_type: FlightType = FlightType.DOMESTIC):
        
        # Core Flight Identity
        self.flight_id = str(uuid.uuid4())
        self.flight_number = flight_number
        self.airline_code = airline_code
        self.flight_type = flight_type
        self.created_at = datetime.now(timezone.utc)
        self.updated_at = self.created_at
        
        # Flight Route
        self.departure_airport = departure_airport
        self.arrival_airport = arrival_airport
        self.legs: List[FlightLeg] = []
        
        # Create primary leg
        primary_leg = FlightLeg(
            leg_number=1,
            departure_airport=departure_airport,
            arrival_airport=arrival_airport,
            scheduled_departure=scheduled_departure,
            scheduled_arrival=scheduled_arrival
        )
        self.legs.append(primary_leg)
        
        # Timing
        self.scheduled_departure = scheduled_departure
        self.scheduled_arrival = scheduled_arrival
        self.actual_departure: Optional[datetime] = None
        self.actual_arrival: Optional[datetime] = None
        self.estimated_departure: Optional[datetime] = None
        self.estimated_arrival: Optional[datetime] = None
        
        # Aircraft & Configuration
        self.aircraft = aircraft
        self.seat_maps: Dict[CabinClass, SeatMap] = {}
        self._initialize_seat_maps()
        
        # Operational Status
        self.status = FlightStatus.SCHEDULED
        self.gate_departure: Optional[str] = None
        self.gate_arrival: Optional[str] = None
        self.terminal_departure: Optional[str] = None
        self.terminal_arrival: Optional[str] = None
        
        # Crew & Operations
        self.crew_assignments: List[CrewAssignment] = []
        self.captain_id: Optional[str] = None
        self.first_officer_id: Optional[str] = None
        
        # Environmental & External Factors
        self.weather_data: Dict[str, WeatherData] = {}  # keyed by airport code
        self.delays: List[OperationalDelay] = []
        self.total_delay_minutes = 0
        
        # Revenue & Performance
        self.revenue_data = RevenueData()
        self.load_factor = 0.0
        self.passenger_count = 0
        self.no_show_count = 0
        self.oversale_count = 0
        
        # Compliance & Regulatory
        self.flight_plan_filed = False
        self.customs_required = flight_type == FlightType.INTERNATIONAL
        self.slot_time: Optional[datetime] = None
        self.atc_clearance: Optional[str] = None
        
        # Ancillary Services
        self.catering_loaded = False
        self.fuel_loaded = False
        self.baggage_loaded = False
        self.cargo_loaded = False
        self.cleaning_completed = False
        
        # Real-time Tracking
        self.current_altitude: Optional[int] = None  # feet
        self.current_speed: Optional[int] = None  # knots
        self.current_location: Optional[Tuple[float, float]] = None  # lat, lon
        self.eta_updated_at: Optional[datetime] = None
        
        # Historical & Analytics
        self.historical_performance: Dict[str, Any] = {}
        self.passenger_feedback_score: Optional[float] = None
        self.on_time_performance_rate: float = 0.0
        
        # Calculate initial metrics
        self._calculate_distance_and_flight_time()
        self._estimate_fuel_requirements()
    
    def _initialize_seat_maps(self):
        """Initialize seat map configurations based on aircraft."""
        for cabin_class, seat_count in self.aircraft.seat_configuration.items():
            if seat_count > 0:
                seat_map = SeatMap(
                    cabin_class=cabin_class,
                    total_seats=seat_count,
                    available_seats=seat_count,
                    blocked_seats=0,
                    occupied_seats=0,
                    seat_pitch=self._get_seat_pitch(cabin_class),
                    seat_width=self._get_seat_width(cabin_class),
                    configuration=self._get_seat_configuration(cabin_class)
                )
                self.seat_maps[cabin_class] = seat_map
    
    def _get_seat_pitch(self, cabin_class: CabinClass) -> int:
        """Get typical seat pitch for cabin class."""
        pitch_map = {
            CabinClass.ECONOMY: 31,
            CabinClass.PREMIUM_ECONOMY: 36,
            CabinClass.BUSINESS: 60,
            CabinClass.FIRST: 78
        }
        return pitch_map.get(cabin_class, 31)
    
    def _get_seat_width(self, cabin_class: CabinClass) -> int:
        """Get typical seat width for cabin class."""
        width_map = {
            CabinClass.ECONOMY: 17,
            CabinClass.PREMIUM_ECONOMY: 18,
            CabinClass.BUSINESS: 21,
            CabinClass.FIRST: 24
        }
        return width_map.get(cabin_class, 17)
    
    def _get_seat_configuration(self, cabin_class: CabinClass) -> str:
        """Get typical seat configuration for cabin class."""
        config_map = {
            CabinClass.ECONOMY: "3-3",
            CabinClass.PREMIUM_ECONOMY: "3-3",
            CabinClass.BUSINESS: "2-2",
            CabinClass.FIRST: "1-1"
        }
        return config_map.get(cabin_class, "3-3")
    
    def _calculate_distance_and_flight_time(self):
        """Calculate total flight distance and time."""
        total_distance = 0.0
        total_time = timedelta()
        
        for leg in self.legs:
            total_distance += leg.distance
            total_time += leg.flight_time
        
        self.total_distance = total_distance
        self.total_flight_time = total_time
    
    def _estimate_fuel_requirements(self):
        """Estimate fuel requirements for flight."""
        if self.legs:
            primary_leg = self.legs[0]
            self.estimated_fuel = self.aircraft.calculate_fuel_consumption(
                primary_leg.distance, self.load_factor or 0.8
            )
    
    def update_status(self, new_status: FlightStatus, timestamp: Optional[datetime] = None):
        """Update flight operational status."""
        old_status = self.status
        self.status = new_status
        self.updated_at = timestamp or datetime.now(timezone.utc)
        
        # Update specific timestamps based on status
        if new_status == FlightStatus.DEPARTED and not self.actual_departure:
            self.actual_departure = self.updated_at
        elif new_status == FlightStatus.ARRIVED and not self.actual_arrival:
            self.actual_arrival = self.updated_at
        
        # Log status change in historical performance
        if 'status_changes' not in self.historical_performance:
            self.historical_performance['status_changes'] = []
        
        self.historical_performance['status_changes'].append({
            'from_status': old_status.value,
            'to_status': new_status.value,
            'timestamp': self.updated_at.isoformat(),
            'delay_minutes': self.total_delay_minutes
        })
    
    def add_delay(self, category: DelayCategory, duration_minutes: int, 
                  description: str = "", cost_impact: Decimal = None):
        """Add operational delay to flight."""
        delay = OperationalDelay(
            category=category,
            duration_minutes=duration_minutes,
            description=description,
            cost_impact=cost_impact or Decimal('0.00'),
            passenger_impact=self.passenger_count
        )
        
        self.delays.append(delay)
        self.total_delay_minutes += duration_minutes
        
        # Update estimated times
        if self.status in [FlightStatus.SCHEDULED, FlightStatus.BOARDING]:
            self.estimated_departure = self.scheduled_departure + timedelta(minutes=self.total_delay_minutes)
            self.estimated_arrival = self.scheduled_arrival + timedelta(minutes=self.total_delay_minutes)
        
        self.updated_at = datetime.now(timezone.utc)
    
    def assign_crew(self, crew_assignment: CrewAssignment):
        """Assign crew member to flight."""
        self.crew_assignments.append(crew_assignment)
        
        if crew_assignment.role == "CAPTAIN":
            self.captain_id = crew_assignment.crew_id
        elif crew_assignment.role == "FIRST_OFFICER":
            self.first_officer_id = crew_assignment.crew_id
        
        self.updated_at = datetime.now(timezone.utc)
    
    def update_weather(self, airport_code: str, weather_data: WeatherData):
        """Update weather data for departure or arrival airport."""
        self.weather_data[airport_code] = weather_data
        
        # Assess weather impact on delays
        if weather_data.impact_score > 0.5:  # Significant weather impact
            weather_delay_minutes = int(weather_data.impact_score * 60)  # Up to 60 minutes
            self.add_delay(
                DelayCategory.WEATHER,
                weather_delay_minutes,
                f"Weather delay due to {weather_data.condition.value}",
                Decimal(str(weather_delay_minutes * 150))  # $150 per minute cost
            )
    
    def update_passenger_count(self, boarded_passengers: int, no_shows: int = 0):
        """Update passenger boarding information."""
        self.passenger_count = boarded_passengers
        self.no_show_count = no_shows
        
        # Update seat occupancy
        remaining_passengers = boarded_passengers
        for cabin_class in [CabinClass.FIRST, CabinClass.BUSINESS, 
                           CabinClass.PREMIUM_ECONOMY, CabinClass.ECONOMY]:
            if cabin_class in self.seat_maps and remaining_passengers > 0:
                seat_map = self.seat_maps[cabin_class]
                occupied = min(remaining_passengers, seat_map.total_seats - seat_map.blocked_seats)
                seat_map.occupied_seats = occupied
                seat_map.available_seats = seat_map.total_seats - seat_map.blocked_seats - occupied
                remaining_passengers -= occupied
        
        # Calculate load factor
        total_available_seats = sum(sm.total_seats - sm.blocked_seats 
                                  for sm in self.seat_maps.values())
        self.load_factor = (boarded_passengers / total_available_seats) * 100 if total_available_seats > 0 else 0
        
        self.updated_at = datetime.now(timezone.utc)
    
    def block_seats(self, cabin_class: CabinClass, seat_numbers: List[str], reason: str = ""):
        """Block specific seats in cabin class."""
        if cabin_class in self.seat_maps:
            seat_map = self.seat_maps[cabin_class]
            seat_map.blocked_seat_list.extend(seat_numbers)
            seat_map.blocked_seats += len(seat_numbers)
            seat_map.available_seats = max(0, seat_map.available_seats - len(seat_numbers))
        
        self.updated_at = datetime.now(timezone.utc)
    
    def calculate_revenue_metrics(self):
        """Calculate comprehensive revenue and cost metrics."""
        # Revenue calculations
        total_seats = sum(sm.total_seats for sm in self.seat_maps.values())
        occupied_seats = sum(sm.occupied_seats for sm in self.seat_maps.values())
        
        if occupied_seats > 0:
            self.revenue_data.revenue_per_seat = self.revenue_data.passenger_revenue / occupied_seats
        
        if total_seats > 0:
            self.revenue_data.cost_per_seat = self.revenue_data.total_cost / total_seats
        
        # Profit margin calculation
        if self.revenue_data.total_revenue > 0:
            profit = self.revenue_data.total_revenue - self.revenue_data.total_cost
            self.revenue_data.profit_margin = float(profit / self.revenue_data.total_revenue) * 100
        
        # Break-even load factor
        if self.revenue_data.passenger_revenue > 0:
            avg_ticket_price = self.revenue_data.passenger_revenue / occupied_seats if occupied_seats > 0 else 0
            if avg_ticket_price > 0:
                break_even_passengers = float(self.revenue_data.total_cost / avg_ticket_price)
                self.revenue_data.break_even_load_factor = (break_even_passengers / total_seats) * 100 if total_seats > 0 else 0
    
    def update_real_time_position(self, latitude: float, longitude: float, 
                                 altitude: int, speed: int):
        """Update real-time flight tracking data."""
        self.current_location = (latitude, longitude)
        self.current_altitude = altitude
        self.current_speed = speed
        self.eta_updated_at = datetime.now(timezone.utc)
        
        # Estimate new arrival time based on current position and speed
        if self.status == FlightStatus.EN_ROUTE and self.current_speed and self.current_speed > 0:
            # Calculate remaining distance to destination
            if self.current_location:
                remaining_distance = self._calculate_remaining_distance()
                remaining_time_hours = remaining_distance / self.current_speed
                self.estimated_arrival = self.eta_updated_at + timedelta(hours=remaining_time_hours)
    
    def _calculate_remaining_distance(self) -> float:
        """Calculate remaining distance to destination airport."""
        if not self.current_location:
            return 0.0
        
        # Simplified great circle distance calculation
        lat1, lon1 = math.radians(self.current_location[0]), math.radians(self.current_location[1])
        lat2, lon2 = math.radians(self.arrival_airport.latitude), math.radians(self.arrival_airport.longitude)
        
        dlat = lat2 - lat1
        dlon = lon2 - lon1
        
        a = math.sin(dlat/2)**2 + math.cos(lat1) * math.cos(lat2) * math.sin(dlon/2)**2
        c = 2 * math.asin(math.sqrt(a))
        
        earth_radius_nm = 3440.065  # Earth radius in nautical miles
        return earth_radius_nm * c
    
    def calculate_on_time_performance(self) -> float:
        """Calculate on-time performance based on arrival delay."""
        if not self.actual_arrival or not self.scheduled_arrival:
            return 0.0
        
        delay_minutes = (self.actual_arrival - self.scheduled_arrival).total_seconds() / 60
        
        # On-time is considered within 15 minutes of scheduled arrival
        if delay_minutes <= 15:
            return 100.0
        else:
            # Graduated scoring based on delay severity
            if delay_minutes <= 30:
                return 80.0
            elif delay_minutes <= 60:
                return 60.0
            elif delay_minutes <= 120:
                return 40.0
            else:
                return 20.0
    
    def get_delay_analysis(self) -> Dict[str, Any]:
        """Provide comprehensive delay analysis."""
        total_delays = len(self.delays)
        if total_delays == 0:
            return {"total_delays": 0, "total_minutes": 0, "categories": {}}
        
        category_breakdown = {}
        total_cost_impact = Decimal('0.00')
        
        for delay in self.delays:
            category = delay.category.value
            if category not in category_breakdown:
                category_breakdown[category] = {
                    "count": 0,
                    "total_minutes": 0,
                    "cost_impact": Decimal('0.00')
                }
            
            category_breakdown[category]["count"] += 1
            category_breakdown[category]["total_minutes"] += delay.duration_minutes
            category_breakdown[category]["cost_impact"] += delay.cost_impact
            total_cost_impact += delay.cost_impact
        
        return {
            "total_delays": total_delays,
            "total_minutes": self.total_delay_minutes,
            "total_cost_impact": str(total_cost_impact),
            "categories": category_breakdown,
            "average_delay_minutes": self.total_delay_minutes / total_delays,
            "most_common_category": max(category_breakdown.keys(), 
                                      key=lambda k: category_breakdown[k]["count"]) if category_breakdown else None
        }
    
    def get_operational_summary(self) -> Dict[str, Any]:
        """Get comprehensive operational summary."""
        return {
            "flight_info": {
                "flight_id": self.flight_id,
                "flight_number": self.flight_number,
                "airline_code": self.airline_code,
                "route": f"{self.departure_airport.code}-{self.arrival_airport.code}",
                "aircraft_type": self.aircraft.aircraft_type.value,
                "status": self.status.value
            },
            "timing": {
                "scheduled_departure": self.scheduled_departure.isoformat(),
                "scheduled_arrival": self.scheduled_arrival.isoformat(),
                "actual_departure": self.actual_departure.isoformat() if self.actual_departure else None,
                "actual_arrival": self.actual_arrival.isoformat() if self.actual_arrival else None,
                "total_delay_minutes": self.total_delay_minutes,
                "on_time_performance": self.calculate_on_time_performance()
            },
            "passengers": {
                "passenger_count": self.passenger_count,
                "load_factor": round(self.load_factor, 2),
                "no_show_count": self.no_show_count,
                "total_capacity": sum(sm.total_seats for sm in self.seat_maps.values())
            },
            "revenue": {
                "total_revenue": str(self.revenue_data.total_revenue),
                "total_cost": str(self.revenue_data.total_cost),
                "profit_margin": round(self.revenue_data.profit_margin, 2),
                "revenue_per_seat": str(self.revenue_data.revenue_per_seat),
                "break_even_load_factor": round(self.revenue_data.break_even_load_factor, 2)
            },
            "operational": {
                "crew_assigned": len(self.crew_assignments),
                "weather_reports": len(self.weather_data),
                "gate_departure": self.gate_departure,
                "gate_arrival": self.gate_arrival,
                "fuel_loaded": self.fuel_loaded,
                "catering_loaded": self.catering_loaded
            }
        }
    
    def to_dict(self, include_sensitive: bool = False) -> Dict[str, Any]:
        """Convert flight to dictionary representation."""
        flight_dict = {
            "flight_id": self.flight_id,
            "flight_number": self.flight_number,
            "airline_code": self.airline_code,
            "flight_type": self.flight_type.value,
            "status": self.status.value,
            "departure_airport": {
                "code": self.departure_airport.code,
                "name": self.departure_airport.name,
                "city": self.departure_airport.city
            },
            "arrival_airport": {
                "code": self.arrival_airport.code,
                "name": self.arrival_airport.name,
                "city": self.arrival_airport.city
            },
            "scheduled_departure": self.scheduled_departure.isoformat(),
            "scheduled_arrival": self.scheduled_arrival.isoformat(),
            "actual_departure": self.actual_departure.isoformat() if self.actual_departure else None,
            "actual_arrival": self.actual_arrival.isoformat() if self.actual_arrival else None,
            "aircraft": {
                "tail_number": self.aircraft.tail_number,
                "type": self.aircraft.aircraft_type.value,
                "total_seats": self.aircraft.total_seats
            },
            "passenger_count": self.passenger_count,
            "load_factor": round(self.load_factor, 2),
            "total_delay_minutes": self.total_delay_minutes,
            "gate_departure": self.gate_departure,
            "gate_arrival": self.gate_arrival,
            "created_at": self.created_at.isoformat(),
            "updated_at": self.updated_at.isoformat()
        }
        
        if include_sensitive:
            flight_dict.update({
                "seat_maps": {cls.value: sm.__dict__ for cls, sm in self.seat_maps.items()},
                "crew_assignments": [crew.__dict__ for crew in self.crew_assignments],
                "delays": [delay.__dict__ for delay in self.delays],
                "revenue_data": self.revenue_data.__dict__,
                "weather_data": {k: v.__dict__ for k, v in self.weather_data.items()},
                "current_location": self.current_location,
                "current_altitude": self.current_altitude,
                "current_speed": self.current_speed,
                "historical_performance": self.historical_performance
            })
        
        return flight_dict
    
    def validate(self) -> bool:
        """Validate flight data integrity."""
        # Basic validation
        if not self.flight_number or not self.airline_code:
            return False
        
        # Time validation
        if self.scheduled_departure >= self.scheduled_arrival:
            return False
        
        # Airport validation
        if not self.departure_airport or not self.arrival_airport:
            return False
        
        # Aircraft validation
        if not self.aircraft or self.aircraft.total_seats <= 0:
            return False
        
        return True
    
    def __repr__(self) -> str:
        return (f"Flight(id='{self.flight_id}', number='{self.flight_number}', "
                f"route='{self.departure_airport.code}-{self.arrival_airport.code}', "
                f"status='{self.status.value}', load_factor={self.load_factor:.1f}%)")
