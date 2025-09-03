package rest

// import (
// 	"context"
// 	"log"
// 	"net/http"
// 	"time"

// 	"github.com/bwise1/waze_kibris/internal/http/mapbox"
// 	"github.com/bwise1/waze_kibris/util"
// 	"github.com/bwise1/waze_kibris/util/tracing"
// 	"github.com/bwise1/waze_kibris/util/values"
// 	"github.com/go-chi/chi/v5"
// )

// // LocationSnappingRoutes defines routes for location snapping functionality
// func (api *API) LocationSnappingRoutes() chi.Router {
// 	mux := chi.NewRouter()

// 	mux.Group(func(r chi.Router) {
// 		r.Use(api.RequireLogin) // Require authentication for all location snapping endpoints

// 		// Smart location snapping (route-aware)
// 		r.Method(http.MethodPost, "/snap", Handler(api.SnapLocationHandler))

// 		// Batch location snapping for GPS tracks
// 		r.Method(http.MethodPost, "/snap/batch", Handler(api.BatchSnapLocationHandler))

// 		// Road snapping specifically for reports
// 		r.Method(http.MethodPost, "/snap/report", Handler(api.SnapReportLocationHandler))
// 	})

// 	return mux
// }

// // SnapLocationRequest represents the request body for location snapping
// type SnapLocationRequest struct {
// 	// Current user location
// 	Location mapbox.LocationPoint `json:"location" validate:"required"`

// 	// Navigation context
// 	IsNavigating  bool               `json:"is_navigating"`
// 	ActiveRoute   *mapbox.LineString `json:"active_route,omitempty"`   // Current navigation route
// 	RouteProgress float64            `json:"route_progress,omitempty"` // Progress along route (0.0-1.0)

// 	// Snapping preferences
// 	Profile      string `json:"profile,omitempty"`      // driving, walking, cycling
// 	SnapRadius   int    `json:"snap_radius,omitempty"`  // Max snap distance in meters
// 	OppositeSide bool   `json:"opposite_side,omitempty"` // For report placement

// 	// Performance options
// 	UseCache bool `json:"use_cache,omitempty"` // Use cached results if available
// }

// // BatchSnapLocationRequest for processing multiple locations
// type BatchSnapLocationRequest struct {
// 	Locations     []mapbox.LocationPoint `json:"locations" validate:"required,min=1,max=100"`
// 	ActiveRoute   *mapbox.LineString     `json:"active_route,omitempty"`
// 	Profile       string                 `json:"profile,omitempty"`
// 	SnapRadius    int                    `json:"snap_radius,omitempty"`
// 	IsNavigating  bool                   `json:"is_navigating"`
// 	UseCache      bool                   `json:"use_cache,omitempty"`
// }

// // ReportSnapLocationRequest for report-specific snapping
// type ReportSnapLocationRequest struct {
// 	Location     mapbox.LocationPoint   `json:"location" validate:"required"`
// 	ReportType   string                 `json:"report_type" validate:"required"` // POLICE, TRAFFIC, ACCIDENT
// 	OppositeSide bool                   `json:"opposite_side"`                   // Place on opposite side of road
// 	Direction    string                 `json:"direction,omitempty"`             // BOTH_SIDES, MY_SIDE, OPPOSITE_SIDE
// 	ActiveRoute  *mapbox.LineString     `json:"active_route,omitempty"`
// 	IsNavigating bool                   `json:"is_navigating"`
// }

// // SnapLocationHandler handles smart location snapping requests
// func (api *API) SnapLocationHandler(_ http.ResponseWriter, r *http.Request) *ServerResponse {
// 	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

// 	var req SnapLocationRequest
// 	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
// 		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
// 	}

// 	// Validate required fields
// 	if req.Location.Latitude == 0 || req.Location.Longitude == 0 {
// 		return respondWithError(nil, "invalid location coordinates", values.BadRequestBody, &tc)
// 	}

// 	// Set defaults
// 	if req.Profile == "" {
// 		req.Profile = "driving"
// 	}
// 	if req.SnapRadius == 0 {
// 		req.SnapRadius = 25 // 25 meters default
// 	}

// 	// Prepare Mapbox request
// 	snapRequest := mapbox.LocationSnapRequest{
// 		Locations: []mapbox.LocationPoint{req.Location},
// 		Profile:   req.Profile,
// 		SnapRadius: req.SnapRadius,
// 		OppositeSide: req.OppositeSide,
// 	}

// 	// Include active route if navigating
// 	if req.IsNavigating && req.ActiveRoute != nil {
// 		snapRequest.RouteGeometry = req.ActiveRoute
// 		log.Printf("📍 Processing navigation-aware location snap for user at %.6f,%.6f",
// 			req.Location.Latitude, req.Location.Longitude)
// 	} else {
// 		log.Printf("📍 Processing road-snap for user at %.6f,%.6f",
// 			req.Location.Latitude, req.Location.Longitude)
// 	}

// 	// Call Mapbox service
// 	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
// 	defer cancel()

// 	snapResponse, err := api.MapboxClient.SnapLocationToRoad(ctx, snapRequest)
// 	if err != nil {
// 		log.Printf("❌ Location snapping failed: %v", err)
// 		return respondWithError(err, "location snapping failed", values.InternalError, &tc)
// 	}

// 	// Log success
// 	if len(snapResponse.SnappedLocations) > 0 {
// 		snapped := snapResponse.SnappedLocations[0]
// 		log.Printf("✅ Location snapped successfully: %.6f,%.6f -> %.6f,%.6f (distance: %.1fm, type: %s, confidence: %.2f)",
// 			snapped.Original.Latitude, snapped.Original.Longitude,
// 			snapped.Snapped.Latitude, snapped.Snapped.Longitude,
// 			snapped.SnapDistance, snapResponse.SnapType, snapResponse.Confidence)
// 	}

// 	return &ServerResponse{
// 		Message:    "location snapped successfully",
// 		Status:     values.Success,
// 		StatusCode: util.StatusCode(values.Success),
// 		Data:       snapResponse,
// 	}
// }

// // BatchSnapLocationHandler handles batch location snapping
// func (api *API) BatchSnapLocationHandler(_ http.ResponseWriter, r *http.Request) *ServerResponse {
// 	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

// 	var req BatchSnapLocationRequest
// 	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
// 		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
// 	}

// 	// Validate
// 	if len(req.Locations) == 0 {
// 		return respondWithError(nil, "no locations provided", values.BadRequestBody, &tc)
// 	}
// 	if len(req.Locations) > 100 {
// 		return respondWithError(nil, "too many locations (max 100)", values.BadRequestBody, &tc)
// 	}

// 	// Set defaults
// 	if req.Profile == "" {
// 		req.Profile = "driving"
// 	}
// 	if req.SnapRadius == 0 {
// 		req.SnapRadius = 25
// 	}

// 	// Prepare request
// 	snapRequest := mapbox.LocationSnapRequest{
// 		Locations:  req.Locations,
// 		Profile:    req.Profile,
// 		SnapRadius: req.SnapRadius,
// 	}

// 	if req.IsNavigating && req.ActiveRoute != nil {
// 		snapRequest.RouteGeometry = req.ActiveRoute
// 	}

// 	log.Printf("📍 Processing batch location snap for %d locations", len(req.Locations))

// 	// Call service
// 	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second) // Longer timeout for batch
// 	defer cancel()

// 	snapResponse, err := api.MapboxClient.SnapLocationToRoad(ctx, snapRequest)
// 	if err != nil {
// 		log.Printf("❌ Batch location snapping failed: %v", err)
// 		return respondWithError(err, "batch location snapping failed", values.InternalError, &tc)
// 	}

// 	log.Printf("✅ Batch snapped %d locations (type: %s, avg confidence: %.2f)",
// 		len(snapResponse.SnappedLocations), snapResponse.SnapType, snapResponse.Confidence)

// 	return &ServerResponse{
// 		Message:    "batch locations snapped successfully",
// 		Status:     values.Success,
// 		StatusCode: util.StatusCode(values.Success),
// 		Data:       snapResponse,
// 	}
// }

// // SnapReportLocationHandler handles report-specific location snapping
// func (api *API) SnapReportLocationHandler(_ http.ResponseWriter, r *http.Request) *ServerResponse {
// 	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

// 	var req ReportSnapLocationRequest
// 	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
// 		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
// 	}

// 	// Validate
// 	if req.Location.Latitude == 0 || req.Location.Longitude == 0 {
// 		return respondWithError(nil, "invalid location coordinates", values.BadRequestBody, &tc)
// 	}
// 	if req.ReportType == "" {
// 		return respondWithError(nil, "report type is required", values.BadRequestBody, &tc)
// 	}

// 	// Determine opposite side placement based on report type and direction
// 	oppositeSide := req.OppositeSide || req.Direction == "OPPOSITE_SIDE"

// 	// For certain report types, we might want different snapping behavior
// 	snapRadius := 25
// 	switch req.ReportType {
// 	case "POLICE":
// 		snapRadius = 50  // Police can be further from road
// 	case "ACCIDENT":
// 		snapRadius = 30  // Accidents might be slightly off road
// 	case "TRAFFIC":
// 		snapRadius = 20  // Traffic reports should be very close to road
// 	}

// 	snapRequest := mapbox.LocationSnapRequest{
// 		Locations:    []mapbox.LocationPoint{req.Location},
// 		Profile:      "driving", // Always use driving for reports
// 		SnapRadius:   snapRadius,
// 		OppositeSide: oppositeSide,
// 	}

// 	// Include route context if navigating
// 	if req.IsNavigating && req.ActiveRoute != nil {
// 		snapRequest.RouteGeometry = req.ActiveRoute
// 	}

// 	log.Printf("🚨 Processing %s report location snap at %.6f,%.6f (opposite_side: %v)",
// 		req.ReportType, req.Location.Latitude, req.Location.Longitude, oppositeSide)

// 	// Call service
// 	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
// 	defer cancel()

// 	snapResponse, err := api.MapboxClient.SnapLocationToRoad(ctx, snapRequest)
// 	if err != nil {
// 		log.Printf("❌ Report location snapping failed: %v", err)
// 		return respondWithError(err, "report location snapping failed", values.InternalError, &tc)
// 	}

// 	// Add report-specific metadata
// 	response := struct {
// 		*mapbox.LocationSnapResponse
// 		ReportType   string `json:"report_type"`
// 		OppositeSide bool   `json:"opposite_side"`
// 		Direction    string `json:"direction,omitempty"`
// 	}{
// 		LocationSnapResponse: snapResponse,
// 		ReportType:          req.ReportType,
// 		OppositeSide:        oppositeSide,
// 		Direction:           req.Direction,
// 	}

// 	if len(snapResponse.SnappedLocations) > 0 {
// 		snapped := snapResponse.SnappedLocations[0]
// 		log.Printf("✅ %s report location snapped: %.6f,%.6f -> %.6f,%.6f (distance: %.1fm)",
// 			req.ReportType,
// 			snapped.Original.Latitude, snapped.Original.Longitude,
// 			snapped.Snapped.Latitude, snapped.Snapped.Longitude,
// 			snapped.SnapDistance)
// 	}

// 	return &ServerResponse{
// 		Message:    "report location snapped successfully",
// 		Status:     values.Success,
// 		StatusCode: util.StatusCode(values.Success),
// 		Data:       response,
// 	}
// }
