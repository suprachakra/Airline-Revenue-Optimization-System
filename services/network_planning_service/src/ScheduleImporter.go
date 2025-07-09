package network

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// FlightSchedule represents a flight schedule record.
type FlightSchedule struct {
	Route     string    `json:"route"`
	Departure time.Time `json:"departure"`
	Arrival   time.Time `json:"arrival"`
	Aircraft  string    `json:"aircraft"`
	Status    string    `json:"status"`
}

// ScheduleImporter imports flight schedules from an external API.
func ScheduleImporter(apiURL string) ([]FlightSchedule, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	var schedules []FlightSchedule

	// Attempt to fetch schedules with retry logic.
	for i := 0; i < 3; i++ {
		resp, err := client.Get(apiURL)
		if err != nil {
			log.Printf("Attempt %d: Error fetching schedules: %v", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}
		defer resp.Body.Close()

		if err := json.NewDecoder(resp.Body).Decode(&schedules); err != nil {
			log.Printf("Attempt %d: Error decoding schedules: %v", i+1, err)
			time.Sleep(2 * time.Second)
			continue
		}

		if len(schedules) > 0 {
			return schedules, nil
		}
		log.Printf("Attempt %d: No schedules returned, retrying...", i+1)
		time.Sleep(2 * time.Second)
	}

	return nil, http.ErrHandlerTimeout
}
