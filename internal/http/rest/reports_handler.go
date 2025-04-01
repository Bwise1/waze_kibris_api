package rest

import (
	"log"
	"net/http"
	"strconv"
	"time"

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

func (api *API) CreateReport(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	var req model.CreateReportRequest
	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
	}

	userId, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		return respondWithError(err, "unable to get user ID from context", values.NotAuthorised, &tc)
	}

	req.UserID = userId
	req.ExpiresAt = time.Now().Add(time.Hour * 6) // Default expiry time is 1 days

	// // Handle image upload if provided
	// if req.ImageURL != "" {
	// 	imageURL, err := api.Deps.Cloudinary.UploadImage(r.Context(), req.ImageURL, "reports")
	// 	if err != nil {
	// 		return respondWithError(err, "failed to upload image", values.Error, &tc)
	// 	}
	// 	req.ImageURL = imageURL
	// }

	newReport, status, message, err := api.CreateReportHelper(r.Context(), req)
	if err != nil {
		return respondWithError(err, message, status, &tc)
	}

	return &ServerResponse{
		Message:    message,
		Status:     status,
		StatusCode: util.StatusCode(status),
		Data:       newReport,
	}
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
