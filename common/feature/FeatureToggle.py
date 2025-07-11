# FeatureToggle.py
"""
Feature Toggle Manager
----------------------
Manages runtime feature flags for gradual rollouts and controlled experiments.
Allows dynamic enabling/disabling of features based on user segments.
"""

class FeatureManager:
    def __init__(self):
        self.flags = {
            "new_pricing_algorithm": {
                "enabled": False,
                "rollout_percent": 5  # 5% initial rollout
            },
            "enhanced_ui": {
                "enabled": False,
                "rollout_percent": 10
            }
        }
        
    def is_enabled(self, feature: str, user) -> bool:
        flag = self.flags.get(feature)
        if not flag:
            return False
        # Example: Rollout based on user ID modulo 100
        return (user.id % 100) < flag['rollout_percent']
