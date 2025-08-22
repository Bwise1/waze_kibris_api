package mapbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MapboxClient handles communication with Mapbox APIs
type MapboxClient struct {
	APIKey string // IMPORTANT: Handle your API Key securely! Load from environment variable.
	Client *http.Client
}

// NewMapboxClient creates a new Mapbox client instance
func NewMapboxClient(apiKey string) *MapboxClient {
	if apiKey == "" {
		log.Println("Warning: Mapbox API Key is empty.")
	}
	return &MapboxClient{
		APIKey: apiKey,
		Client: &http.Client{Timeout: 30 * time.Second},
	}
}

// --- Directions Structures for Mapbox ---

// DirectionsResponse represents the top-level response from Mapbox Directions API
type DirectionsResponse struct {
	Routes []Route `json:"routes"`
	Code   string  `json:"code"` // "Ok", "NoRoute", "NoSegment", "ProfileNotFound", etc.
}

// Route contains a single route with geometry and legs
type Route struct {
	Geometry     LineString `json:"geometry"`     // High-resolution road-snapped coordinates
	Legs         []Leg      `json:"legs"`
	WeightName   string     `json:"weight_name"`   // e.g., "routability"
	Weight       float64    `json:"weight"`
	Duration     float64    `json:"duration"`      // in seconds
	Distance     float64    `json:"distance"`      // in meters
}

// LineString contains the route geometry with road-snapped coordinates
type LineString struct {
	Type        string        `json:"type"`        // "LineString"
	Coordinates [][]float64   `json:"coordinates"` // [longitude, latitude] pairs - HIGH RESOLUTION!
}

// Leg represents a section of the route between waypoints
type Leg struct {
	Steps      []Step  `json:"steps"`
	Summary    string  `json:"summary"`
	Weight     float64 `json:"weight"`
	Duration   float64 `json:"duration"` // in seconds
	Distance   float64 `json:"distance"` // in meters
}

// Step contains detailed navigation instructions
type Step struct {
	Intersections []Intersection `json:"intersections"`
	Geometry      LineString     `json:"geometry"`
	Maneuver      Maneuver       `json:"maneuver"`
	Name          string         `json:"name"`
	Duration      float64        `json:"duration"` // in seconds
	Distance      float64        `json:"distance"` // in meters
	Mode          string         `json:"mode"`     // "driving", "walking", etc.
}

// Intersection contains information about road intersections
type Intersection struct {
	Location []float64 `json:"location"` // [longitude, latitude]
	Bearings []int     `json:"bearings"` // Available road directions
	Entry    []bool    `json:"entry"`    // Which roads can be entered
}

// Maneuver contains turn-by-turn navigation instructions
type Maneuver struct {
	Type          string    `json:"type"`           // "depart", "turn", "arrive", etc.
	Instruction   string    `json:"instruction"`    // Human-readable instruction
	BearingAfter  int       `json:"bearing_after"`  // Direction after maneuver
	BearingBefore int       `json:"bearing_before"` // Direction before maneuver
	Location      []float64 `json:"location"`       // [longitude, latitude]
	Modifier      string    `json:"modifier"`       // "left", "right", "straight", etc.
}

// Directions fetches directions between waypoints using Mapbox Directions API
// This provides HIGH-RESOLUTION, ROAD-SNAPPED coordinates for professional polylines
func (mc *MapboxClient) Directions(ctx context.Context, coordinates []string, profile string, alternatives bool, steps bool, geometries string) (*DirectionsResponse, error) {
	if mc.APIKey == "" {
		return nil, fmt.Errorf("mapbox API key is not set")
	}
	if len(coordinates) < 2 {
		return nil, fmt.Errorf("at least 2 coordinates (origin and destination) are required")
	}

	// Set defaults
	if profile == "" {
		profile = "driving" // "driving", "walking", "cycling", "driving-traffic"
	}
	if geometries == "" {
		geometries = "geojson" // Better for road-snapped coordinates
	}

	// Build coordinates string: "lon1,lat1;lon2,lat2;..."
	coordinatesStr := strings.Join(coordinates, ";")
	
	// Build Mapbox Directions API URL
	// Format: https://api.mapbox.com/directions/v5/mapbox/driving/coordinates?params
	baseURL := fmt.Sprintf("https://api.mapbox.com/directions/v5/mapbox/%s/%s", profile, coordinatesStr)
	
	params := url.Values{}
	params.Set("access_token", mc.APIKey)
	params.Set("geometries", geometries) // "geojson" gives high-resolution coordinates
	
	if alternatives {
		params.Set("alternatives", "true")
	}
	if steps {
		params.Set("steps", "true") // Include turn-by-turn instructions
	}
	
	// Add other useful parameters for better road snapping
	params.Set("overview", "full")      // Full geometry detail
	params.Set("continue_straight", "false") // Allow U-turns for better routes
	
	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Mapbox Directions request: %w", err)
	}

	resp, err := mc.Client.Do(req)
	if err != nil {
		log.Printf("Error making Mapbox Directions request: %v\n", err)
		return nil, fmt.Errorf("failed to execute Mapbox Directions request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading Mapbox Directions response body: %v\n", err)
		return nil, fmt.Errorf("failed to read Mapbox Directions response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Mapbox Directions request failed with status %d: %s\n", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("mapbox directions error: status code %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var dirResp DirectionsResponse
	err = json.Unmarshal(bodyBytes, &dirResp)
	if err != nil {
		log.Printf("Error decoding Mapbox Directions response: %v\nBody: %s\n", err, string(bodyBytes))
		return nil, fmt.Errorf("failed to decode Mapbox Directions response: %w", err)
	}

	// Check the code field in the response
	if dirResp.Code != "Ok" {
		log.Printf("Mapbox Directions API returned code: %s\n", dirResp.Code)
		return nil, fmt.Errorf("mapbox directions API error: %s", dirResp.Code)
	}

	return &dirResp, nil
}

// Helper function to convert lat,lng string to Mapbox format (lng,lat)
func FormatCoordinate(latLng string) string {
	// Convert "lat,lng" to "lng,lat" for Mapbox format
	parts := strings.Split(latLng, ",")
	if len(parts) != 2 {
		return latLng // Return as-is if invalid format
	}
	return fmt.Sprintf("%s,%s", parts[1], parts[0]) // lng,lat
}