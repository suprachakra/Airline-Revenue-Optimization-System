## Forecasting Service for IAROS
This module generates demand forecasts using 83 models—including ARIMA, LSTM, and hybrid ensembles—to drive dynamic pricing and network planning. It features automated retraining, drift detection using statistical tests (KS-test), and robust fallback strategies that revert to historical averages if live predictions fail.

### Key Features
- **Model Diversity:** ARIMA for linear trends; LSTM for non‑linear patterns.
- **Continuous Retraining:** Triggered via AWS SageMaker pipelines.
- **Drift Detection:** Uses KS-test to detect model drift.
- **Fallback Mechanisms:** Reverts to 7‑day moving average forecasts on failure.

For detailed design, refer to [Forecasting_Model_Design.md](../../technical_blueprint/Forecasting_Model_Design.md).
