## Forecast Models for IAROS
*83 models continuously retrained to ensure accurate demand forecasting across airline operations*

### 1. Model Catalog & Methodologies
- **Types of Models:**  
  - **ARIMA:** For linear trends and seasonality.
  - **LSTM:** For complex, non-linear patterns.
  - **Hybrid/Ensemble:** Combining multiple methods for robust predictions.
- **Validation Metrics:**  
  - Target MAPE <10%
  - KS-statistic <0.3 for drift detection.

### 2. Retraining & Drift Detection
- **Automated Retraining:**  
  - Scheduled via AWS SageMaker pipelines (hourly/daily as required).
- **Drift Detection:**  
  - Implemented using KS-tests to compare prediction distributions against actual outcomes.
- **Fallback Strategy:**  
  - If retraining fails, revert to a 7-day moving average forecast and trigger an automated alert.

**Example Drift Detection (Python):**
```python
# File: services/forecasting_service/src/drift.py
from scipy.stats import ks_2samp

def check_drift(predictions, actuals):
    ks_stat, p_value = ks_2samp(predictions, actuals)
    if ks_stat > 0.3:
        trigger_retrain()  # Automatically initiate retraining via CI/CD pipeline
    return ks_stat, p_value
```
