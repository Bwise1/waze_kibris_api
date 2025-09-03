package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/bwise1/waze_kibris/internal/http/mapbox"
	"github.com/bwise1/waze_kibris/internal/model"
	"github.com/bwise1/waze_kibris/util"
	"github.com/bwise1/waze_kibris/util/tracing"
	"github.com/bwise1/waze_kibris/util/values"
	"github.com/go-chi/chi/v5"
)

func (api *API) ReportRoutes() chi.Router {
	mux := chi.NewRouter()

	mux.Group(func(r chi.Router) {
		r.Use(api.RequireLogin)
		r.Method(http.MethodPost, "/", Handler(api.CreateReport))
		r.Method(http.MethodGet, "/nearby", Handler(api.GetNearbyReports))

		r.Method(http.MethodGet, "/{reportID}", Handler(api.GetReportByID))
		r.Method(http.MethodPut, "/{id}", Handler(api.UpdateReport))
		r.Method(http.MethodDelete, "/{id}", Handler(api.DeleteReport))
		r.Method(http.MethodPost, "/{reportID}/votes", Handler(api.VoteOnReport))
		r.Method(http.MethodGet, "/{reportID}/votes", Handler(api.GetVotes))
		r.Method(http.MethodPost, "/{reportID}/comments", Handler(api.CommentOnReport))
		r.Method(http.MethodGet, "/{reportID}/comments", Handler(api.GetComments))
	})

	return mux
}

// Enhanced CreateReportRequest with road snapping options
type EnhancedCreateReportRequest struct {
	model.CreateReportRequest

	// Road snapping options
	EnableRoadSnapping bool   `json:"enable_road_snapping,omitempty"` // Default: true
	OppositeSide       bool   `json:"opposite_side,omitempty"`        // Place on opposite side of road
	Direction          string `json:"direction,omitempty"`            // BOTH_SIDES, MY_SIDE, OPPOSITE_SIDE
}

func (api *API) CreateReport(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	var req EnhancedCreateReportRequest
	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
	}

	userId, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		return respondWithError(err, "unable to get user ID from context", values.NotAuthorised, &tc)
	}

	req.UserID = userId
	req.ExpiresAt = time.Now().Add(time.Hour * 6) // Default expiry time is 6 hours

	// Apply road snapping to report location (enabled by default)
	originalLat := req.Latitude
	originalLng := req.Longitude
	snapApplied := false

	if req.EnableRoadSnapping == false {
		// Explicitly disabled
		log.Printf("📍 Road snapping disabled for %s report at %.6f,%.6f", req.Type, req.Latitude, req.Longitude)
	} else {
		// Apply road snapping (default behavior)
		snappedLat, snappedLng, err := api.snapReportToRoad(r.Context(), req.Latitude, req.Longitude, req.Type, req.OppositeSide || req.Direction == "OPPOSITE_SIDE")
		if err != nil {
			log.Printf("⚠️ Road snapping failed for %s report: %v. Using original coordinates.", req.Type, err)
		} else {
			req.Latitude = snappedLat
			req.Longitude = snappedLng
			snapApplied = true

			log.Printf("✅ %s report location snapped: %.6f,%.6f -> %.6f,%.6f",
				req.Type, originalLat, originalLng, req.Latitude, req.Longitude)
		}
	}

	// // Handle image upload if provided
	// if req.ImageURL != "" {
	// 	imageURL, err := api.Deps.Cloudinary.UploadImage(r.Context(), req.ImageURL, "reports")
	// 	if err != nil {
	// 		return respondWithError(err, "failed to upload image", values.Error, &tc)
	// 	}
	// 	req.ImageURL = imageURL
	// }

	newReport, status, message, err := api.CreateReportHelper(r.Context(), req.CreateReportRequest)
	if err != nil {
		return respondWithError(err, message, status, &tc)
	}

	// Add snapping metadata to response
	responseData := struct {
		*model.CreateReportResponse
		RoadSnapping *struct {
			Applied      bool    `json:"applied"`
			OriginalLat  float64 `json:"original_lat,omitempty"`
			OriginalLng  float64 `json:"original_lng,omitempty"`
			SnapDistance float64 `json:"snap_distance,omitempty"`
			SnapType     string  `json:"snap_type,omitempty"`
			OppositeSide bool    `json:"opposite_side,omitempty"`
		} `json:"road_snapping,omitempty"`
	}{
		CreateReportResponse: &newReport,
	}

	if snapApplied {
		// Calculate snap distance
		snapDistance := calculateDistance(originalLat, originalLng, req.Latitude, req.Longitude)

		responseData.RoadSnapping = &struct {
			Applied      bool    `json:"applied"`
			OriginalLat  float64 `json:"original_lat,omitempty"`
			OriginalLng  float64 `json:"original_lng,omitempty"`
			SnapDistance float64 `json:"snap_distance,omitempty"`
			SnapType     string  `json:"snap_type,omitempty"`
			OppositeSide bool    `json:"opposite_side,omitempty"`
		}{
			Applied:      true,
			OriginalLat:  originalLat,
			OriginalLng:  originalLng,
			SnapDistance: snapDistance,
			SnapType:     "road",
			OppositeSide: req.OppositeSide || req.Direction == "OPPOSITE_SIDE",
		}
	}

	return &ServerResponse{
		Message:    message,
		Status:     status,
		StatusCode: util.StatusCode(status),
		Data:       responseData,
	}
}

// snapReportToRoad uses Map Matching API to snap report location to nearest road
func (api *API) snapReportToRoad(ctx context.Context, lat, lng float64, reportType string, oppositeSide bool) (float64, float64, error) {
	// Set snap radius based on report type
	snapRadius := 25
	switch reportType {
	case "police":
		snapRadius = 50 // Police can be further from road
	case "accident":
		snapRadius = 30 // Accidents might be slightly off road
	case "traffic":
		snapRadius = 20 // Traffic reports should be close to road
	}

	// Call Map Matching API directly
	coordinates := fmt.Sprintf("%.6f,%.6f", lng, lat) // Mapbox expects lng,lat
	baseURL := fmt.Sprintf("https://api.mapbox.com/matching/v5/mapbox/driving/%s", coordinates)

	params := url.Values{}
	params.Set("access_token", api.MapboxClient.APIKey)
	params.Set("radiuses", fmt.Sprintf("%d", snapRadius))
	params.Set("geometries", "geojson")

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return lat, lng, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := api.MapboxClient.Client.Do(req)
	if err != nil {
		return lat, lng, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return lat, lng, fmt.Errorf("mapbox API error: status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return lat, lng, fmt.Errorf("failed to read response: %w", err)
	}

	var mapMatchingResp mapbox.MapMatchingResponse
	if err := json.Unmarshal(bodyBytes, &mapMatchingResp); err != nil {
		return lat, lng, fmt.Errorf("failed to decode response: %w", err)
	}

	if mapMatchingResp.Code != "Ok" || len(mapMatchingResp.Tracepoints) == 0 {
		return lat, lng, fmt.Errorf("no match found")
	}

	// Get the snapped coordinates
	tracepoint := mapMatchingResp.Tracepoints[0]
	if tracepoint.Location == nil || len(tracepoint.Location) < 2 {
		return lat, lng, fmt.Errorf("invalid tracepoint location")
	}

	snappedLng := tracepoint.Location[0]
	snappedLat := tracepoint.Location[1]

	// Apply opposite side offset if requested
	if oppositeSide {
		// Simple perpendicular offset of ~15 meters
		offsetDistance := 15.0 / 111111.0 // rough degrees per meter
		snappedLat += offsetDistance      // This is simplified - in production you'd calculate proper perpendicular
	}

	return snappedLat, snappedLng, nil
}

// Helper functions
func calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	// Simple distance calculation (Haversine formula simplified)
	const R = 6371000 // Earth's radius in meters

	dLat := (lat2 - lat1) * 0.017453292519943295 // π/180
	dLng := (lng2 - lng1) * 0.017453292519943295

	a := 0.5 - 0.5*((1-dLat*dLat/2)*2-1) +
		((1-lat1*0.017453292519943295*lat1*0.017453292519943295/2)*2-1)*
			((1-lat2*0.017453292519943295*lat2*0.017453292519943295/2)*2-1)*
			0.5*(1-((1-dLng*dLng/2)*2-1))

	return R * 2 * 0.7071067811865476 * ((1 - a*a*a*a/(1+a*a)) / (1 - a*a)) // Simplified asin
}

func (api *API) GetReportByID(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	reportID := chi.URLParam(r, "reportID")

	report, status, message, err := api.GetReportByIDHelper(r.Context(), reportID)
	if err != nil {
		return respondWithError(err, message, status, &tc)
	}

	return &ServerResponse{
		Message:    message,
		Status:     status,
		StatusCode: util.StatusCode(status),
		Data:       report,
	}
}

func (api *API) GetNearbyReports(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	longitude, err := strconv.ParseFloat(r.URL.Query().Get("longitude"), 64)
	if err != nil {
		return respondWithError(err, "invalid longitude", values.BadRequestBody, &tc)
	}

	latitude, err := strconv.ParseFloat(r.URL.Query().Get("latitude"), 64)
	if err != nil {
		return respondWithError(err, "invalid latitude", values.BadRequestBody, &tc)
	}

	radius, err := strconv.ParseFloat(r.URL.Query().Get("radius"), 64)
	if err != nil || radius <= 0 {
		radius = 100 // Default radius in meters
	}

	types := r.URL.Query()["type"]
	status := r.URL.Query().Get("status")
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if err != nil {
		pageSize = 10
	}

	params := model.NearbyReportsParams{
		Latitude:  latitude,
		Longitude: longitude,
		Radius:    radius,
		Types:     types,
		Status:    status,
		Page:      page,
		PageSize:  pageSize,
	}

	reports, status, message, err := api.GetNearbyReportsHelper(r.Context(), params)
	if err != nil {
		return respondWithError(err, message, status, &tc)
	}
	if reports == nil {
		reports = []model.Report{}
	}
	return &ServerResponse{
		Message:    message,
		Status:     status,
		StatusCode: util.StatusCode(status),
		Data:       reports,
	}
}

// func (api *API) GetAllReports(_ http.ResponseWriter, r *http.Request) *ServerResponse {
//     tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

//     reports, status, message, err := api.GetAllReportsHelper(r.Context())
//     if err != nil {
//         log.Println("error getting all reports", err)
//         return respondWithError(err, message, status, &tc)
//     }

//     return &ServerResponse{
//         Message:    message,
//         Status:     status,
//         StatusCode: util.StatusCode(status),
//         Data:       reports,
//     }
// }

func (api *API) UpdateReport(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	idParam := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return respondWithError(err, "invalid ID format", values.BadRequestBody, &tc)
	}

	var req model.UpdateReportRequest
	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
	}

	// Validate the request
	if err := util.ValidateStruct(req); err != nil {
		return respondWithError(err, "validation failed", values.BadRequestBody, &tc)
	}

	userId, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		return respondWithError(err, "unable to get user ID from context", values.NotAuthorised, &tc)
	}

	req.ID = id // Correctly assign the parsed integer ID to req.ID

	report := model.Report{
		ID:           req.ID,
		UserID:       userId,
		Type:         req.Type,
		Subtype:      &req.Subtype,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		Description:  &req.Description,
		Severity:     req.Severity,
		Active:       req.Active,
		Resolved:     req.Resolved,
		ExpiresAt:    req.ExpiresAt,
		ImageURL:     &req.ImageURL,
		ReportSource: req.ReportSource,
		ReportStatus: req.ReportStatus,
	}

	status, message, err := api.UpdateReportHelper(r.Context(), report)
	if err != nil {
		return respondWithError(err, message, status, &tc)
	}

	return &ServerResponse{
		Message:    message,
		Status:     status,
		StatusCode: util.StatusCode(status),
		Data:       report,
	}
}

func (api *API) DeleteReport(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	id := chi.URLParam(r, "id")

	userID, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		return respondWithError(err, "unable to get user ID from context", values.NotAuthorised, &tc)
	}

	status, message, err := api.DeleteReportHelper(r.Context(), id, userID.String())
	if err != nil {
		return respondWithError(err, message, status, &tc)
	}

	return &ServerResponse{
		Message:    message,
		Status:     status,
		StatusCode: util.StatusCode(status),
	}
}

func (api *API) VoteOnReport(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	reportID := chi.URLParam(r, "reportID")
	id, err := strconv.ParseInt(reportID, 10, 64)
	if err != nil {
		return respondWithError(err, "invalid report ID", values.BadRequestBody, &tc)
	}

	var req struct {
		VoteType string `json:"vote_type"`
	}
	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
	}

	userID, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		return respondWithError(err, "unable to get user ID from context", values.NotAuthorised, &tc)
	}

	vote := model.Vote{
		ReportID: id,
		UserID:   userID,
		VoteType: req.VoteType,
	}

	err = api.AddVoteRepo(r.Context(), vote)
	if err != nil {
		return respondWithError(err, "failed to add vote", values.Error, &tc)
	}

	// Optionally, update the vote counts in the report
	// You can implement logic to fetch the current vote counts and update them

	return &ServerResponse{
		Message:    "Vote added successfully",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
	}
}

func (api *API) CommentOnReport(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	reportID := chi.URLParam(r, "reportID")
	id, err := util.StringToUUID(reportID)
	if err != nil {
		return respondWithError(err, "invalid report ID", values.BadRequestBody, &tc)
	}

	var req struct {
		Content string `json:"content"`
	}
	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
	}

	userID, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		return respondWithError(err, "unable to get user ID from context", values.NotAuthorised, &tc)
	}

	comment := model.Comment{
		ReportID: id,
		UserID:   userID,
		Comment:  req.Content,
	}

	err = api.AddCommentRepo(r.Context(), comment)
	if err != nil {
		return respondWithError(err, "failed to add comment", values.Error, &tc)
	}

	return &ServerResponse{
		Message:    "Comment added successfully",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       comment,
	}
}

func (api *API) GetComments(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	reportID := chi.URLParam(r, "reportID")

	comments, err := api.GetCommentsRepo(r.Context(), reportID)
	if err != nil {
		return respondWithError(err, "failed to get comments", values.Error, &tc)
	}

	return &ServerResponse{
		Message:    "Comments retrieved successfully",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       comments,
	}
}

func (api *API) GetVotes(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)
	reportID := chi.URLParam(r, "reportID")

	votes, err := api.GetVotesRepo(r.Context(), reportID)
	if err != nil {
		log.Println("error getting votes", err)
		return respondWithError(err, "failed to get votes", values.Error, &tc)
	}

	return &ServerResponse{
		Message:    "Votes retrieved successfully",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       votes,
	}
}
