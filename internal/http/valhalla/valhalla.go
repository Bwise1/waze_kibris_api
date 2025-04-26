package valhalla

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// ValhallaClient handles communication with the Valhalla API
type ValhallaClient struct {
	BaseURL string
	Client  *http.Client
}

// NewValhallaClient creates a new client instance
func NewValhallaClient(baseURL string) *ValhallaClient {
	return &ValhallaClient{
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: 30 * time.Second}, // Add a timeout
	}
}

// --- Request Structures ---

// Location represents a point in the route request
type Location struct {
	Lat           float64 `json:"lat"`
	Lon           float64 `json:"lon"`
	Type          *string `json:"type,omitempty"`           // Optional: break, through, via, break_through
	Heading       *int    `json:"heading,omitempty"`        // Optional: 0-360
	Street        *string `json:"street,omitempty"`         // Optional: For linking hints
	SideOfStreet  *string `json:"side_of_street,omitempty"` // Optional: left, right
	MinimumReach  *int    `json:"minimum_reachability,omitempty"`
	Radius        *int    `json:"radius,omitempty"`
	RankCandidate *bool   `json:"rank_candidates,omitempty"`
}

// CostingOptions allows specifying detailed options for a costing model (e.g., "auto")
type CostingOptions struct {
	Auto *AutoCostingOptions `json:"auto,omitempty"`
	// Add other costing models like pedestrian, bicycle, truck etc. as needed
	// Pedestrian *PedestrianCostingOptions `json:"pedestrian,omitempty"`
}

// AutoCostingOptions specific options for the "auto" costing model
type AutoCostingOptions struct {
	AvoidTolls    *bool    `json:"avoid_tolls,omitempty"`    // Avoid tolls where possible
	AvoidHighways *bool    `json:"avoid_highways,omitempty"` // Avoid highways where possible
	AvoidFerry    *bool    `json:"avoid_ferry,omitempty"`    // Avoid ferries where possible
	AvoidUnpaved  *bool    `json:"avoid_unpaved,omitempty"`  // Avoid unpaved roads where possible
	Height        *float64 `json:"height,omitempty"`         // Vehicle height restriction
	Width         *float64 `json:"width,omitempty"`          // Vehicle width restriction
	// Add more options as needed (e.g., top_speed, use_living_streets)
}

// RouteRequest represents the enhanced request payload for the /route endpoint
type RouteRequest struct {
	Locations      []Location      `json:"locations"`                 // Required: Start, End, and optional Via points
	Costing        string          `json:"costing"`                   // Required: e.g., "auto", "pedestrian", "bicycle"
	CostingOptions *CostingOptions `json:"costing_options,omitempty"` // Optional: Detailed costing parameters
	Alternates     *int            `json:"alternates,omitempty"`      // Optional: Number of alternative routes (e.g., 2)
	Units          *string         `json:"units,omitempty"`           // Optional: "kilometers" or "miles" (defaults to kilometers)
	Language       *string         `json:"language,omitempty"`        // Optional: Language for narrative instructions (e.g., "en-US")
	DateTime       *DateTime       `json:"date_time,omitempty"`       // Optional: Specify time for time-dependent routing
	ID             *string         `json:"id,omitempty"`              // Optional: User-defined ID for the request
	// Add other top-level parameters like directions_type, avoid_locations etc. if needed
}

// DateTime allows specifying departure/arrival time
type DateTime struct {
	Type  int    `json:"type"`  // 0 for departure, 1 for arrival
	Value string `json:"value"` // Format: YYYY-MM-DDTHH:MM (e.g., "2024-05-15T08:00")
}

// --- Raw Valhalla Response Structures ---
// (Using the detailed structure you provided, which is good)

// RouteResponse represents the raw, detailed response from the /route endpoint
type RouteResponse struct {
	Trip       Trip    `json:"trip"`
	Alternates []Trip  `json:"alternates,omitempty"` // Include alternates directly if API provides them at top level
	ID         *string `json:"id,omitempty"`         // Echoes request ID if provided
	// Note: Sometimes alternatives are nested within the 'trip' itself. Adjust if needed based on actual Valhalla output.
	// If alternatives are nested inside trip:
	// Trip struct { ... Alternates []Trip `json:"alternates,omitempty"` }
}

// Trip represents a single route (either primary or an alternative)
type Trip struct {
	Locations     []TripLocation `json:"locations"`
	Legs          []Leg          `json:"legs"`
	Summary       Summary        `json:"summary"`
	Status        int            `json:"status,omitempty"`
	StatusMessage string         `json:"status_message,omitempty"`
	Units         string         `json:"units,omitempty"`
	Language      string         `json:"language,omitempty"`
	// If alternatives are nested inside the main trip object:
	// Alternates []Trip `json:"alternates,omitempty"`
}

// TripLocation represents a location snapped to the graph in the response
type TripLocation struct {
	Lat          float64 `json:"lat"`
	Lon          float64 `json:"lon"`
	Name         string  `json:"name,omitempty"`
	Street       string  `json:"street,omitempty"`
	City         string  `json:"city,omitempty"`
	State        string  `json:"state,omitempty"`
	PostalCode   string  `json:"postal_code,omitempty"`
	Country      string  `json:"country,omitempty"`
	Type         string  `json:"type,omitempty"`
	SideOfStreet string  `json:"side_of_street,omitempty"`
}

// Summary provides overall details for a trip or leg
type Summary struct {
	Length             float64 `json:"length"` // In specified units (km or miles)
	Time               float64 `json:"time"`   // In seconds
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
}

// Leg represents a segment of the trip between two break locations
type Leg struct {
	Shape     string     `json:"shape"` // Encoded polyline
	Summary   Summary    `json:"summary"`
	Maneuvers []Maneuver `json:"maneuvers"`
}

// Maneuver represents a single turn-by-turn instruction
type Maneuver struct {
	Type                             int      `json:"type"`
	Instruction                      string   `json:"instruction"`
	VerbalTransitionAlertInstruction string   `json:"verbal_transition_alert_instruction,omitempty"`
	VerbalPreTransitionInstruction   string   `json:"verbal_pre_transition_instruction,omitempty"`
	VerbalPostTransitionInstruction  string   `json:"verbal_post_transition_instruction,omitempty"`
	StreetNames                      []string `json:"street_names,omitempty"`
	BeginStreetNames                 []string `json:"begin_street_names,omitempty"`
	Time                             float64  `json:"time"`   // Seconds for this maneuver
	Length                           float64  `json:"length"` // Distance for this maneuver (in specified units)
	BeginShapeIndex                  int      `json:"begin_shape_index"`
	EndShapeIndex                    int      `json:"end_shape_index"`
	TravelMode                       string   `json:"travel_mode"`           // e.g., "drive", "walk"
	TravelType                       string   `json:"travel_type,omitempty"` // e.g., "car", "foot"
	RoundaboutExitCount              int      `json:"roundabout_exit_count,omitempty"`
	DepartInstruction                string   `json:"depart_instruction,omitempty"`
	ArriveInstruction                string   `json:"arrive_instruction,omitempty"`
	// Add other fields like sign elements if needed
}

// --- Mobile-Friendly Formatted Structures ---

// --- Client Method ---

// GetRoute fetches a route from Valhalla using the enhanced request structure
func (vc *ValhallaClient) GetRoute(ctx context.Context, request RouteRequest) (*MobileRouteResponse, error) {
	url := fmt.Sprintf("%s/route", vc.BaseURL)

	// Marshal the request payload
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal route request: %w", err)
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Make the HTTP request
	resp, err := vc.Client.Do(req)
	if err != nil {
		log.Printf("Error making Valhalla request: %v\n", err)
		return nil, fmt.Errorf("failed to make route request to Valhalla: %w", err)
	}
	defer resp.Body.Close()

	// Read body first for better error reporting
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading Valhalla response body: %v\n", err)
		return nil, fmt.Errorf("failed to read Valhalla response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		log.Printf("Valhalla request failed with status %d: %s\n", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("valhalla error: status code %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the response
	var routeResponse RouteResponse
	err = json.Unmarshal(bodyBytes, &routeResponse)
	if err != nil {
		log.Printf("Error decoding Valhalla response: %v\nBody: %s\n", err, string(bodyBytes))
		return nil, fmt.Errorf("failed to decode Valhalla route response: %w", err)
	}

	// Basic validation of response
	if len(routeResponse.Trip.Legs) == 0 {
		log.Printf("Valhalla response contained no route legs. Status: %d, Message: %s\n", routeResponse.Trip.Status, routeResponse.Trip.StatusMessage)
		// Consider returning a more specific error or allowing empty result depending on use case
		// return nil, fmt.Errorf("no route found or error in Valhalla response (Status: %d, Msg: %s)", routeResponse.Trip.Status, routeResponse.Trip.StatusMessage)
	}
	mobileResponse, err := FormatRouteForMobile(&routeResponse)

	return mobileResponse, nil
}
