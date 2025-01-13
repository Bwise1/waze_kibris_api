package rest

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/bwise1/waze_kibris/internal/model"
	"github.com/bwise1/waze_kibris/util"
	"github.com/bwise1/waze_kibris/util/tracing"
	"github.com/bwise1/waze_kibris/util/values"
	"github.com/go-chi/chi/v5"
)

func (api *API) SavedLocationRoutes() chi.Router {
	mux := chi.NewRouter()

	mux.Route("/", func(r chi.Router) {
		r.Use(api.RequireLogin)
		r.Method(http.MethodPost, "/", Handler(api.CreateSavedLocation))
		r.Method(http.MethodGet, "/{id}", Handler(api.GetSavedLocation))
		r.Method(http.MethodGet, "/", Handler(api.GetAllSavedLocation))
	})

	// mux.Method(http.MethodPost, "/", Handler(api.CreateSavedLocation))
	// mux.Method(http.MethodGet, "/{id}", Handler(api.GetSavedLocation))
	// mux.Method(http.MethodPut, "/{id}", Handler(api.UpdateSavedLocation))
	// mux.Method(http.MethodDelete, "/{id}", Handler(api.DeleteSavedLocation))
	return mux
}

func (api *API) CreateSavedLocation(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	var ctx = context.TODO()

	var req model.LocationRequest

	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
	}

	userID, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		log.Println("unable to get user ID from context", err)
		return respondWithError(err, "unable to get user ID from context", values.NotAuthorised, &tc)
	}

	if err := util.ValidateStruct(req); err != nil {
		return respondWithError(err, "validation failed", values.BadRequestBody, &tc)
	}

	newLocation := model.SavedLocation{
		UserID:   userID,
		Name:     req.Name,
		Location: util.PointFromLatLon(req.Latitude, req.Longitude),
	}

	err = api.CreateSavedLocationRepo(ctx, newLocation)
	if err != nil {
		return respondWithError(err, "failed to create saved location", values.Error, &tc)
	}

	return &ServerResponse{
		Message:    "Saved location created successfully",
		Status:     values.Created,
		StatusCode: util.StatusCode(values.Created),
		Data:       req,
	}
}

func (api *API) GetAllSavedLocation(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	userID, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		log.Println("unable to get user ID from context", err)
		return respondWithError(err, "Not authorized", values.NotAuthorised, &tc)
	}

	locations, err := api.GetSavedLocationsRepo(r.Context(), userID)
	if err != nil {
		log.Println("failed to get saved locations", err)
		return respondWithError(err, "failed to get saved locations", values.Error, &tc)
	}

	return &ServerResponse{
		Message:    "Saved locations retrieved successfully",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       locations,
	}
}

func (api *API) GetSavedLocation(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		return respondWithError(err, "invalid ID format", values.BadRequestBody, &tc)
	}

	location, err := api.GetSavedLocationRepo(r.Context(), id)
	if err != nil {
		return respondWithError(err, "failed to get saved location", values.Error, &tc)
	}

	lat, lon := util.PointToLatLon(location.Location)

	return &ServerResponse{
		Message:    "Saved location retrieved successfully",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data: map[string]interface{}{
			"id":         location.ID,
			"user_id":    location.UserID,
			"name":       location.Name,
			"latitude":   lat,
			"longitude":  lon,
			"created_at": location.CreatedAt,
		},
	}
}

// func (api *API) UpdateSavedLocation(_ http.ResponseWriter, r *http.Request) *ServerResponse {
// 	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

// 	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
// 	if err != nil {
// 		return respondWithError(err, "invalid ID format", values.BadRequestBody, &tc)
// 	}

// 	var req model.SavedLocation
// 	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
// 		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
// 	}

// 	req.ID = id

// 	err = api.UpdateSavedLocationRepo(r.Context(), req)
// 	if err != nil {
// 		return respondWithError(err, "failed to update saved location", values.Error, &tc)
// 	}

// 	return &ServerResponse{
// 		Message:    "Saved location updated successfully",
// 		Status:     values.Success,
// 		StatusCode: util.StatusCode(values.Success),
// 		Data:       req,
// 	}
// }

// func (api *API) DeleteSavedLocation(_ http.ResponseWriter, r *http.Request) *ServerResponse {
// 	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

// 	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
// 	if err != nil {
// 		return respondWithError(err, "invalid ID format", values.BadRequestBody, &tc)
// 	}

// 	err = api.DeleteSavedLocationRepo(r.Context(), id)
// 	if err != nil {
// 		return respondWithError(err, "failed to delete saved location", values.Error, &tc)
// 	}

// 	return &ServerResponse{
// 		Message:    "Saved location deleted successfully",
// 		Status:     values.Success,
// 		StatusCode: util.StatusCode(values.Success),
// 	}
// }
