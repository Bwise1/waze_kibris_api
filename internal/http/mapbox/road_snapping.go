package mapbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Note: MapMatchingResponse, Matching, and Tracepoint types are defined in mapbox.go

// LocationSnapRequest represents a request to snap locations to roads
type LocationSnapRequest struct {
	Locations    []LocationPoint `json:"locations"`              // GPS coordinates to snap
	Profile      string          `json:"profile,omitempty"`      // driving, walking, cycling
	RouteGeometry *LineString    `json:"route_geometry,omitempty"` // Active route for snap-to-route
	SnapRadius   int             `json:"snap_radius,omitempty"`  // Max snap distance in meters
	OppositeSide bool            `json:"opposite_side,omitempty"` // For reports on opposite side
}

// LocationPoint represents a GPS coordinate with optional metadata
type LocationPoint struct {
	Latitude  float64    `json:"latitude"`
	Longitude float64    `json:"longitude"`
	Timestamp *time.Time `json:"timestamp,omitempty"` // For better matching accuracy
	Accuracy  float64    `json:"accuracy,omitempty"`  // GPS accuracy in meters
	Heading   float64    `json:"heading,omitempty"`   // Direction of travel
}

// LocationSnapResponse represents the response with snapped coordinates
type LocationSnapResponse struct {
	SnappedLocations []SnappedLocation `json:"snapped_locations"`
	SnapType        string            `json:"snap_type"`        // "route", "road", "none"
	Confidence      float64           `json:"confidence"`       // 0.0 to 1.0
	Message         string            `json:"message,omitempty"`
}

// SnappedLocation represents a location snapped to road/route
type SnappedLocation struct {
	Original    LocationPoint `json:"original"`
	Snapped     LocationPoint `json:"snapped"`
	SnapDistance float64       `json:"snap_distance"` // Distance moved in meters
	OnRoute     bool          `json:"on_route"`       // If snapped to active route
}

// SnapLocationToRoad intelligently snaps location based on context
// Prioritizes route snapping during navigation, falls back to road snapping
func (mc *MapboxClient) SnapLocationToRoad(ctx context.Context, req LocationSnapRequest) (*LocationSnapResponse, error) {
	if mc.APIKey == "" {
		return nil, fmt.Errorf("mapbox API key is not set")
	}
	if len(req.Locations) == 0 {
		return nil, fmt.Errorf("no locations provided")
	}

	// Set defaults
	if req.Profile == "" {
		req.Profile = "driving"
	}
	if req.SnapRadius == 0 {
		req.SnapRadius = 25 // 25 meters default snap radius
	}

	var response *LocationSnapResponse
	var err error

	// Strategy 1: Try route snapping first if route geometry provided
	if req.RouteGeometry != nil && len(req.RouteGeometry.Coordinates) > 0 {
		log.Println("🛣️ Attempting route snapping...")
		response, err = mc.snapToRoute(ctx, req)
		if err == nil && response.Confidence > 0.6 {
			response.SnapType = "route"
			log.Printf("✅ Route snapping successful with confidence: %.2f", response.Confidence)
			return response, nil
		}
		log.Printf("⚠️ Route snapping failed or low confidence: %.2f", response.Confidence)
	}

	// Strategy 2: Fall back to road snapping using Map Matching API
	log.Println("🛣️ Attempting road snapping...")
	response, err = mc.snapToRoadNetwork(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("road snapping failed: %w", err)
	}

	response.SnapType = "road"
	log.Printf("✅ Road snapping successful with confidence: %.2f", response.Confidence)

	// Strategy 3: Handle opposite side placement for reports
	if req.OppositeSide {
		response = mc.adjustForOppositeSide(response)
		log.Println("🔄 Applied opposite side adjustment for report placement")
	}

	return response, nil
}

// snapToRoute snaps location to the provided route geometry
func (mc *MapboxClient) snapToRoute(ctx context.Context, req LocationSnapRequest) (*LocationSnapResponse, error) {
	response := &LocationSnapResponse{
		SnappedLocations: make([]SnappedLocation, 0, len(req.Locations)),
		SnapType:        "route",
	}

	totalConfidence := 0.0
	
	for _, location := range req.Locations {
		snapped, distance := mc.findNearestPointOnRoute(location, req.RouteGeometry)
		
		confidence := mc.calculateRouteSnapConfidence(distance, req.SnapRadius)
		totalConfidence += confidence

		response.SnappedLocations = append(response.SnappedLocations, SnappedLocation{
			Original:     location,
			Snapped:      snapped,
			SnapDistance: distance,
			OnRoute:      distance <= float64(req.SnapRadius),
		})
	}

	response.Confidence = totalConfidence / float64(len(req.Locations))
	
	if response.Confidence < 0.3 {
		response.Message = "Low confidence route snapping - consider road snapping"
	}

	return response, nil
}

// snapToRoadNetwork uses Mapbox Map Matching API for road snapping
func (mc *MapboxClient) snapToRoadNetwork(ctx context.Context, req LocationSnapRequest) (*LocationSnapResponse, error) {
	// Prepare coordinates for Map Matching API
	coordinates := make([]string, len(req.Locations))
	for i, loc := range req.Locations {
		coordinates[i] = fmt.Sprintf("%.6f,%.6f", loc.Longitude, loc.Latitude)
	}

	coordinatesStr := strings.Join(coordinates, ";")
	baseURL := fmt.Sprintf("https://api.mapbox.com/matching/v5/mapbox/%s/%s", req.Profile, coordinatesStr)

	params := url.Values{}
	params.Set("access_token", mc.APIKey)
	params.Set("geometries", "geojson")
	params.Set("radiuses", strings.Repeat(fmt.Sprintf("%d;", req.SnapRadius), len(coordinates)-1)+fmt.Sprintf("%d", req.SnapRadius))
	params.Set("steps", "false")
	params.Set("overview", "full")

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := mc.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mapbox API error: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var mapMatchingResp MapMatchingResponse
	if err := json.Unmarshal(bodyBytes, &mapMatchingResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if mapMatchingResp.Code != "Ok" {
		return nil, fmt.Errorf("mapbox API returned error: %s", mapMatchingResp.Code)
	}

	return mc.processMapMatchingResponse(req.Locations, &mapMatchingResp), nil
}

// processMapMatchingResponse converts Map Matching response to our format
func (mc *MapboxClient) processMapMatchingResponse(originalLocations []LocationPoint, resp *MapMatchingResponse) *LocationSnapResponse {
	response := &LocationSnapResponse{
		SnappedLocations: make([]SnappedLocation, 0, len(originalLocations)),
		SnapType:        "road",
	}

	// Calculate overall confidence from matchings
	totalConfidence := 0.0
	validMatchings := 0

	for _, matching := range resp.Matchings {
		if matching.Confidence > 0 {
			totalConfidence += matching.Confidence
			validMatchings++
		}
	}

	if validMatchings > 0 {
		response.Confidence = totalConfidence / float64(validMatchings)
	}

	// Process tracepoints to create snapped locations
	for i, original := range originalLocations {
		snappedLocation := SnappedLocation{
			Original: original,
			Snapped:  original, // Default to original if no match
			OnRoute:  false,
		}

		// Find corresponding tracepoint
		if i < len(resp.Tracepoints) && resp.Tracepoints[i].Location != nil {
			tracepoint := resp.Tracepoints[i]
			snapped := LocationPoint{
				Longitude: tracepoint.Location[0],
				Latitude:  tracepoint.Location[1],
				Timestamp: original.Timestamp,
				Accuracy:  original.Accuracy,
				Heading:   original.Heading,
			}

			distance := mc.calculateDistance(original.Latitude, original.Longitude, snapped.Latitude, snapped.Longitude)
			
			snappedLocation.Snapped = snapped
			snappedLocation.SnapDistance = distance
			snappedLocation.OnRoute = true
		}

		response.SnappedLocations = append(response.SnappedLocations, snappedLocation)
	}

	return response
}

// adjustForOppositeSide adjusts snapped locations for opposite side placement (for reports)
func (mc *MapboxClient) adjustForOppositeSide(response *LocationSnapResponse) *LocationSnapResponse {
	for i := range response.SnappedLocations {
		snapped := &response.SnappedLocations[i]
		
		// Calculate perpendicular offset to place on opposite side
		// This is a simplified approach - in production, you'd use road geometry
		offsetDistance := 15.0 // 15 meters offset for opposite side
		
		// Use heading to determine offset direction
		if snapped.Original.Heading != 0 {
			// Calculate perpendicular heading (90 degrees offset)
			perpHeading := snapped.Original.Heading + 90
			if perpHeading >= 360 {
				perpHeading -= 360
			}
			
			// Convert to radians for calculation
			headingRad := perpHeading * math.Pi / 180
			
			// Calculate offset coordinates
			latOffset := offsetDistance * math.Cos(headingRad) / 111111 // Rough lat degrees per meter
			lngOffset := offsetDistance * math.Sin(headingRad) / (111111 * math.Cos(snapped.Snapped.Latitude * math.Pi / 180))
			
			// Apply offset
			snapped.Snapped.Latitude += latOffset
			snapped.Snapped.Longitude += lngOffset
			snapped.SnapDistance += offsetDistance
		}
	}
	
	response.Message = "Adjusted for opposite side placement"
	return response
}

// Helper functions

func (mc *MapboxClient) findNearestPointOnRoute(location LocationPoint, route *LineString) (LocationPoint, float64) {
	if len(route.Coordinates) == 0 {
		return location, math.Inf(1)
	}

	minDistance := math.Inf(1)
	nearestPoint := location

	// Find nearest point on route
	for _, coord := range route.Coordinates {
		if len(coord) >= 2 {
			routeLat := coord[1]
			routeLng := coord[0]
			distance := mc.calculateDistance(location.Latitude, location.Longitude, routeLat, routeLng)
			
			if distance < minDistance {
				minDistance = distance
				nearestPoint = LocationPoint{
					Latitude:  routeLat,
					Longitude: routeLng,
					Timestamp: location.Timestamp,
					Accuracy:  location.Accuracy,
					Heading:   location.Heading,
				}
			}
		}
	}

	return nearestPoint, minDistance
}

func (mc *MapboxClient) calculateRouteSnapConfidence(distance float64, maxRadius int) float64 {
	if distance > float64(maxRadius) {
		return 0.0
	}
	// Linear confidence decay: 1.0 at distance 0, 0.0 at maxRadius
	return 1.0 - (distance / float64(maxRadius))
}

func (mc *MapboxClient) calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	// Haversine formula for distance calculation
	const R = 6371000 // Earth's radius in meters

	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
		math.Sin(dLng/2)*math.Sin(dLng/2)
	
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	return R * c
}