package valhalla

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type ValhallaClient struct {
	BaseURL string
	Client  *http.Client
}

func NewValhallaClient(baseURL string) *ValhallaClient {
	return &ValhallaClient{
		BaseURL: baseURL,
		Client:  &http.Client{},
	}
}

// RouteRequest represents the request payload for the /route endpoint
type RouteRequest struct {
	Locations []Location `json:"locations"`
	Costing   string     `json:"costing"`
}

// Location represents a point in the route
type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// RouteResponse represents the response from the /route endpoint
// type RouteResponse struct {
// 	Trip struct {
// 		Locations []Location `json:"locations"`
// 		Legs      []struct {
// 			Maneuvers []struct {
// 				Instruction string  `json:"instruction"`
// 				Length      float64 `json:"length"`
// 				Time        float64 `json:"time"`
// 			} `json:"maneuvers"`
// 		} `json:"legs"`
// 	} `json:"trip"`
// }

type RouteResponse struct {
	Trip struct {
		Locations []struct {
			Lat        float64 `json:"lat"`
			Lon        float64 `json:"lon"`
			Name       string  `json:"name,omitempty"`
			Street     string  `json:"street,omitempty"`
			City       string  `json:"city,omitempty"`
			State      string  `json:"state,omitempty"`
			PostalCode string  `json:"postal_code,omitempty"`
			Country    string  `json:"country,omitempty"`
			Type       string  `json:"type,omitempty"`
		} `json:"locations"`
		Summary struct {
			Length             float64 `json:"length"`
			Time               float64 `json:"time"`
			MinLat             float64 `json:"min_lat"`
			MinLon             float64 `json:"min_lon"`
			MaxLat             float64 `json:"max_lat"`
			MaxLon             float64 `json:"max_lon"`
			HasTollRoad        bool    `json:"has_toll_roads,omitempty"`
			HasHighway         bool    `json:"has_highways,omitempty"`
			HasFerry           bool    `json:"has_ferry,omitempty"`
			HasUnpaved         bool    `json:"has_unpaved,omitempty"`
			HasTunnel          bool    `json:"has_tunnel,omitempty"`
			HasSeasonalClosure bool    `json:"has_seasonal_closure,omitempty"`
			HasCountryCross    bool    `json:"has_country_cross,omitempty"`
		} `json:"summary"`
		Legs []struct {
			Shape   string `json:"shape"`
			Summary struct {
				Length float64 `json:"length"`
				Time   float64 `json:"time"`
			} `json:"summary"`
			Maneuvers []struct {
				Type                             int      `json:"type"`
				Instruction                      string   `json:"instruction"`
				VerbalTransitionAlertInstruction string   `json:"verbal_transition_alert_instruction,omitempty"`
				VerbalPreTransitionInstruction   string   `json:"verbal_pre_transition_instruction,omitempty"`
				VerbalPostTransitionInstruction  string   `json:"verbal_post_transition_instruction,omitempty"`
				StreetNames                      []string `json:"street_names,omitempty"`
				BeginStreetNames                 []string `json:"begin_street_names,omitempty"`
				Time                             float64  `json:"time"`
				Length                           float64  `json:"length"`
				BeginShapeIndex                  int      `json:"begin_shape_index"`
				EndShapeIndex                    int      `json:"end_shape_index"`
				TravelMode                       string   `json:"travel_mode"`
				TravelType                       string   `json:"travel_type,omitempty"`
				RoundaboutExitCount              int      `json:"roundabout_exit_count,omitempty"`
				DepartInstruction                string   `json:"depart_instruction,omitempty"`
				ArriveInstruction                string   `json:"arrive_instruction,omitempty"`
			} `json:"maneuvers"`
		} `json:"legs"`
		StatusMessage string `json:"status_message,omitempty"`
		Status        int    `json:"status,omitempty"`
	} `json:"trip"`
}

// GetRoute fetches a route from Valhalla
func (vc *ValhallaClient) GetRoute(request RouteRequest) (*RouteResponse, error) {
	url := fmt.Sprintf("%s/route", vc.BaseURL)

	// Marshal the request payload
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal route request: %w", err)
	}

	log.Println(bytes.NewBuffer(payload))
	// Make the HTTP request
	resp, err := vc.Client.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Println("Error making request:", err)
		return nil, fmt.Errorf("failed to make route request: %w", err)
	}
	defer resp.Body.Close()

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var routeResponse RouteResponse
	err = json.NewDecoder(resp.Body).Decode(&routeResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to decode route response: %w", err)
	}

	return &routeResponse, nil
}
