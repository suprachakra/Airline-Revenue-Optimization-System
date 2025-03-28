## Pricing Engine Design

The dynamic pricing engine is the core of our revenue optimization portfolio. It implements 142 distinct pricing scenarios that adjust fares in real time using a combination of machine learning outputs and rule‑based algorithms.

### Core Components

#### 1. Geo-Fencing Logic
- **Purpose:**  
  Determine regional discount rates based on customer geo-IP.
- **Implementation Details:**  
  ```go
  // GetGeoDiscount returns the discount percentage based on geo-IP.
  func GetGeoDiscount(geoIP string) float64 {
      switch geoIP {
      case "IN":
          return 0.15 // 15% discount for India-GCC routes
      case "AE", "QA":
          return 0.0  // No discount for domestic or Gulf-specific routes
      default:
          return 0.05 // Default 5% discount for other regions
      }
  }
```
