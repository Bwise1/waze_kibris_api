package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/bwise1/waze_kibris/internal/http/mapbox"
	"github.com/bwise1/waze_kibris/util"
	"github.com/bwise1/waze_kibris/util/tracing"
	"github.com/bwise1/waze_kibris/util/values"
	"github.com/go-chi/chi/v5"
)

func (api *API) RoutingRoutes() chi.Router {
	mux := chi.NewRouter()

	mux.Group(func(r chi.Router) {
		// r.Use(api.RequireLogin)
		r.Method(http.MethodPost, "/", Handler(api.GetRouteHandler))
		r.Method(http.MethodPost, "/enhanced", Handler(api.GetRouteHandler)) // Alias for enhanced navigation
	})

	return mux
}

// Location represents a coordinate pair for routing
type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// RouteRequest represents the request payload for route calculation
type RouteRequest struct {
	Locations          []Location `json:"locations"`
	Profile            string     `json:"profile,omitempty"` // "driving", "driving-traffic", "walking", "cycling"
	Alternatives       bool       `json:"alternatives,omitempty"`
	VoiceInstructions  bool       `json:"voice_instructions,omitempty"`
	BannerInstructions bool       `json:"banner_instructions,omitempty"`
	VoiceUnits         string     `json:"voice_units,omitempty"` // "metric" or "imperial"
	Language           string     `json:"language,omitempty"`    // "en", "es", etc.
	RoundaboutExits    bool       `json:"roundabout_exits,omitempty"`
	WaypointNames      bool       `json:"waypoint_names,omitempty"`
	Approaches         string     `json:"approaches,omitempty"` // "unrestricted", "curb", etc.
	Exclude            string     `json:"exclude,omitempty"`    // "toll", "ferry", "motorway"
}

func (api *API) GetRouteHandler(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	// Parse request parameters
	var req RouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body: %v", err)
		return respondWithError(err, "Invalid request payload", values.BadRequestBody, &tc)
	}

	if req.Locations == nil || len(req.Locations) < 2 {
		log.Printf("No locations provided or insufficient locations")
		return respondWithError(nil, "At least 2 locations required", values.BadRequestBody, &tc)
	}

	// Set defaults
	if req.Profile == "" {
		req.Profile = "driving" // Use basic driving profile for lane guidance support
	}

	// Use existing Mapbox client
	if api.MapboxClient == nil {
		return respondWithError(nil, "Mapbox client not configured", values.SystemErr, &tc)
	}

	// Convert locations to coordinate strings in Mapbox format (lng,lat)
	coordinates := make([]string, len(req.Locations))
	for i, loc := range req.Locations {
		coordinates[i] = fmt.Sprintf("%s,%s",
			strconv.FormatFloat(loc.Lng, 'f', 6, 64),
			strconv.FormatFloat(loc.Lat, 'f', 6, 64))
	}

	// Configure navigation options
	navOptions := &mapbox.NavigationOptions{
		VoiceInstructions:  req.VoiceInstructions,
		BannerInstructions: req.BannerInstructions,
		VoiceUnits:         req.VoiceUnits,
		Language:           req.Language,
		RoundaboutExits:    req.RoundaboutExits,
		WaypointNames:      req.WaypointNames,
		Approaches:         req.Approaches,
		Exclude:            req.Exclude,
	}

	// Set defaults if not specified
	if navOptions.VoiceUnits == "" {
		navOptions.VoiceUnits = "metric"
	}
	if navOptions.Language == "" {
		navOptions.Language = "en"
	}
	// Enable voice and banner instructions by default
	if !req.VoiceInstructions && !req.BannerInstructions {
		navOptions.VoiceInstructions = true
		navOptions.BannerInstructions = true
		navOptions.RoundaboutExits = true
		// Only enable waypoint_names if explicitly requested
		// navOptions.WaypointNames = true
	}

	// Fetch the route with enhanced navigation features
	routeResponse, err := api.MapboxClient.DirectionsWithNavigation(
		context.TODO(),
		coordinates,
		req.Profile,
		req.Alternatives,
		navOptions,
	)
	if err != nil {
		log.Printf("Error fetching Mapbox route: %v", err)
		return respondWithError(err, "Failed to calculate route", values.SystemErr, &tc)
	}

	return &ServerResponse{
		Message:    "Routes retrieved successfully with enhanced navigation data",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       routeResponse,
	}
}
