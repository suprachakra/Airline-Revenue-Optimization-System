## Pricing Service for IAROS
This module implements a dynamic pricing engine for 142 scenarios. It integrates with the forecasting service to adjust fares in real time based on geo‑fencing, corporate contracts, and event‑driven adjustments. Robust fallback mechanisms (via the FallbackEngine) ensure uninterrupted service even if live data is delayed.

### Key Features
- **Dynamic Pricing Algorithms:** Implements multiple pricing scenarios.
- **Integration with Forecasting:** Adjusts fares based on predictive data.
- **Fallback Strategy:** Multi‑layer fallback using live data, Redis cache, historical averages, and static floor pricing.
- **Compliance:** Enforces rule‑based pricing constraints per ATPCO standards.

For detailed design, refer to [Pricing_Engine_Design.md](../../technical_blueprint/Pricing_Engine_Design.md).
