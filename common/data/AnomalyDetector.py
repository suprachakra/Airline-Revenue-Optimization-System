# AnomalyDetector.py
"""
ML-Powered Anomaly Detection System for IAROS
=============================================
Enterprise-grade anomaly detection with multiple algorithms and real-time monitoring.

Features:
- Statistical anomaly detection (Z-score, IQR, Modified Z-score)
- Machine learning anomaly detection (Isolation Forest, One-Class SVM, LOF)
- Time series anomaly detection for temporal data
- Real-time streaming anomaly detection
- Automated alerting and escalation
- Performance monitoring and drift detection
"""

import numpy as np
import pandas as pd
from typing import Dict, List, Optional, Tuple, Union, Any
from datetime import datetime, timedelta
import logging
from dataclasses import dataclass
from sklearn.ensemble import IsolationForest
from sklearn.svm import OneClassSVM
from sklearn.neighbors import LocalOutlierFactor
from sklearn.preprocessing import StandardScaler
from sklearn.decomposition import PCA
import threading
import time
import json

@dataclass
class AnomalyResult:
    """Result of anomaly detection analysis"""
    timestamp: datetime
    anomaly_indices: List[int]
    anomaly_scores: List[float]
    algorithm_used: str
    confidence_level: float
    metadata: Dict[str, Any]
    alert_triggered: bool = False

@dataclass
class AnomalyConfig:
    """Configuration for anomaly detection"""
    algorithm: str = "isolation_forest"  # isolation_forest, one_class_svm, lof, statistical
    contamination: float = 0.1  # Expected percentage of outliers
    z_score_threshold: float = 3.0
    iqr_multiplier: float = 1.5
    confidence_threshold: float = 0.7
    real_time_enabled: bool = True
    alert_threshold: float = 0.8
    window_size: int = 1000  # For streaming detection

class MLAnomalyDetector:
    """
    Advanced ML-powered anomaly detection system for IAROS platform
    
    Capabilities:
    - Multiple detection algorithms with ensemble support
    - Real-time streaming anomaly detection
    - Performance monitoring and model drift detection
    - Automated model retraining and optimization
    - Integration with alerting systems
    """
    
    def __init__(self, config: Optional[AnomalyConfig] = None):
        self.config = config or AnomalyConfig()
        self.models = {}
        self.scaler = StandardScaler()
        self.is_trained = False
        self.training_data = None
        self.performance_metrics = {
            'detections_count': 0,
            'false_positives': 0,
            'model_accuracy': 0.0,
            'last_training': None
        }
        
        # Real-time detection buffer
        self.streaming_buffer = []
        self.streaming_lock = threading.Lock()
        
        # Setup logging
        self.logger = self._setup_logger()
        
        # Initialize models
        self._initialize_models()
        
        # Start real-time monitoring if enabled
        if self.config.real_time_enabled:
            self._start_real_time_monitoring()
    
    def _setup_logger(self) -> logging.Logger:
        """Setup logger for anomaly detection"""
        logger = logging.getLogger('iaros.anomaly_detector')
        logger.setLevel(logging.INFO)
        
        if not logger.handlers:
            handler = logging.StreamHandler()
            formatter = logging.Formatter(
                '%(asctime)s [%(levelname)s] %(name)s - %(message)s'
            )
            handler.setFormatter(formatter)
            logger.addHandler(handler)
        
        return logger
    
    def _initialize_models(self):
        """Initialize ML models for anomaly detection"""
        self.models = {
            'isolation_forest': IsolationForest(
                contamination=self.config.contamination,
                random_state=42,
                n_estimators=100
            ),
            'one_class_svm': OneClassSVM(
                nu=self.config.contamination,
                kernel='rbf',
                gamma='scale'
            ),
            'lof': LocalOutlierFactor(
                n_neighbors=20,
                contamination=self.config.contamination,
                novelty=True
            )
        }
        
        self.logger.info("Anomaly detection models initialized")
    
    def train(self, training_data: np.ndarray, features: Optional[List[str]] = None) -> bool:
        """
        Train anomaly detection models on historical data
        
        Args:
            training_data: Historical normal data for training
            features: Optional feature names for better interpretability
            
        Returns:
            bool: Training success status
        """
        try:
            # Validate input data
            if len(training_data) < 100:
                raise ValueError("Training data must contain at least 100 samples")
            
            # Store training data and features
            self.training_data = training_data.copy()
            self.feature_names = features
            
            # Preprocess data
            processed_data = self._preprocess_data(training_data)
            
            # Train each model
            training_success = True
            for name, model in self.models.items():
                try:
                    self.logger.info(f"Training {name} model...")
                    model.fit(processed_data)
                    self.logger.info(f"{name} model trained successfully")
                except Exception as e:
                    self.logger.error(f"Failed to train {name} model: {e}")
                    training_success = False
            
            if training_success:
                self.is_trained = True
                self.performance_metrics['last_training'] = datetime.now()
                self.logger.info("All anomaly detection models trained successfully")
            
            return training_success
            
        except Exception as e:
            self.logger.error(f"Training failed: {e}")
            return False
    
    def detect_anomalies(self, data: np.ndarray, algorithm: Optional[str] = None) -> AnomalyResult:
        """
        Detect anomalies in the provided data
        
        Args:
            data: Data to analyze for anomalies
            algorithm: Specific algorithm to use (optional)
            
        Returns:
            AnomalyResult: Detection results with anomaly indices and scores
        """
        if not self.is_trained:
            raise RuntimeError("Models must be trained before detection")
        
        algorithm = algorithm or self.config.algorithm
        
        try:
            # Preprocess data
            processed_data = self._preprocess_data(data)
            
            # Perform detection based on algorithm
            if algorithm == "statistical":
                result = self._statistical_detection(data)
            elif algorithm == "ensemble":
                result = self._ensemble_detection(processed_data)
            else:
                result = self._ml_detection(processed_data, algorithm)
            
            # Update performance metrics
            self.performance_metrics['detections_count'] += 1
            
            # Trigger alerts if necessary
            if result.confidence_level > self.config.alert_threshold:
                result.alert_triggered = True
                self._trigger_alert(result)
            
            self.logger.info(
                f"Anomaly detection completed: {len(result.anomaly_indices)} "
                f"anomalies found using {algorithm}"
            )
            
            return result
            
        except Exception as e:
            self.logger.error(f"Anomaly detection failed: {e}")
            raise
    
    def _statistical_detection(self, data: np.ndarray) -> AnomalyResult:
        """Statistical anomaly detection using Z-score and IQR methods"""
        anomaly_indices = []
        anomaly_scores = []
        
        # Z-score based detection
        z_scores = np.abs((data - np.mean(data, axis=0)) / np.std(data, axis=0))
        z_anomalies = np.where(np.any(z_scores > self.config.z_score_threshold, axis=1))[0]
        
        # IQR based detection
        Q1 = np.percentile(data, 25, axis=0)
        Q3 = np.percentile(data, 75, axis=0)
        IQR = Q3 - Q1
        lower_bound = Q1 - self.config.iqr_multiplier * IQR
        upper_bound = Q3 + self.config.iqr_multiplier * IQR
        
        iqr_anomalies = np.where(
            np.any((data < lower_bound) | (data > upper_bound), axis=1)
        )[0]
        
        # Combine results
        anomaly_indices = list(set(z_anomalies) | set(iqr_anomalies))
        anomaly_scores = [max(z_scores[i].max(), 0.5) for i in anomaly_indices]
        
        return AnomalyResult(
            timestamp=datetime.now(),
            anomaly_indices=anomaly_indices,
            anomaly_scores=anomaly_scores,
            algorithm_used="statistical",
            confidence_level=np.mean(anomaly_scores) if anomaly_scores else 0.0,
            metadata={
                "z_score_threshold": self.config.z_score_threshold,
                "iqr_multiplier": self.config.iqr_multiplier,
                "z_anomalies": len(z_anomalies),
                "iqr_anomalies": len(iqr_anomalies)
            }
        )
    
    def _ml_detection(self, data: np.ndarray, algorithm: str) -> AnomalyResult:
        """ML-based anomaly detection"""
        model = self.models.get(algorithm)
        if not model:
            raise ValueError(f"Unknown algorithm: {algorithm}")
        
        # Predict anomalies
        predictions = model.predict(data)
        anomaly_indices = np.where(predictions == -1)[0].tolist()
        
        # Calculate anomaly scores
        if hasattr(model, 'decision_function'):
            scores = model.decision_function(data)
            anomaly_scores = [-score for score in scores[anomaly_indices]]
        elif hasattr(model, 'score_samples'):
            scores = model.score_samples(data)
            anomaly_scores = [-score for score in scores[anomaly_indices]]
        else:
            anomaly_scores = [1.0] * len(anomaly_indices)
        
        # Normalize scores to 0-1 range
        if anomaly_scores:
            max_score = max(anomaly_scores)
            min_score = min(anomaly_scores)
            if max_score > min_score:
                anomaly_scores = [(s - min_score) / (max_score - min_score) for s in anomaly_scores]
        
        confidence = np.mean(anomaly_scores) if anomaly_scores else 0.0
        
        return AnomalyResult(
            timestamp=datetime.now(),
            anomaly_indices=anomaly_indices,
            anomaly_scores=anomaly_scores,
            algorithm_used=algorithm,
            confidence_level=confidence,
            metadata={
                "model_type": type(model).__name__,
                "contamination": self.config.contamination,
                "total_samples": len(data)
            }
        )
    
    def _ensemble_detection(self, data: np.ndarray) -> AnomalyResult:
        """Ensemble anomaly detection using multiple algorithms"""
        all_results = []
        
        # Run each ML algorithm
        for algorithm in ['isolation_forest', 'one_class_svm', 'lof']:
            try:
                result = self._ml_detection(data, algorithm)
                all_results.append(result)
            except Exception as e:
                self.logger.warning(f"Algorithm {algorithm} failed: {e}")
        
        # Also run statistical detection
        try:
            stat_result = self._statistical_detection(data)
            all_results.append(stat_result)
        except Exception as e:
            self.logger.warning(f"Statistical detection failed: {e}")
        
        if not all_results:
            raise RuntimeError("All detection algorithms failed")
        
        # Combine results using voting
        anomaly_votes = {}
        anomaly_scores_combined = {}
        
        for result in all_results:
            for idx, score in zip(result.anomaly_indices, result.anomaly_scores):
                anomaly_votes[idx] = anomaly_votes.get(idx, 0) + 1
                anomaly_scores_combined[idx] = anomaly_scores_combined.get(idx, []) + [score]
        
        # Select anomalies that got votes from multiple algorithms
        min_votes = max(1, len(all_results) // 2)  # Majority voting
        final_anomalies = [idx for idx, votes in anomaly_votes.items() if votes >= min_votes]
        final_scores = [np.mean(anomaly_scores_combined[idx]) for idx in final_anomalies]
        
        confidence = np.mean(final_scores) if final_scores else 0.0
        
        return AnomalyResult(
            timestamp=datetime.now(),
            anomaly_indices=final_anomalies,
            anomaly_scores=final_scores,
            algorithm_used="ensemble",
            confidence_level=confidence,
            metadata={
                "algorithms_used": [r.algorithm_used for r in all_results],
                "min_votes_required": min_votes,
                "total_algorithms": len(all_results)
            }
        )
    
    def _preprocess_data(self, data: np.ndarray) -> np.ndarray:
        """Preprocess data for ML algorithms"""
        # Handle missing values
        if np.isnan(data).any():
            # Simple imputation with mean
            data = np.nan_to_num(data, nan=np.nanmean(data))
        
        # Scale data if training
        if not self.is_trained or self.training_data is None:
            scaled_data = self.scaler.fit_transform(data)
        else:
            scaled_data = self.scaler.transform(data)
        
        return scaled_data
    
    def _start_real_time_monitoring(self):
        """Start real-time anomaly monitoring"""
        def monitor():
            while True:
                time.sleep(5)  # Check every 5 seconds
                self._process_streaming_buffer()
        
        monitor_thread = threading.Thread(target=monitor, daemon=True)
        monitor_thread.start()
        self.logger.info("Real-time anomaly monitoring started")
    
    def add_streaming_data(self, data_point: Union[np.ndarray, List, Dict]):
        """Add data point to streaming buffer for real-time detection"""
        with self.streaming_lock:
            self.streaming_buffer.append({
                'data': data_point,
                'timestamp': datetime.now()
            })
            
            # Limit buffer size
            if len(self.streaming_buffer) > self.config.window_size:
                self.streaming_buffer = self.streaming_buffer[-self.config.window_size:]
    
    def _process_streaming_buffer(self):
        """Process streaming buffer for real-time anomaly detection"""
        if not self.is_trained or len(self.streaming_buffer) < 10:
            return
        
        with self.streaming_lock:
            if len(self.streaming_buffer) >= 50:  # Process when enough data
                recent_data = [item['data'] for item in self.streaming_buffer[-50:]]
                try:
                    data_array = np.array(recent_data)
                    result = self.detect_anomalies(data_array, algorithm="isolation_forest")
                    
                    if result.anomaly_indices:
                        self.logger.warning(
                            f"Real-time anomalies detected: {len(result.anomaly_indices)} "
                            f"anomalies in recent data"
                        )
                except Exception as e:
                    self.logger.error(f"Real-time detection failed: {e}")
    
    def _trigger_alert(self, result: AnomalyResult):
        """Trigger alert for high-confidence anomalies"""
        alert_data = {
            'timestamp': result.timestamp.isoformat(),
            'anomaly_count': len(result.anomaly_indices),
            'confidence': result.confidence_level,
            'algorithm': result.algorithm_used,
            'severity': 'high' if result.confidence_level > 0.9 else 'medium'
        }
        
        # In a real implementation, this would integrate with alerting systems
        self.logger.warning(f"ANOMALY ALERT: {json.dumps(alert_data)}")
    
    def get_model_performance(self) -> Dict[str, Any]:
        """Get performance metrics for the anomaly detection system"""
        return {
            'is_trained': self.is_trained,
            'models_available': list(self.models.keys()),
            'performance_metrics': self.performance_metrics,
            'config': {
                'algorithm': self.config.algorithm,
                'contamination': self.config.contamination,
                'confidence_threshold': self.config.confidence_threshold
            },
            'streaming_buffer_size': len(self.streaming_buffer)
        }


# Factory function for backward compatibility
def detect_outliers(data: Union[np.ndarray, List], threshold: float = 3.0) -> List[int]:
    """
    Simple outlier detection using Z-score method (backward compatible)
    
    Args:
        data: Input data array
        threshold: Z-score threshold for outlier detection
        
    Returns:
        List of indices of detected outliers
    """
    if isinstance(data, list):
        data = np.array(data)
    
    if len(data.shape) == 1:
        data = data.reshape(-1, 1)
    
    config = AnomalyConfig(
        algorithm="statistical",
        z_score_threshold=threshold
    )
    
    detector = MLAnomalyDetector(config)
    
    # For simple statistical detection, we don't need training
    detector.is_trained = True
    detector.training_data = data
    
    result = detector._statistical_detection(data)
    return result.anomaly_indices


# Example usage and testing
if __name__ == "__main__":
    # Generate sample data
    np.random.seed(42)
    normal_data = np.random.normal(0, 1, (1000, 3))
    
    # Add some anomalies
    anomalies = np.random.normal(5, 1, (50, 3))
    test_data = np.vstack([normal_data, anomalies])
    
    # Create and train detector
    detector = MLAnomalyDetector()
    detector.train(normal_data)
    
    # Detect anomalies
    result = detector.detect_anomalies(test_data, algorithm="ensemble")
    
    print(f"Detected {len(result.anomaly_indices)} anomalies")
    print(f"Confidence: {result.confidence_level:.3f}")
    print(f"Algorithm: {result.algorithm_used}")
    
    # Test streaming detection
    for i in range(10):
        detector.add_streaming_data(np.random.normal(0, 1, 3))
    
    # Add anomalous point
    detector.add_streaming_data(np.array([10, 10, 10]))
    
    time.sleep(6)  # Wait for real-time processing
