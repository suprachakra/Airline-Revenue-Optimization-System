package model

// Ancillary defines an ancillary offering.
type Ancillary struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

// DefaultBundle returns a standard ancillary bundle.
func DefaultBundle() []Ancillary {
	return []Ancillary{
		{ID: 1, Name: "Standard Seat Selection", Price: 10.0, Description: "Basic seat selection service."},
		{ID: 2, Name: "Basic Meal Upgrade", Price: 15.0, Description: "Standard meal upgrade service."},
	}
}
