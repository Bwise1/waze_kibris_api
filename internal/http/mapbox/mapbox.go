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
	Intersections       []Intersection       `json:"intersections"`
	Geometry            LineString           `json:"geometry"`
	Maneuver            Maneuver             `json:"maneuver"`
	Name                string               `json:"name"`
	Duration            float64              `json:"duration"` // in seconds
	Distance            float64              `json:"distance"` // in meters
	Mode                string               `json:"mode"`     // "driving", "walking", etc.
	VoiceInstructions   []VoiceInstruction   `json:"voiceInstructions,omitempty"`
	BannerInstructions  []BannerInstruction  `json:"bannerInstructions,omitempty"`
	Ref                 string               `json:"ref,omitempty"`          // Road reference/number
	Destinations        string               `json:"destinations,omitempty"` // Destination signage
	Exits               string               `json:"exits,omitempty"`        // Exit numbers
	Pronunciation       string               `json:"pronunciation,omitempty"`
	RotaryName          string               `json:"rotary_name,omitempty"`
	RotaryPronunciation string               `json:"rotary_pronunciation,omitempty"`
}

// Intersection contains information about road intersections
type Intersection struct {
	Location    []float64 `json:"location"`    // [longitude, latitude]
	Bearings    []int     `json:"bearings"`    // Available road directions
	Entry       []bool    `json:"entry"`       // Which roads can be entered
	In          int       `json:"in,omitempty"`       // Entry bearing index
	Out         int       `json:"out,omitempty"`      // Exit bearing index
	Lanes       []Lane    `json:"lanes,omitempty"`    // Lane guidance information
	Classes     []string  `json:"classes,omitempty"`  // Road classification
	MapboxStreetsV8 *MapboxStreetsV8 `json:"mapbox_streets_v8,omitempty"` // Additional road data
}

// Maneuver contains turn-by-turn navigation instructions
type Maneuver struct {
	Type           string    `json:"type"`           // "depart", "turn", "arrive", etc.
	Instruction    string    `json:"instruction"`    // Human-readable instruction
	BearingAfter   int       `json:"bearing_after"`  // Direction after maneuver
	BearingBefore  int       `json:"bearing_before"` // Direction before maneuver
	Location       []float64 `json:"location"`       // [longitude, latitude]
	Modifier       string    `json:"modifier"`       // "left", "right", "straight", etc.
	Exit           int       `json:"exit,omitempty"`           // Roundabout exit number
	RoundaboutExits int      `json:"roundabout_exits,omitempty"` // Total exits in roundabout
}

// VoiceInstruction contains voice guidance data
type VoiceInstruction struct {
	DistanceAlongGeometry float64 `json:"distanceAlongGeometry"` // Distance from start of step
	Announcement          string  `json:"announcement"`          // Text to be spoken
	SSMLAnnouncement      string  `json:"ssmlAnnouncement"`      // SSML formatted text
}

// BannerInstruction contains visual banner guidance
type BannerInstruction struct {
	DistanceAlongGeometry float64           `json:"distanceAlongGeometry"` // Distance from start of step
	Primary               BannerContent     `json:"primary"`               // Primary instruction text
	Secondary             *BannerContent    `json:"secondary,omitempty"`   // Secondary instruction text
	Sub                   *BannerContent    `json:"sub,omitempty"`         // Sub instruction text
	View                  *JunctionView     `json:"view,omitempty"`        // Junction view data
}

// BannerContent contains instruction text and components
type BannerContent struct {
	Text       string              `json:"text"`       // Display text
	Components []BannerComponent   `json:"components"` // Text components
	Type       string              `json:"type"`       // Instruction type
	Modifier   string              `json:"modifier"`   // Direction modifier
	Degrees    float64             `json:"degrees,omitempty"` // Turn angle
	DrivingSide string             `json:"driving_side,omitempty"` // left/right
}

// BannerComponent contains parts of instruction text
type BannerComponent struct {
	Text         string `json:"text"`
	Type         string `json:"type"`         // "text", "icon", "delimiter", "exit-number", etc.
	Abbreviation string `json:"abbr,omitempty"`
	AbbreviationPriority int `json:"abbr_priority,omitempty"`
}

// Lane contains lane guidance information
type Lane struct {
	Valid       bool     `json:"valid"`       // Whether this lane can be used
	Active      bool     `json:"active"`      // Whether this lane is recommended
	Indications []string `json:"indications"` // Lane markings: "left", "straight", "right", etc.
}

// JunctionView contains 3D intersection imagery data
type JunctionView struct {
	BaseURL   string `json:"base_url"`   // Base URL for junction images
	DataId    string `json:"data_id"`    // Junction data identifier
}

// MapboxStreetsV8 contains additional road metadata
type MapboxStreetsV8 struct {
	Class string `json:"class,omitempty"` // Road classification
}

// NavigationOptions contains parameters for enhanced navigation
type NavigationOptions struct {
	VoiceInstructions   bool   `json:"voice_instructions"`
	BannerInstructions  bool   `json:"banner_instructions"`
	VoiceUnits          string `json:"voice_units"`          // "metric" or "imperial"
	Language            string `json:"language"`             // "en", "es", etc.
	RoundaboutExits     bool   `json:"roundabout_exits"`
	WaypointNames       bool   `json:"waypoint_names"`
	Approaches          string `json:"approaches,omitempty"`  // "unrestricted", "curb", etc.
	Exclude             string `json:"exclude,omitempty"`     // "toll", "ferry", "motorway"
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
	
	// Enhanced navigation parameters
	params.Set("voice_instructions", "true")   // Include voice guidance
	params.Set("banner_instructions", "true")  // Include visual banners
	params.Set("voice_units", "metric")        // Distance units for voice
	params.Set("language", "en")               // Voice instruction language
	params.Set("roundabout_exits", "true")     // Include roundabout exit info
	// params.Set("waypoint_names", "true")    // Only enable when waypoint names are provided
	params.Set("annotations", "duration,distance,speed") // Additional route metadata
	
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

// DirectionsWithNavigation fetches directions with enhanced navigation features
func (mc *MapboxClient) DirectionsWithNavigation(ctx context.Context, coordinates []string, profile string, alternatives bool, options *NavigationOptions) (*DirectionsResponse, error) {
	if mc.APIKey == "" {
		return nil, fmt.Errorf("mapbox API key is not set")
	}
	if len(coordinates) < 2 {
		return nil, fmt.Errorf("at least 2 coordinates (origin and destination) are required")
	}

	// Set defaults
	if profile == "" {
		profile = "driving-traffic" // Use traffic-aware routing
	}
	if options == nil {
		options = &NavigationOptions{
			VoiceInstructions:  true,
			BannerInstructions: true,
			VoiceUnits:         "metric",
			Language:           "en",
			RoundaboutExits:    true,
			WaypointNames:      false, // Only enable when explicitly requested
		}
	}

	// Build coordinates string: "lon1,lat1;lon2,lat2;..."
	coordinatesStr := strings.Join(coordinates, ";")
	
	// Build Mapbox Directions API URL
	baseURL := fmt.Sprintf("https://api.mapbox.com/directions/v5/mapbox/%s/%s", profile, coordinatesStr)
	
	params := url.Values{}
	params.Set("access_token", mc.APIKey)
	params.Set("geometries", "geojson") // High-resolution coordinates
	params.Set("steps", "true")         // Always include steps for navigation
	params.Set("overview", "full")      // Full geometry detail
	params.Set("continue_straight", "false") // Allow U-turns
	
	if alternatives {
		params.Set("alternatives", "true")
	}

	// Enhanced navigation parameters
	if options.VoiceInstructions {
		params.Set("voice_instructions", "true")
	}
	if options.BannerInstructions {
		params.Set("banner_instructions", "true")
	}
	if options.VoiceUnits != "" {
		params.Set("voice_units", options.VoiceUnits)
	}
	if options.Language != "" {
		params.Set("language", options.Language)
	}
	if options.RoundaboutExits {
		params.Set("roundabout_exits", "true")
	}
	if options.WaypointNames {
		params.Set("waypoint_names", "true")
	}
	if options.Approaches != "" {
		params.Set("approaches", options.Approaches)
	}
	if options.Exclude != "" {
		params.Set("exclude", options.Exclude)
	}
	
	// Additional route metadata
	params.Set("annotations", "duration,distance,speed")
	
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