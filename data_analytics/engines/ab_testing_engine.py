#!/usr/bin/env python3
"""
IAROS A/B Testing Engine - Enterprise Multi-Armed Bandit Testing Platform
Implements advanced statistical validation and automated rollbacks
"""

import numpy as np
import pandas as pd
import asyncio
import redis
import json
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Tuple
from dataclasses import dataclass, asdict
from enum import Enum
from scipy import stats
import logging

class TestType(Enum):
    AB = "AB"
    MULTIVARIATE = "MULTIVARIATE"
    BANDIT = "BANDIT"
    SEQUENTIAL = "SEQUENTIAL"

class TestStatus(Enum):
    DRAFT = "DRAFT"
    RUNNING = "RUNNING"
    PAUSED = "PAUSED"
    COMPLETED = "COMPLETED"
    TERMINATED = "TERMINATED"

@dataclass
class TestConfiguration:
    test_id: str
    name: str
    test_type: TestType
    objective: str
    target_metric: str
    traffic_allocation: float
    min_sample_size: int
    max_duration_days: int
    statistical_power: float
    confidence_level: float
    variants: List[Dict[str, Any]]
    success_criteria: Dict[str, Any]
    rollback_criteria: Dict[str, Any]

@dataclass
class TestResult:
    test_id: str
    variant_results: Dict[str, Dict[str, Any]]
    winner: Optional[str]
    confidence: float
    statistical_significance: bool
    business_impact: Dict[str, float]
    recommendation: str
    timestamp: datetime

class ABTestingEngine:
    """
    Advanced A/B Testing Engine with Multi-Armed Bandits
    Supports revenue optimization testing with automated decision making
    """
    
    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.redis_client = redis.Redis(decode_responses=True)
        self.active_tests = {}
        self.bandit_algorithms = {
            'epsilon_greedy': EpsilonGreedyBandit(),
            'thompson_sampling': ThompsonSamplingBandit(),
            'ucb': UCBBandit()
        }
        self.logger = self._setup_logging()
        
    def _setup_logging(self):
        logging.basicConfig(level=logging.INFO)
        return logging.getLogger(__name__)

    async def create_test(self, config: TestConfiguration) -> str:
        """Create new A/B test with validation"""
        
        # Validate configuration
        await self._validate_test_config(config)
        
        # Calculate required sample size
        sample_size = self._calculate_sample_size(
            config.statistical_power,
            config.confidence_level,
            0.05  # Minimum detectable effect
        )
        
        test_data = {
            'config': asdict(config),
            'status': TestStatus.DRAFT.value,
            'created_at': datetime.now().isoformat(),
            'required_sample_size': sample_size,
            'variants': {v['id']: {'exposure': 0, 'conversions': 0, 'revenue': 0.0} 
                        for v in config.variants}
        }
        
        # Store in Redis
        await self._store_test(config.test_id, test_data)
        
        self.logger.info(f"Created test {config.test_id} with {len(config.variants)} variants")
        return config.test_id

    async def start_test(self, test_id: str) -> bool:
        """Start A/B test execution"""
        test_data = await self._get_test(test_id)
        
        if not test_data:
            raise ValueError(f"Test {test_id} not found")
        
        test_data['status'] = TestStatus.RUNNING.value
        test_data['started_at'] = datetime.now().isoformat()
        
        # Initialize bandit if applicable
        config = TestConfiguration(**test_data['config'])
        if config.test_type == TestType.BANDIT:
            self._initialize_bandit(test_id, config)
        
        await self._store_test(test_id, test_data)
        self.active_tests[test_id] = test_data
        
        self.logger.info(f"Started test {test_id}")
        return True

    async def assign_variant(self, test_id: str, user_id: str, context: Dict[str, Any] = None) -> str:
        """Assign user to test variant using appropriate algorithm"""
        test_data = await self._get_test(test_id)
        
        if not test_data or test_data['status'] != TestStatus.RUNNING.value:
            return 'control'  # Default fallback
        
        config = TestConfiguration(**test_data['config'])
        
        if config.test_type == TestType.BANDIT:
            variant = await self._bandit_assignment(test_id, user_id, context)
        else:
            variant = await self._static_assignment(test_id, user_id, config)
        
        # Track assignment
        await self._track_assignment(test_id, user_id, variant)
        
        return variant

    async def record_conversion(self, test_id: str, user_id: str, metric_value: float, 
                              revenue: float = 0.0) -> bool:
        """Record conversion event for statistical analysis"""
        test_data = await self._get_test(test_id)
        
        if not test_data:
            return False
        
        # Get user's assigned variant
        assignment = await self._get_user_assignment(test_id, user_id)
        if not assignment:
            return False
        
        variant_id = assignment['variant']
        
        # Update variant metrics
        test_data['variants'][variant_id]['conversions'] += 1
        test_data['variants'][variant_id]['revenue'] += revenue
        
        # Store conversion event
        conversion_data = {
            'test_id': test_id,
            'user_id': user_id,
            'variant': variant_id,
            'metric_value': metric_value,
            'revenue': revenue,
            'timestamp': datetime.now().isoformat()
        }
        
        await self._store_conversion(test_id, conversion_data)
        await self._store_test(test_id, test_data)
        
        # Check for early stopping conditions
        await self._check_early_stopping(test_id, test_data)
        
        return True

    async def analyze_test(self, test_id: str) -> TestResult:
        """Perform comprehensive statistical analysis"""
        test_data = await self._get_test(test_id)
        config = TestConfiguration(**test_data['config'])
        
        # Calculate metrics for each variant
        variant_results = {}
        
        for variant_id, data in test_data['variants'].items():
            exposures = data['exposure']
            conversions = data['conversions']
            revenue = data['revenue']
            
            conversion_rate = conversions / exposures if exposures > 0 else 0
            revenue_per_user = revenue / exposures if exposures > 0 else 0
            
            variant_results[variant_id] = {
                'exposures': exposures,
                'conversions': conversions,
                'conversion_rate': conversion_rate,
                'revenue': revenue,
                'revenue_per_user': revenue_per_user,
                'confidence_interval': self._calculate_confidence_interval(
                    conversions, exposures, config.confidence_level
                )
            }
        
        # Statistical significance testing
        significance_result = self._test_statistical_significance(variant_results, config)
        
        # Determine winner
        winner = self._determine_winner(variant_results, config.target_metric)
        
        # Calculate business impact
        business_impact = self._calculate_business_impact(variant_results)
        
        result = TestResult(
            test_id=test_id,
            variant_results=variant_results,
            winner=winner,
            confidence=significance_result['confidence'],
            statistical_significance=significance_result['significant'],
            business_impact=business_impact,
            recommendation=self._generate_recommendation(variant_results, winner),
            timestamp=datetime.now()
        )
        
        return result

    async def _bandit_assignment(self, test_id: str, user_id: str, context: Dict[str, Any]) -> str:
        """Multi-armed bandit variant assignment"""
        test_data = await self._get_test(test_id)
        config = TestConfiguration(**test_data['config'])
        
        # Get bandit algorithm
        algorithm_name = config.success_criteria.get('bandit_algorithm', 'thompson_sampling')
        bandit = self.bandit_algorithms[algorithm_name]
        
        # Get current performance data
        arms_performance = []
        for variant_id, data in test_data['variants'].items():
            exposures = data['exposure']
            conversions = data['conversions']
            arms_performance.append({
                'arm_id': variant_id,
                'trials': exposures,
                'successes': conversions
            })
        
        # Select arm using bandit algorithm
        selected_arm = bandit.select_arm(arms_performance)
        
        return selected_arm

    async def _static_assignment(self, test_id: str, user_id: str, config: TestConfiguration) -> str:
        """Static assignment based on user hash"""
        # Hash user ID to ensure consistent assignment
        user_hash = hash(f"{test_id}_{user_id}") % 100
        
        # Calculate cumulative traffic allocation
        cumulative = 0
        for variant in config.variants:
            cumulative += variant['traffic_split']
            if user_hash < cumulative:
                return variant['id']
        
        return config.variants[0]['id']  # Fallback

    def _calculate_sample_size(self, power: float, alpha: float, effect_size: float) -> int:
        """Calculate required sample size for statistical power"""
        from scipy.stats import norm
        
        z_alpha = norm.ppf(1 - alpha/2)
        z_beta = norm.ppf(power)
        
        # Simplified calculation for conversion rate
        sample_size = 2 * ((z_alpha + z_beta) / effect_size) ** 2
        
        return int(sample_size)

    def _test_statistical_significance(self, variant_results: Dict, config: TestConfiguration) -> Dict:
        """Test for statistical significance using appropriate test"""
        variants = list(variant_results.keys())
        
        if len(variants) != 2:
            return {'significant': False, 'confidence': 0.0, 'p_value': 1.0}
        
        control, treatment = variants[0], variants[1]
        
        # Chi-square test for proportions
        control_data = variant_results[control]
        treatment_data = variant_results[treatment]
        
        observed = np.array([
            [control_data['conversions'], control_data['exposures'] - control_data['conversions']],
            [treatment_data['conversions'], treatment_data['exposures'] - treatment_data['conversions']]
        ])
        
        chi2, p_value, dof, expected = stats.chi2_contingency(observed)
        
        alpha = 1 - config.confidence_level
        significant = p_value < alpha
        
        return {
            'significant': significant,
            'p_value': p_value,
            'confidence': 1 - p_value if significant else 0.0,
            'chi2_stat': chi2
        }

    def _calculate_confidence_interval(self, successes: int, trials: int, confidence_level: float) -> Tuple[float, float]:
        """Calculate confidence interval for conversion rate"""
        if trials == 0:
            return (0.0, 0.0)
        
        p = successes / trials
        z = stats.norm.ppf(1 - (1 - confidence_level) / 2)
        
        margin = z * np.sqrt(p * (1 - p) / trials)
        
        return (max(0, p - margin), min(1, p + margin))

    def _determine_winner(self, variant_results: Dict, target_metric: str) -> Optional[str]:
        """Determine winning variant based on target metric"""
        best_variant = None
        best_value = float('-inf')
        
        for variant_id, results in variant_results.items():
            value = results.get(target_metric, 0)
            if value > best_value:
                best_value = value
                best_variant = variant_id
        
        return best_variant

    def _calculate_business_impact(self, variant_results: Dict) -> Dict[str, float]:
        """Calculate projected business impact"""
        variants = list(variant_results.keys())
        
        if len(variants) < 2:
            return {}
        
        control = variant_results[variants[0]]
        treatment = variant_results[variants[1]]
        
        # Calculate relative improvements
        conversion_lift = ((treatment['conversion_rate'] - control['conversion_rate']) / 
                          control['conversion_rate'] * 100) if control['conversion_rate'] > 0 else 0
        
        revenue_lift = ((treatment['revenue_per_user'] - control['revenue_per_user']) / 
                       control['revenue_per_user'] * 100) if control['revenue_per_user'] > 0 else 0
        
        return {
            'conversion_rate_lift_pct': conversion_lift,
            'revenue_per_user_lift_pct': revenue_lift,
            'projected_annual_revenue_impact': treatment['revenue_per_user'] * 365 * 10000  # Simplified
        }

    def _generate_recommendation(self, variant_results: Dict, winner: Optional[str]) -> str:
        """Generate business recommendation based on results"""
        if not winner:
            return "Insufficient data for recommendation. Continue test or increase sample size."
        
        winner_data = variant_results[winner]
        
        if winner_data['conversion_rate'] > 0.05:  # Arbitrary threshold
            return f"Recommend implementing variant {winner}. Strong performance indicators."
        else:
            return f"Variant {winner} shows promise but requires validation with larger sample."

    async def _check_early_stopping(self, test_id: str, test_data: Dict):
        """Check if test should be stopped early"""
        config = TestConfiguration(**test_data['config'])
        
        # Check for sufficient sample size
        total_exposures = sum(v['exposure'] for v in test_data['variants'].values())
        
        if total_exposures >= test_data['required_sample_size']:
            result = await self.analyze_test(test_id)
            
            if result.statistical_significance:
                await self._stop_test(test_id, "Statistical significance achieved")
                return
        
        # Check rollback criteria
        await self._check_rollback_criteria(test_id, test_data)

    async def _check_rollback_criteria(self, test_id: str, test_data: Dict):
        """Check if test should be rolled back due to poor performance"""
        config = TestConfiguration(**test_data['config'])
        rollback_criteria = config.rollback_criteria
        
        for variant_id, data in test_data['variants'].items():
            if variant_id == 'control':
                continue
            
            # Check conversion rate degradation
            if data['exposure'] > 100:  # Minimum exposure threshold
                conversion_rate = data['conversions'] / data['exposure']
                control_rate = test_data['variants']['control']['conversions'] / test_data['variants']['control']['exposure']
                
                if conversion_rate < control_rate * rollback_criteria.get('min_conversion_rate_ratio', 0.8):
                    await self._rollback_test(test_id, f"Variant {variant_id} conversion rate below threshold")
                    return

    async def _stop_test(self, test_id: str, reason: str):
        """Stop test execution"""
        test_data = await self._get_test(test_id)
        test_data['status'] = TestStatus.COMPLETED.value
        test_data['stopped_at'] = datetime.now().isoformat()
        test_data['stop_reason'] = reason
        
        await self._store_test(test_id, test_data)
        self.logger.info(f"Stopped test {test_id}: {reason}")

    async def _rollback_test(self, test_id: str, reason: str):
        """Rollback test due to poor performance"""
        test_data = await self._get_test(test_id)
        test_data['status'] = TestStatus.TERMINATED.value
        test_data['terminated_at'] = datetime.now().isoformat()
        test_data['termination_reason'] = reason
        
        await self._store_test(test_id, test_data)
        self.logger.warning(f"Rolled back test {test_id}: {reason}")

    # Redis operations
    async def _store_test(self, test_id: str, data: Dict):
        """Store test data in Redis"""
        await asyncio.get_event_loop().run_in_executor(
            None, self.redis_client.setex, f"test:{test_id}", 86400, json.dumps(data)
        )

    async def _get_test(self, test_id: str) -> Optional[Dict]:
        """Retrieve test data from Redis"""
        data = await asyncio.get_event_loop().run_in_executor(
            None, self.redis_client.get, f"test:{test_id}"
        )
        return json.loads(data) if data else None

    async def _track_assignment(self, test_id: str, user_id: str, variant: str):
        """Track user assignment"""
        assignment_data = {
            'test_id': test_id,
            'user_id': user_id,
            'variant': variant,
            'timestamp': datetime.now().isoformat()
        }
        
        await asyncio.get_event_loop().run_in_executor(
            None, self.redis_client.setex, 
            f"assignment:{test_id}:{user_id}", 86400, json.dumps(assignment_data)
        )

    async def _get_user_assignment(self, test_id: str, user_id: str) -> Optional[Dict]:
        """Get user's variant assignment"""
        data = await asyncio.get_event_loop().run_in_executor(
            None, self.redis_client.get, f"assignment:{test_id}:{user_id}"
        )
        return json.loads(data) if data else None

    async def _store_conversion(self, test_id: str, conversion_data: Dict):
        """Store conversion event"""
        key = f"conversion:{test_id}:{datetime.now().timestamp()}"
        await asyncio.get_event_loop().run_in_executor(
            None, self.redis_client.setex, key, 86400, json.dumps(conversion_data)
        )

    async def _validate_test_config(self, config: TestConfiguration):
        """Validate test configuration"""
        if not config.variants or len(config.variants) < 2:
            raise ValueError("At least 2 variants required")
        
        total_traffic = sum(v['traffic_split'] for v in config.variants)
        if abs(total_traffic - 100) > 0.1:
            raise ValueError("Traffic splits must sum to 100%")

# Bandit algorithm implementations
class EpsilonGreedyBandit:
    """Epsilon-greedy multi-armed bandit algorithm"""
    
    def __init__(self, epsilon: float = 0.1):
        self.epsilon = epsilon
    
    def select_arm(self, arms_performance: List[Dict]) -> str:
        """Select arm using epsilon-greedy strategy"""
        if np.random.random() < self.epsilon:
            # Explore: random selection
            return np.random.choice([arm['arm_id'] for arm in arms_performance])
        else:
            # Exploit: select best performing arm
            best_arm = max(arms_performance, 
                          key=lambda x: x['successes'] / max(x['trials'], 1))
            return best_arm['arm_id']

class ThompsonSamplingBandit:
    """Thompson sampling multi-armed bandit algorithm"""
    
    def select_arm(self, arms_performance: List[Dict]) -> str:
        """Select arm using Thompson sampling"""
        sampled_values = []
        
        for arm in arms_performance:
            alpha = arm['successes'] + 1
            beta = arm['trials'] - arm['successes'] + 1
            sampled_value = np.random.beta(alpha, beta)
            sampled_values.append((arm['arm_id'], sampled_value))
        
        # Select arm with highest sampled value
        best_arm = max(sampled_values, key=lambda x: x[1])
        return best_arm[0]

class UCBBandit:
    """Upper Confidence Bound multi-armed bandit algorithm"""
    
    def select_arm(self, arms_performance: List[Dict]) -> str:
        """Select arm using UCB strategy"""
        total_trials = sum(arm['trials'] for arm in arms_performance)
        
        if total_trials == 0:
            return arms_performance[0]['arm_id']
        
        ucb_values = []
        
        for arm in arms_performance:
            if arm['trials'] == 0:
                ucb_value = float('inf')
            else:
                mean_reward = arm['successes'] / arm['trials']
                confidence_interval = np.sqrt(2 * np.log(total_trials) / arm['trials'])
                ucb_value = mean_reward + confidence_interval
            
            ucb_values.append((arm['arm_id'], ucb_value))
        
        # Select arm with highest UCB value
        best_arm = max(ucb_values, key=lambda x: x[1])
        return best_arm[0]

# Example usage
async def main():
    engine = ABTestingEngine({})
    
    # Create test configuration
    config = TestConfiguration(
        test_id="pricing_test_001",
        name="Dynamic Pricing Algorithm Test",
        test_type=TestType.BANDIT,
        objective="Increase revenue per passenger",
        target_metric="revenue_per_user",
        traffic_allocation=0.5,
        min_sample_size=1000,
        max_duration_days=30,
        statistical_power=0.8,
        confidence_level=0.95,
        variants=[
            {"id": "control", "traffic_split": 50, "description": "Current pricing"},
            {"id": "dynamic", "traffic_split": 50, "description": "AI dynamic pricing"}
        ],
        success_criteria={"bandit_algorithm": "thompson_sampling"},
        rollback_criteria={"min_conversion_rate_ratio": 0.8}
    )
    
    # Create and start test
    test_id = await engine.create_test(config)
    await engine.start_test(test_id)
    
    print(f"Started A/B test: {test_id}")

if __name__ == "__main__":
    asyncio.run(main()) 