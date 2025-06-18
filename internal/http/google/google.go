package googlemaps

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

// GoogleMapsClient handles communication with Google Maps APIs
type GoogleMapsClient struct {
	APIKey string // IMPORTANT: Handle your API Key securely! Do not hardcode.
	Client *http.Client
}

// NewGoogleMapsClient creates a new client instance
// apiKey should be loaded securely (e.g., from environment variable)
func NewGoogleMapsClient(apiKey string) *GoogleMapsClient {
	if apiKey == "" {
		log.Println("Warning: Google Maps API Key is empty.")
	}
	return &GoogleMapsClient{
		APIKey: apiKey,
		Client: &http.Client{Timeout: 30 * time.Second},
	}
}

// --- Place Details Structures ---

// PlaceDetailsResponse represents the top-level response for a Place Details request
type PlaceDetailsResponse struct {
	HTMLAttributions []string           `json:"html_attributions"`
	Result           PlaceDetailsResult `json:"result"`
	Status           string             `json:"status"`                  // e.g., "OK", "ZERO_RESULTS", "INVALID_REQUEST", "OVER_QUERY_LIMIT", "REQUEST_DENIED", "UNKNOWN_ERROR"
	InfoMessages     []string           `json:"info_messages,omitempty"` // Additional info messages
}

// PlaceDetailsResult contains the detailed information about the place
type PlaceDetailsResult struct {
	AddressComponents  []AddressComponent `json:"address_components"`
	AdrAddress         string             `json:"adr_address"`     // Address in adr microformat
	BusinessStatus     string             `json:"business_status"` // e.g., "OPERATIONAL", "CLOSED_TEMPORARILY", "CLOSED_PERMANENTLY"
	FormattedAddress   string             `json:"formatted_address"`
	FormattedPhone     string             `json:"formatted_phone_number"`
	Geometry           Geometry           `json:"geometry"`
	Icon               string             `json:"icon"` // URL to icon
	IconMaskBaseURI    string             `json:"icon_mask_base_uri"`
	IconBgColor        string             `json:"icon_background_color"`
	InternationalPhone string             `json:"international_phone_number"`
	Name               string             `json:"name"`
	OpeningHours       *OpeningHours      `json:"opening_hours,omitempty"` // Pointer as it might be missing
	Photos             []Photo            `json:"photos,omitempty"`        // Array of photos
	PlaceID            string             `json:"place_id"`
	PlusCode           *PlusCode          `json:"plus_code,omitempty"`
	Rating             float64            `json:"rating"`            // Average rating
	Reference          string             `json:"reference"`         // Deprecated
	Reviews            []Review           `json:"reviews,omitempty"` // Array of reviews
	Types              []string           `json:"types"`             // e.g., ["restaurant", "food", "point_of_interest", "establishment"]
	URL                string             `json:"url"`               // Google Maps URL
	UserRatingsTotal   int                `json:"user_ratings_total"`
	UTCOffset          int                `json:"utc_offset_minutes"` // Offset from UTC in minutes
	Vicinity           string             `json:"vicinity"`           // Simplified address
	Website            string             `json:"website"`
	// Add other fields as needed based on the 'fields' parameter used
}

// AddressComponent represents a component of an address
type AddressComponent struct {
	LongName  string   `json:"long_name"`
	ShortName string   `json:"short_name"`
	Types     []string `json:"types"`
}

// Geometry contains location information
type Geometry struct {
	Location LatLng `json:"location"`
	Viewport Bounds `json:"viewport"`
}

// LatLng represents latitude and longitude
type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// Bounds represents a viewport bounding box
type Bounds struct {
	NorthEast LatLng `json:"northeast"`
	SouthWest LatLng `json:"southwest"`
}

// OpeningHours contains opening hours information
type OpeningHours struct {
	OpenNow     *bool           `json:"open_now,omitempty"` // Pointer as it might be missing
	Periods     []OpeningPeriod `json:"periods"`
	WeekdayText []string        `json:"weekday_text"`           // Formatted weekly hours
	SpecialDays []SpecialDay    `json:"special_days,omitempty"` // Upcoming special hours (e.g. holidays)
}

// OpeningPeriod represents a period when the place is open
type OpeningPeriod struct {
	Open  TimeOfWeek  `json:"open"`
	Close *TimeOfWeek `json:"close,omitempty"` // Close might be missing for always open
}

// TimeOfWeek represents a time point in a week
type TimeOfWeek struct {
	Day       int    `json:"day"`                 // 0=Sunday, 1=Monday, ..., 6=Saturday
	Time      string `json:"time"`                // HHMM format (e.g., "1700")
	Date      string `json:"date,omitempty"`      // YYYY-MM-DD format (used in special_days)
	Truncated bool   `json:"truncated,omitempty"` // If true, the closing time extends to the next day
}

// SpecialDay represents opening hours for a specific date (e.g., holiday)
type SpecialDay struct {
	Date        string `json:"date"`              // YYYY-MM-DD
	Exceptional bool   `json:"exceptional_hours"` // True if differs from regular hours
	// Include fields similar to OpeningPeriod if needed, check API docs
}

// Photo contains information about a place photo
type Photo struct {
	Height           int      `json:"height"`
	Width            int      `json:"width"`
	HTMLAttributions []string `json:"html_attributions"`
	PhotoReference   string   `json:"photo_reference"` // Use this reference to fetch the actual photo
}

// Review contains a user review
type Review struct {
	AuthorName       string `json:"author_name"`
	AuthorURL        string `json:"author_url"` // URL to author's Google profile
	Language         string `json:"language"`
	ProfilePhotoURL  string `json:"profile_photo_url"`
	Rating           int    `json:"rating"`                    // 1 to 5
	RelativeTimeDesc string `json:"relative_time_description"` // e.g., "a month ago"
	Text             string `json:"text"`
	Time             int64  `json:"time"` // Unix timestamp
	Translated       bool   `json:"translated"`
}

// PlusCode is an encoded location reference
type PlusCode struct {
	GlobalCode   string `json:"global_code"`
	CompoundCode string `json:"compound_code"`
}

// --- Client Methods ---

// GetPlaceDetails fetches detailed information about a place using its Place ID.
// placeID: The unique identifier for the place.
// fields: A list of fields to request (e.g., "name", "rating", "opening_hours", "photo", "review").
//
//	Requesting specific fields is REQUIRED and helps manage costs.
//	See https://developers.google.com/maps/documentation/places/web-service/details#fields
func (gc *GoogleMapsClient) GetPlaceDetails(ctx context.Context, placeID string, fields []string) (*PlaceDetailsResult, error) {
	if gc.APIKey == "" {
		return nil, fmt.Errorf("google maps API key is not set")
	}
	if placeID == "" {
		return nil, fmt.Errorf("placeID cannot be empty")
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("fields parameter cannot be empty for Place Details request")
	}

	baseURL := "https://maps.googleapis.com/maps/api/place/details/json"
	params := url.Values{}
	params.Set("place_id", placeID)
	params.Set("key", gc.APIKey)
	params.Set("fields", strings.Join(fields, ","))
	// Optional: Add language parameter: params.Set("language", "en")

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Place Details request: %w", err)
	}

	resp, err := gc.Client.Do(req)
	if err != nil {
		log.Printf("Error making Place Details request: %v\n", err)
		return nil, fmt.Errorf("failed to execute Place Details request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading Place Details response body: %v\n", err)
		return nil, fmt.Errorf("failed to read Place Details response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Place Details request failed with status %d: %s\n", resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("google maps error: status code %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var detailsResponse PlaceDetailsResponse
	err = json.Unmarshal(bodyBytes, &detailsResponse)
	if err != nil {
		log.Printf("Error decoding Place Details response: %v\nBody: %s\n", err, string(bodyBytes))
		return nil, fmt.Errorf("failed to decode Place Details response: %w", err)
	}

	// Check the status field in the response JSON
	if detailsResponse.Status != "OK" {
		log.Printf("Google Maps API returned status: %s\n", detailsResponse.Status)
		return nil, fmt.Errorf("google maps API error: %s", detailsResponse.Status)
	}

	return &detailsResponse.Result, nil
}

// ...existing code...

// --- Place Autocomplete Structures ---
type AutocompleteResponse struct {
	Predictions []AutocompletePrediction `json:"predictions"`
	Status      string                   `json:"status"`
}

type AutocompletePrediction struct {
	Description string   `json:"description"`
	PlaceID     string   `json:"place_id"`
	Types       []string `json:"types"`
}

// PlaceAutocomplete provides suggestions as the user types.
func (gc *GoogleMapsClient) PlaceAutocomplete(ctx context.Context, input string, location *LatLng, radius int) (*AutocompleteResponse, error) {
	if gc.APIKey == "" {
		return nil, fmt.Errorf("google maps API key is not set")
	}
	if input == "" {
		return nil, fmt.Errorf("input cannot be empty")
	}

	baseURL := "https://maps.googleapis.com/maps/api/place/autocomplete/json"
	params := url.Values{}
	params.Set("input", input)
	params.Set("key", gc.APIKey)
	if location != nil {
		params.Set("location", fmt.Sprintf("%f,%f", location.Lat, location.Lng))
	}
	if radius > 0 {
		params.Set("radius", fmt.Sprintf("%d", radius))
	}

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Place Autocomplete request: %w", err)
	}

	resp, err := gc.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Place Autocomplete request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Place Autocomplete response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google maps error: status code %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var autoResp AutocompleteResponse
	err = json.Unmarshal(bodyBytes, &autoResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Place Autocomplete response: %w", err)
	}
	if autoResp.Status != "OK" {
		return nil, fmt.Errorf("google maps API error: %s", autoResp.Status)
	}
	return &autoResp, nil
}

// --- Place Search Structures ---
type PlaceSearchResponse struct {
	Results []PlaceDetailsResult `json:"results"`
	Status  string               `json:"status"`
}

// PlaceSearch (Text Search) finds places by query.
func (gc *GoogleMapsClient) PlaceSearch(ctx context.Context, query string, location *LatLng, radius int) (*PlaceSearchResponse, error) {
	if gc.APIKey == "" {
		return nil, fmt.Errorf("google maps API key is not set")
	}
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	baseURL := "https://maps.googleapis.com/maps/api/place/textsearch/json"
	params := url.Values{}
	params.Set("query", query)
	params.Set("key", gc.APIKey)
	if location != nil {
		params.Set("location", fmt.Sprintf("%f,%f", location.Lat, location.Lng))
	}
	if radius > 0 {
		params.Set("radius", fmt.Sprintf("%d", radius))
	}

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Place Search request: %w", err)
	}

	resp, err := gc.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Place Search request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Place Search response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google maps error: status code %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var searchResp PlaceSearchResponse
	err = json.Unmarshal(bodyBytes, &searchResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Place Search response: %w", err)
	}
	if searchResp.Status != "OK" {
		return nil, fmt.Errorf("google maps API error: %s", searchResp.Status)
	}
	return &searchResp, nil
}

// --- Directions Structures ---
type DirectionsResponse struct {
	Routes []Route `json:"routes"`
	Status string  `json:"status"`
}

type Route struct {
	Summary          string   `json:"summary"`
	Legs             []Leg    `json:"legs"`
	OverviewPolyline Polyline `json:"overview_polyline"`
}

type Leg struct {
	Distance      TextValue `json:"distance"`
	Duration      TextValue `json:"duration"`
	StartAddress  string    `json:"start_address"`
	EndAddress    string    `json:"end_address"`
	StartLocation LatLng    `json:"start_location"`
	EndLocation   LatLng    `json:"end_location"`
	Steps         []Step    `json:"steps"`
}

type Step struct {
	Distance   TextValue `json:"distance"`
	Duration   TextValue `json:"duration"`
	HtmlInstr  string    `json:"html_instructions"`
	Polyline   Polyline  `json:"polyline"`
	StartLoc   LatLng    `json:"start_location"`
	EndLoc     LatLng    `json:"end_location"`
	TravelMode string    `json:"travel_mode"`
}

type Polyline struct {
	Points string `json:"points"`
}

type TextValue struct {
	Text  string `json:"text"`
	Value int    `json:"value"`
}

// Directions fetches directions between two points.
func (gc *GoogleMapsClient) Directions(ctx context.Context, origin, destination string, waypoints []string, mode string, alternatives bool) (*DirectionsResponse, error) {
	if gc.APIKey == "" {
		return nil, fmt.Errorf("google maps API key is not set")
	}
	if origin == "" || destination == "" {
		return nil, fmt.Errorf("origin and destination cannot be empty")
	}

	baseURL := "https://maps.googleapis.com/maps/api/directions/json"
	params := url.Values{}
	params.Set("origin", origin)
	params.Set("destination", destination)
	params.Set("key", gc.APIKey)
	if len(waypoints) > 0 {
		params.Set("waypoints", strings.Join(waypoints, "|"))
	}
	if mode != "" {
		params.Set("mode", mode)
	}
	if alternatives {
		params.Set("alternatives", "true")
	}

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Directions request: %w", err)
	}

	resp, err := gc.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Directions request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Directions response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google maps error: status code %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var dirResp DirectionsResponse
	err = json.Unmarshal(bodyBytes, &dirResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Directions response: %w", err)
	}
	if dirResp.Status != "OK" {
		return nil, fmt.Errorf("google maps API error: %s", dirResp.Status)
	}
	return &dirResp, nil
}

// --- Reverse Geocoding Structures ---
type GeocodeResponse struct {
	Results []GeocodeResult `json:"results"`
	Status  string          `json:"status"`
}

type GeocodeResult struct {
	FormattedAddress string   `json:"formatted_address"`
	Geometry         Geometry `json:"geometry"`
	PlaceID          string   `json:"place_id"`
	Types            []string `json:"types"`
}

// ReverseGeocode converts coordinates to an address.
func (gc *GoogleMapsClient) ReverseGeocode(ctx context.Context, lat, lng float64) (*GeocodeResponse, error) {
	if gc.APIKey == "" {
		return nil, fmt.Errorf("google maps API key is not set")
	}

	baseURL := "https://maps.googleapis.com/maps/api/geocode/json"
	params := url.Values{}
	params.Set("latlng", fmt.Sprintf("%f,%f", lat, lng))
	params.Set("key", gc.APIKey)

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Reverse Geocode request: %w", err)
	}

	resp, err := gc.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Reverse Geocode request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Reverse Geocode response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google maps error: status code %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var geoResp GeocodeResponse
	err = json.Unmarshal(bodyBytes, &geoResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Reverse Geocode response: %w", err)
	}
	if geoResp.Status != "OK" {
		return nil, fmt.Errorf("google maps API error: %s", geoResp.Status)
	}
	return &geoResp, nil
}

// ...existing code...
