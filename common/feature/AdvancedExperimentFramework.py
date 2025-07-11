# AdvancedExperimentFramework.py
"""
Advanced Personalization Experimentation Framework
==================================================
Extends the basic A/B testing framework with advanced personalization capabilities,
multi-armed bandits, advanced targeting, and comprehensive analytics.
"""

import random
import json
import time
import hashlib
from typing import Dict, List, Optional, Any, Callable
from datetime import datetime, timedelta
from dataclasses import dataclass, field
from enum import Enum
import logging

from .ExperimentFramework import Experiment as BaseExperiment

# Configure logging
logger = logging.getLogger(__name__)

class ExperimentType(Enum):
    AB_TEST = "ab_test"
    MULTIVARIATE = "multivariate" 
    MULTI_ARMED_BANDIT = "multi_armed_bandit"
    FACTORIAL = "factorial"
    PERSONALIZED = "personalized"

class TargetingCriteria(Enum):
    ALL_USERS = "all_users"
    NEW_USERS = "new_users"
    RETURNING_USERS = "returning_users"
    LOYALTY_TIER = "loyalty_tier"
    GEOGRAPHIC = "geographic"
    BEHAVIORAL = "behavioral"
    DEMOGRAPHIC = "demographic"
    CUSTOM = "custom"

class ExperimentStatus(Enum):
    DRAFT = "draft"
    SCHEDULED = "scheduled"
    RUNNING = "running"
    PAUSED = "paused"
    COMPLETED = "completed"
    CANCELLED = "cancelled"

@dataclass
class UserContext:
    """Rich user context for personalization"""
    user_id: str
    session_id: str
    
    # Demographics
    age_group: Optional[str] = None
    gender: Optional[str] = None
    location: Optional[str] = None
    
    # Behavioral
    loyalty_tier: Optional[str] = None
    booking_frequency: Optional[str] = None
    average_spend: Optional[float] = None
    preferred_class: Optional[str] = None
    
    # Contextual
    device_type: Optional[str] = None
    channel: Optional[str] = None
    time_of_day: Optional[str] = None
    season: Optional[str] = None
    
    # Custom attributes
    custom_attributes: Dict[str, Any] = field(default_factory=dict)
    
    # Historical behavior
    past_experiments: List[str] = field(default_factory=list)
    conversion_history: List[Dict] = field(default_factory=list)

@dataclass
class VariantConfiguration:
    """Configuration for experiment variants"""
    id: str
    name: str
    description: str
    weight: float  # Traffic allocation percentage
    config: Dict[str, Any]
    
    # Targeting
    targeting_rules: List[Dict] = field(default_factory=list)
    exclusion_rules: List[Dict] = field(default_factory=list)
    
    # Performance tracking
    conversions: int = 0
    impressions: int = 0
    revenue: float = 0.0
    
    # Multi-armed bandit
    reward_sum: float = 0.0
    arm_pulls: int = 0
    confidence: float = 0.0

@dataclass 
class ExperimentMetrics:
    """Comprehensive experiment metrics"""
    conversion_rate: float = 0.0
    revenue_per_user: float = 0.0
    statistical_significance: float = 0.0
    confidence_interval: tuple = (0.0, 0.0)
    
    # Advanced metrics
    lift: float = 0.0
    engagement_rate: float = 0.0
    retention_rate: float = 0.0
    customer_lifetime_value: float = 0.0
    
    # Business metrics
    booking_completion_rate: float = 0.0
    average_order_value: float = 0.0
    ancillary_revenue: float = 0.0
    customer_satisfaction: float = 0.0

class AdvancedExperiment:
    """Advanced experiment with personalization and multi-armed bandit capabilities"""
    
    def __init__(self, 
                 experiment_id: str,
                 name: str,
                 description: str,
                 experiment_type: ExperimentType,
                 variants: List[VariantConfiguration],
                 targeting_criteria: Dict[str, Any] = None,
                 duration_days: int = 14):
        
        self.experiment_id = experiment_id
        self.name = name
        self.description = description
        self.experiment_type = experiment_type
        self.variants = {v.id: v for v in variants}
        self.targeting_criteria = targeting_criteria or {}
        self.duration_days = duration_days
        
        # Status tracking
        self.status = ExperimentStatus.DRAFT
        self.start_time: Optional[datetime] = None
        self.end_time: Optional[datetime] = None
        
        # Traffic allocation
        self.traffic_allocation = 1.0  # Percentage of users in experiment
        self.minimum_sample_size = 1000
        
        # Multi-armed bandit configuration
        self.bandit_config = {
            "exploration_rate": 0.1,
            "confidence_threshold": 0.95,
            "update_frequency": 3600  # seconds
        }
        
        # Personalization engine
        self.personalization_model = None
        self.segment_strategies: Dict[str, str] = {}
        
        # Safety mechanisms
        self.guardrails = {
            "max_conversion_drop": 0.05,  # 5% max drop allowed
            "min_statistical_power": 0.8,
            "auto_stop_on_significance": True
        }
        
        # Results storage
        self.participant_assignments: Dict[str, str] = {}
        self.conversion_events: List[Dict] = []
        self.metrics_history: List[ExperimentMetrics] = []
        
        logger.info(f"Advanced experiment created: {self.experiment_id}")

    def start_experiment(self) -> bool:
        """Start the experiment with validation"""
        try:
            # Validate experiment configuration
            if not self._validate_configuration():
                return False
            
            # Initialize tracking
            self.status = ExperimentStatus.RUNNING
            self.start_time = datetime.now()
            self.end_time = self.start_time + timedelta(days=self.duration_days)
            
            # Reset variant counters
            for variant in self.variants.values():
                variant.impressions = 0
                variant.conversions = 0
                variant.revenue = 0.0
                variant.arm_pulls = 0
                variant.reward_sum = 0.0
            
            logger.info(f"Experiment started: {self.experiment_id}")
            return True
            
        except Exception as e:
            logger.error(f"Failed to start experiment {self.experiment_id}: {e}")
            return False

    def assign_variant(self, user_context: UserContext) -> Optional[str]:
        """Assign user to variant with advanced targeting and personalization"""
        try:
            # Check if experiment is running
            if self.status != ExperimentStatus.RUNNING:
                return None
            
            # Check if user is in experiment population
            if not self._is_user_eligible(user_context):
                return None
            
            # Check traffic allocation
            if not self._is_in_traffic_allocation(user_context):
                return None
            
            # Get variant assignment based on experiment type
            variant_id = self._get_variant_assignment(user_context)
            
            if variant_id:
                # Track assignment
                self.participant_assignments[user_context.user_id] = variant_id
                self.variants[variant_id].impressions += 1
                
                # Multi-armed bandit tracking
                if self.experiment_type == ExperimentType.MULTI_ARMED_BANDIT:
                    self.variants[variant_id].arm_pulls += 1
                
                logger.debug(f"User {user_context.user_id} assigned to variant {variant_id}")
            
            return variant_id
            
        except Exception as e:
            logger.error(f"Variant assignment failed for user {user_context.user_id}: {e}")
            return None

    def track_conversion(self, user_id: str, conversion_type: str, value: float = 1.0, metadata: Dict = None) -> bool:
        """Track conversion event with value"""
        try:
            # Check if user is in experiment
            if user_id not in self.participant_assignments:
                return False
            
            variant_id = self.participant_assignments[user_id]
            variant = self.variants[variant_id]
            
            # Record conversion
            conversion_event = {
                "user_id": user_id,
                "variant_id": variant_id,
                "conversion_type": conversion_type,
                "value": value,
                "timestamp": datetime.now().isoformat(),
                "metadata": metadata or {}
            }
            
            self.conversion_events.append(conversion_event)
            
            # Update variant metrics
            variant.conversions += 1
            variant.revenue += value
            
            # Multi-armed bandit reward update
            if self.experiment_type == ExperimentType.MULTI_ARMED_BANDIT:
                variant.reward_sum += value
                self._update_bandit_confidence(variant_id)
            
            # Check guardrails
            self._check_experiment_guardrails()
            
            logger.debug(f"Conversion tracked: {user_id} -> {variant_id} ({conversion_type}: {value})")
            return True
            
        except Exception as e:
            logger.error(f"Conversion tracking failed for user {user_id}: {e}")
            return False

    def get_variant_config(self, user_context: UserContext) -> Optional[Dict[str, Any]]:
        """Get variant configuration for user"""
        variant_id = self.assign_variant(user_context)
        if variant_id and variant_id in self.variants:
            return self.variants[variant_id].config
        return None

    def calculate_metrics(self) -> Dict[str, ExperimentMetrics]:
        """Calculate comprehensive metrics for all variants"""
        results = {}
        
        try:
            for variant_id, variant in self.variants.items():
                metrics = ExperimentMetrics()
                
                # Basic metrics
                if variant.impressions > 0:
                    metrics.conversion_rate = variant.conversions / variant.impressions
                    metrics.revenue_per_user = variant.revenue / variant.impressions
                
                # Statistical significance
                if self._has_minimum_sample_size(variant_id):
                    metrics.statistical_significance = self._calculate_significance(variant_id)
                    metrics.confidence_interval = self._calculate_confidence_interval(variant_id)
                
                # Business metrics from conversion events
                variant_conversions = [e for e in self.conversion_events if e["variant_id"] == variant_id]
                
                if variant_conversions:
                    # Calculate advanced metrics
                    metrics.booking_completion_rate = len([e for e in variant_conversions if e["conversion_type"] == "booking_completed"]) / len(variant_conversions)
                    
                    booking_values = [e["value"] for e in variant_conversions if e["conversion_type"] == "booking_completed"]
                    if booking_values:
                        metrics.average_order_value = sum(booking_values) / len(booking_values)
                    
                    ancillary_values = [e["value"] for e in variant_conversions if e["conversion_type"] == "ancillary_purchase"]
                    metrics.ancillary_revenue = sum(ancillary_values)
                
                # Lift calculation (compared to control)
                if variant_id != "control" and "control" in self.variants:
                    control_rate = self.variants["control"].conversions / max(1, self.variants["control"].impressions)
                    if control_rate > 0:
                        metrics.lift = (metrics.conversion_rate - control_rate) / control_rate
                
                results[variant_id] = metrics
            
            # Store metrics history
            self.metrics_history.append(results.copy())
            
        except Exception as e:
            logger.error(f"Metrics calculation failed: {e}")
        
        return results

    def get_winning_variant(self) -> Optional[str]:
        """Determine winning variant based on statistical significance"""
        try:
            metrics = self.calculate_metrics()
            
            # Find variant with highest conversion rate and statistical significance
            best_variant = None
            best_rate = 0.0
            
            for variant_id, variant_metrics in metrics.items():
                if (variant_metrics.statistical_significance >= 0.95 and 
                    variant_metrics.conversion_rate > best_rate):
                    best_variant = variant_id
                    best_rate = variant_metrics.conversion_rate
            
            return best_variant
            
        except Exception as e:
            logger.error(f"Winner determination failed: {e}")
            return None

    def stop_experiment(self, reason: str = "Manual stop") -> bool:
        """Stop the experiment"""
        try:
            self.status = ExperimentStatus.COMPLETED
            self.end_time = datetime.now()
            
            # Calculate final metrics
            final_metrics = self.calculate_metrics()
            
            # Determine winner
            winner = self.get_winning_variant()
            
            logger.info(f"Experiment stopped: {self.experiment_id}, Winner: {winner}, Reason: {reason}")
            return True
            
        except Exception as e:
            logger.error(f"Failed to stop experiment {self.experiment_id}: {e}")
            return False

    # Private helper methods
    
    def _validate_configuration(self) -> bool:
        """Validate experiment configuration"""
        # Check variant weights sum to 1.0
        total_weight = sum(v.weight for v in self.variants.values())
        if abs(total_weight - 1.0) > 0.01:
            logger.error(f"Variant weights sum to {total_weight}, must equal 1.0")
            return False
        
        # Check minimum variants
        if len(self.variants) < 2:
            logger.error("Experiment must have at least 2 variants")
            return False
        
        return True

    def _is_user_eligible(self, user_context: UserContext) -> bool:
        """Check if user meets targeting criteria"""
        for criteria_type, criteria_value in self.targeting_criteria.items():
            if criteria_type == "loyalty_tier":
                if user_context.loyalty_tier not in criteria_value:
                    return False
            elif criteria_type == "location":
                if user_context.location not in criteria_value:
                    return False
            elif criteria_type == "new_users_only":
                if criteria_value and user_context.booking_frequency != "first_time":
                    return False
            # Add more targeting criteria as needed
        
        return True

    def _is_in_traffic_allocation(self, user_context: UserContext) -> bool:
        """Check if user is in traffic allocation percentage"""
        # Use consistent hashing for traffic allocation
        hash_input = f"{self.experiment_id}_{user_context.user_id}"
        hash_value = int(hashlib.md5(hash_input.encode()).hexdigest(), 16)
        allocation_bucket = (hash_value % 100) / 100.0
        
        return allocation_bucket < self.traffic_allocation

    def _get_variant_assignment(self, user_context: UserContext) -> Optional[str]:
        """Get variant assignment based on experiment type"""
        if self.experiment_type == ExperimentType.MULTI_ARMED_BANDIT:
            return self._get_bandit_assignment(user_context)
        elif self.experiment_type == ExperimentType.PERSONALIZED:
            return self._get_personalized_assignment(user_context)
        else:
            return self._get_random_assignment(user_context)

    def _get_random_assignment(self, user_context: UserContext) -> str:
        """Standard random assignment based on weights"""
        # Use consistent hashing for assignment
        hash_input = f"{self.experiment_id}_{user_context.user_id}_assignment"
        hash_value = int(hashlib.md5(hash_input.encode()).hexdigest(), 16)
        random_value = (hash_value % 10000) / 10000.0
        
        cumulative_weight = 0.0
        for variant_id, variant in self.variants.items():
            cumulative_weight += variant.weight
            if random_value <= cumulative_weight:
                return variant_id
        
        # Fallback to first variant
        return list(self.variants.keys())[0]

    def _get_bandit_assignment(self, user_context: UserContext) -> str:
        """Multi-armed bandit assignment using Upper Confidence Bound"""
        exploration_rate = self.bandit_config["exploration_rate"]
        
        # Calculate UCB scores for each variant
        total_pulls = sum(v.arm_pulls for v in self.variants.values())
        
        if total_pulls < len(self.variants) * 10:  # Exploration phase
            # Ensure each arm is pulled at least 10 times
            for variant_id, variant in self.variants.items():
                if variant.arm_pulls < 10:
                    return variant_id
        
        best_variant = None
        best_score = float('-inf')
        
        for variant_id, variant in self.variants.items():
            if variant.arm_pulls == 0:
                return variant_id  # Pull unexplored arms first
            
            # Calculate average reward
            avg_reward = variant.reward_sum / variant.arm_pulls
            
            # Calculate confidence bound
            import math
            confidence_bound = math.sqrt((2 * math.log(total_pulls)) / variant.arm_pulls)
            
            # UCB score
            ucb_score = avg_reward + exploration_rate * confidence_bound
            
            if ucb_score > best_score:
                best_score = ucb_score
                best_variant = variant_id
        
        return best_variant or list(self.variants.keys())[0]

    def _get_personalized_assignment(self, user_context: UserContext) -> str:
        """Personalized assignment based on user context"""
        # Simplified personalization logic
        # In production, this would use ML models
        
        # Check segment-specific strategies
        user_segment = self._determine_user_segment(user_context)
        if user_segment in self.segment_strategies:
            return self.segment_strategies[user_segment]
        
        # Fallback to bandit assignment
        return self._get_bandit_assignment(user_context)

    def _determine_user_segment(self, user_context: UserContext) -> str:
        """Determine user segment for personalization"""
        # High-value customers
        if user_context.loyalty_tier in ["platinum", "diamond"]:
            return "high_value"
        
        # Frequent travelers
        if user_context.booking_frequency == "frequent":
            return "frequent_traveler"
        
        # Business travelers
        if user_context.preferred_class in ["business", "first"]:
            return "business_traveler"
        
        # Leisure travelers
        return "leisure_traveler"

    def _update_bandit_confidence(self, variant_id: str):
        """Update confidence scores for multi-armed bandit"""
        variant = self.variants[variant_id]
        
        if variant.arm_pulls > 0:
            # Simple confidence calculation based on sample size
            import math
            variant.confidence = min(0.99, variant.arm_pulls / (variant.arm_pulls + 100))

    def _check_experiment_guardrails(self):
        """Check experiment guardrails and auto-stop if needed"""
        try:
            metrics = self.calculate_metrics()
            
            # Check for significant conversion drop
            if "control" in metrics:
                control_rate = metrics["control"].conversion_rate
                
                for variant_id, variant_metrics in metrics.items():
                    if variant_id != "control":
                        if control_rate > 0:
                            drop = (control_rate - variant_metrics.conversion_rate) / control_rate
                            if drop > self.guardrails["max_conversion_drop"]:
                                self.stop_experiment(f"Guardrail triggered: {drop:.2%} conversion drop in {variant_id}")
                                return
            
            # Check for early statistical significance
            if self.guardrails["auto_stop_on_significance"]:
                winner = self.get_winning_variant()
                if winner and self._has_minimum_sample_size(winner):
                    winner_metrics = metrics[winner]
                    if winner_metrics.statistical_significance >= 0.95:
                        self.stop_experiment(f"Early stop: Statistical significance achieved for {winner}")
                        return
        
        except Exception as e:
            logger.error(f"Guardrail check failed: {e}")

    def _has_minimum_sample_size(self, variant_id: str) -> bool:
        """Check if variant has minimum sample size for significance testing"""
        return self.variants[variant_id].impressions >= self.minimum_sample_size

    def _calculate_significance(self, variant_id: str) -> float:
        """Calculate statistical significance using Z-test"""
        # Simplified implementation - in production, use proper statistical libraries
        try:
            if "control" not in self.variants or variant_id == "control":
                return 0.0
            
            variant = self.variants[variant_id]
            control = self.variants["control"]
            
            if variant.impressions < self.minimum_sample_size or control.impressions < self.minimum_sample_size:
                return 0.0
            
            # Basic Z-test calculation
            p1 = variant.conversions / variant.impressions
            p2 = control.conversions / control.impressions
            
            if p1 == p2:
                return 0.0
            
            # Pooled probability
            p_pool = (variant.conversions + control.conversions) / (variant.impressions + control.impressions)
            
            # Standard error
            import math
            se = math.sqrt(p_pool * (1 - p_pool) * (1/variant.impressions + 1/control.impressions))
            
            if se == 0:
                return 0.0
            
            # Z-score
            z_score = abs(p1 - p2) / se
            
            # Convert to confidence level (simplified)
            if z_score >= 2.58:  # 99%
                return 0.99
            elif z_score >= 1.96:  # 95%
                return 0.95
            elif z_score >= 1.65:  # 90%
                return 0.90
            else:
                return z_score / 2.58 * 0.90  # Approximate
            
        except Exception as e:
            logger.error(f"Significance calculation failed: {e}")
            return 0.0

    def _calculate_confidence_interval(self, variant_id: str) -> tuple:
        """Calculate confidence interval for conversion rate"""
        try:
            variant = self.variants[variant_id]
            
            if variant.impressions == 0:
                return (0.0, 0.0)
            
            p = variant.conversions / variant.impressions
            n = variant.impressions
            
            # 95% confidence interval
            import math
            z = 1.96  # 95% confidence
            margin = z * math.sqrt((p * (1 - p)) / n)
            
            return (max(0.0, p - margin), min(1.0, p + margin))
            
        except Exception as e:
            logger.error(f"Confidence interval calculation failed: {e}")
            return (0.0, 0.0)


class PersonalizationExperimentManager:
    """Manager for multiple personalization experiments"""
    
    def __init__(self):
        self.experiments: Dict[str, AdvancedExperiment] = {}
        self.user_experiment_cache: Dict[str, Dict] = {}
        logger.info("Personalization Experiment Manager initialized")

    def create_experiment(self, config: Dict[str, Any]) -> AdvancedExperiment:
        """Create a new experiment from configuration"""
        try:
            # Parse variant configurations
            variants = []
            for variant_config in config["variants"]:
                variant = VariantConfiguration(
                    id=variant_config["id"],
                    name=variant_config["name"],
                    description=variant_config.get("description", ""),
                    weight=variant_config["weight"],
                    config=variant_config["config"]
                )
                variants.append(variant)
            
            # Create experiment
            experiment = AdvancedExperiment(
                experiment_id=config["experiment_id"],
                name=config["name"],
                description=config.get("description", ""),
                experiment_type=ExperimentType(config["type"]),
                variants=variants,
                targeting_criteria=config.get("targeting", {}),
                duration_days=config.get("duration_days", 14)
            )
            
            self.experiments[experiment.experiment_id] = experiment
            logger.info(f"Experiment created: {experiment.experiment_id}")
            
            return experiment
            
        except Exception as e:
            logger.error(f"Failed to create experiment: {e}")
            raise

    def get_user_experiments(self, user_context: UserContext) -> Dict[str, str]:
        """Get all active experiment assignments for a user"""
        assignments = {}
        
        try:
            for experiment_id, experiment in self.experiments.items():
                if experiment.status == ExperimentStatus.RUNNING:
                    variant_id = experiment.assign_variant(user_context)
                    if variant_id:
                        assignments[experiment_id] = variant_id
            
            # Cache assignments
            self.user_experiment_cache[user_context.user_id] = assignments
            
        except Exception as e:
            logger.error(f"Failed to get user experiments: {e}")
        
        return assignments

    def track_user_conversion(self, user_id: str, conversion_type: str, value: float = 1.0, metadata: Dict = None) -> bool:
        """Track conversion across all user's experiments"""
        success = False
        
        try:
            for experiment in self.experiments.values():
                if experiment.track_conversion(user_id, conversion_type, value, metadata):
                    success = True
            
        except Exception as e:
            logger.error(f"Failed to track conversion: {e}")
        
        return success

    def get_experiment_results(self, experiment_id: str) -> Optional[Dict]:
        """Get comprehensive experiment results"""
        try:
            if experiment_id not in self.experiments:
                return None
            
            experiment = self.experiments[experiment_id]
            metrics = experiment.calculate_metrics()
            
            results = {
                "experiment_id": experiment_id,
                "name": experiment.name,
                "status": experiment.status.value,
                "start_time": experiment.start_time.isoformat() if experiment.start_time else None,
                "end_time": experiment.end_time.isoformat() if experiment.end_time else None,
                "type": experiment.experiment_type.value,
                "participants": len(experiment.participant_assignments),
                "conversions": len(experiment.conversion_events),
                "winner": experiment.get_winning_variant(),
                "variants": {}
            }
            
            for variant_id, variant_metrics in metrics.items():
                variant = experiment.variants[variant_id]
                results["variants"][variant_id] = {
                    "name": variant.name,
                    "weight": variant.weight,
                    "impressions": variant.impressions,
                    "conversions": variant.conversions,
                    "revenue": variant.revenue,
                    "conversion_rate": variant_metrics.conversion_rate,
                    "revenue_per_user": variant_metrics.revenue_per_user,
                    "statistical_significance": variant_metrics.statistical_significance,
                    "confidence_interval": variant_metrics.confidence_interval,
                    "lift": variant_metrics.lift
                }
            
            return results
            
        except Exception as e:
            logger.error(f"Failed to get experiment results: {e}")
            return None


# Integration with existing framework
def trigger_advanced_experiment(experiment_name: str, user_context: UserContext, manager: PersonalizationExperimentManager) -> Optional[Dict[str, Any]]:
    """Enhanced experiment trigger with personalization"""
    try:
        # Get user's experiment assignments
        assignments = manager.get_user_experiments(user_context)
        
        # Return configuration for specific experiment
        if experiment_name in assignments:
            experiment = manager.experiments[experiment_name]
            variant_id = assignments[experiment_name]
            return experiment.variants[variant_id].config
        
        return None
        
    except Exception as e:
        logger.error(f"Advanced experiment trigger failed: {e}")
        return None


# Example usage and integration
if __name__ == "__main__":
    # Create experiment manager
    manager = PersonalizationExperimentManager()
    
    # Example experiment configuration
    experiment_config = {
        "experiment_id": "dynamic_pricing_personalization_v3",
        "name": "Dynamic Pricing with Personalization",
        "description": "Test personalized pricing based on user behavior and loyalty",
        "type": "personalized",
        "duration_days": 21,
        "targeting": {
            "loyalty_tier": ["silver", "gold", "platinum"],
            "location": ["US", "CA", "UK"]
        },
        "variants": [
            {
                "id": "control",
                "name": "Standard Pricing",
                "weight": 0.4,
                "config": {"pricing_strategy": "standard", "discount": 0.0}
            },
            {
                "id": "loyalty_discount",
                "name": "Loyalty-Based Discount",
                "weight": 0.3,
                "config": {"pricing_strategy": "loyalty_based", "discount": 0.15}
            },
            {
                "id": "behavioral_pricing",
                "name": "Behavioral Pricing",
                "weight": 0.3,
                "config": {"pricing_strategy": "behavioral", "dynamic_discount": True}
            }
        ]
    }
    
    # Create and start experiment
    experiment = manager.create_experiment(experiment_config)
    experiment.start_experiment()
    
    # Example user context
    user = UserContext(
        user_id="user_12345",
        session_id="session_67890",
        loyalty_tier="gold",
        location="US",
        booking_frequency="frequent",
        average_spend=1250.0,
        device_type="mobile"
    )
    
    # Get experiment configuration for user
    config = trigger_advanced_experiment("dynamic_pricing_personalization_v3", user, manager)
    print(f"User experiment config: {config}")
    
    # Track conversion
    manager.track_user_conversion("user_12345", "booking_completed", 850.0)
    
    # Get results
    results = manager.get_experiment_results("dynamic_pricing_personalization_v3")
    print(f"Experiment results: {json.dumps(results, indent=2)}") 