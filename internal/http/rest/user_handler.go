package rest

import (
	"net/http"

	"github.com/bwise1/waze_kibris/internal/model"
	"github.com/bwise1/waze_kibris/util"
	"github.com/bwise1/waze_kibris/util/tracing"
	"github.com/bwise1/waze_kibris/util/values"
	"github.com/go-chi/chi/v5"
)

func (api *API) UserRoutes() chi.Router {
	mux := chi.NewRouter()

	mux.Route("/", func(r chi.Router) {
		r.Use(api.RequireLogin)
		r.Method(http.MethodGet, "/profile", Handler(api.GetProfile))
		r.Method(http.MethodPut, "/profile", Handler(api.UpdateProfile))
		r.Method(http.MethodPut, "/password", Handler(api.ChangePassword))
		r.Method(http.MethodPut, "/language", Handler(api.UpdateLanguage))
		r.Method(http.MethodDelete, "/account", Handler(api.DeleteAccount))
	})

	return mux
}

func (api *API) GetProfile(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	userID, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		return respondWithError(err, "unable to get user ID from context", values.NotAuthorised, &tc)
	}

	user, err := api.GetUserByID(r.Context(), userID.String())
	if err != nil {
		return respondWithError(err, "failed to get user profile", values.Error, &tc)
	}

	return &ServerResponse{
		Message:    "User profile retrieved successfully",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       user,
	}
}

func (api *API) UpdateProfile(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	userID, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		return respondWithError(err, "unable to get user ID from context", values.NotAuthorised, &tc)
	}

	var req model.User
	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
	}

	req.ID = userID

	err = api.UpdateUserRepo(r.Context(), req)
	if err != nil {
		return respondWithError(err, "failed to update user profile", values.Error, &tc)
	}

	return &ServerResponse{
		Message:    "User profile updated successfully",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       req,
	}
}

func (api *API) ChangePassword(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	userID, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		return respondWithError(err, "unable to get user ID from context", values.NotAuthorised, &tc)
	}

	var req model.ChangePasswordRequest
	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
	}

	err = api.ChangePasswordRepo(r.Context(), userID.String(), req.OldPassword, req.NewPassword)
	if err != nil {
		return respondWithError(err, "failed to change password", values.Error, &tc)
	}

	return &ServerResponse{
		Message:    "Password changed successfully",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
	}
}

func (api *API) UpdateLanguage(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	userID, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		return respondWithError(err, "unable to get user ID from context", values.NotAuthorised, &tc)
	}

	var req model.UpdateLanguageRequest
	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
	}

	err = api.UpdateLanguageRepo(r.Context(), userID.String(), req.Language)
	if err != nil {
		return respondWithError(err, "failed to update language", values.Error, &tc)
	}

	return &ServerResponse{
		Message:    "Language updated successfully",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
	}
}

func (api *API) DeleteAccount(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	userID, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		return respondWithError(err, "unable to get user ID from context", values.NotAuthorised, &tc)
	}

	err = api.DeleteUserRepo(r.Context(), userID.String())
	if err != nil {
		return respondWithError(err, "failed to delete account", values.Error, &tc)
	}

	return &ServerResponse{
		Message:    "Account deleted successfully",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
	}
}
