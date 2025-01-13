package rest

import (
	"net/http"

	"github.com/bwise1/waze_kibris/internal/model"
	"github.com/bwise1/waze_kibris/util"
	"github.com/bwise1/waze_kibris/util/tracing"
	"github.com/bwise1/waze_kibris/util/values"
	"github.com/go-chi/chi/v5"
)

// func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request)
// func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request)
// func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request)
// func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request)
// func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request)
// func (h *AuthHandler) OAuthCallback(w http.ResponseWriter, r *http.Request)

func (api *API) AuthRoutes() chi.Router {
	mux := chi.NewRouter()

	mux.Method(http.MethodPost, "/register", Handler(api.Register))
	mux.Method(http.MethodPost, "/login", Handler(api.Login))
	mux.Method(http.MethodPost, "/verify", Handler(api.VerifyCode))
	mux.Method(http.MethodPost, "/resend", Handler(api.ResendCode))
	return mux
}

func (api *API) Register(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	var req model.RegisterRequest
	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
	}

	user, status, message, err := api.CreateNewUser(req)
	if err != nil {

		return respondWithError(err, message, status, &tc)
	}

	return &ServerResponse{
		Message:    message,
		Status:     status,
		StatusCode: util.StatusCode(status),
		Data:       user,
	}
}

func (api *API) Login(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	var req model.LoginRequest
	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
	}

	user, status, message, err := api.LoginUser(req)
	if err != nil {
		return respondWithError(err, message, status, &tc)
	}

	return &ServerResponse{
		Message:    message,
		Status:     status,
		StatusCode: util.StatusCode(status),
		Data:       user,
	}
}

func (api *API) VerifyCode(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	var req model.VerifyCodeRequest
	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
	}

	user, status, message, err := api.VerifyCodeHelper(req)
	if err != nil {
		return respondWithError(err, message, status, &tc)
	}

	return &ServerResponse{
		Message:    message,
		Status:     status,
		StatusCode: util.StatusCode(status),
		Data:       user,
	}
}

func (api *API) ResendCode(w http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	var req model.ResendCodeRequest
	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
	}

	status, message, err := api.ResendVerificationCode(req)
	if err != nil {
		return respondWithError(err, message, status, &tc)
	}

	return &ServerResponse{
		Message:    message,
		Status:     status,
		StatusCode: util.StatusCode(status),
	}
}
