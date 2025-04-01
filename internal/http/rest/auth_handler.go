package rest

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/bwise1/waze_kibris/internal/model"
	"github.com/bwise1/waze_kibris/util"
	"github.com/bwise1/waze_kibris/util/tracing"
	"github.com/bwise1/waze_kibris/util/values"
	"github.com/go-chi/chi/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request)
// func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request)
// func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request)
// func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request)
// func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request)
// func (h *AuthHandler) OAuthCallback(w http.ResponseWriter, r *http.Request)

var googleOauthConfig *oauth2.Config

func (api *API) Init() {
	log.Println("Initializing google auth")
	googleOauthConfig = &oauth2.Config{
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		ClientID:     api.Config.GoogleClientID,
		ClientSecret: api.Config.GoogleClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}
}

func (api *API) AuthRoutes() chi.Router {
	mux := chi.NewRouter()

	mux.Method(http.MethodPost, "/register", Handler(api.Register))
	mux.Method(http.MethodPost, "/login", Handler(api.Login))
	mux.Method(http.MethodPost, "/verify", Handler(api.VerifyCode))
	mux.Method(http.MethodPost, "/resend", Handler(api.ResendCode))
	mux.Method(http.MethodPost, "/google/create", Handler(api.CreateAccountWithGoogle))
	mux.Method(http.MethodPost, "/refresh", Handler(api.RefreshTokenHandler)) // Add this line
	mux.Method(http.MethodPost, "/google/login", Handler(api.MobileGoogleLogin))
	return mux
}

func (api *API) CreateAccountWithGoogle(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	var req struct {
		AccessToken string `json:"access_token"`
	}
	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
	}

	token := &oauth2.Token{AccessToken: req.AccessToken}
	client := googleOauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return respondWithError(err, "failed to get user info", values.Error, &tc)
	}
	defer resp.Body.Close()

	var userInfo model.UserInfo

	//log.Println(resp.Body)
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return respondWithError(err, "failed to decode user info", values.Error, &tc)
	}

	log.Println(userInfo)
	// Check if user already exists
	Euser, err := api.GetUserByEmail(r.Context(), userInfo.Email)
	log.Println(Euser)
	if err == nil {
		return respondWithError(nil, "user already exists", values.Conflict, &tc)
	}

	// Create a new user
	user := model.User{
		ID:           util.GenerateUUID(),
		Email:        userInfo.Email,
		FirstName:    &userInfo.GivenName,
		LastName:     &userInfo.FamilyName,
		AuthProvider: "google",
		IsVerified:   userInfo.VerifiedEmail,
	}
	err = api.CreateNewUserRepo(r.Context(), user)
	if err != nil {
		return respondWithError(err, "failed to create new user", values.Error, &tc)
	}

	// Generate JWT token
	tokenString, _, err := api.createToken(user.ID.String())
	if err != nil {
		return respondWithError(err, "failed to create token", values.Error, &tc)
	}

	return &ServerResponse{
		Message:    "Account created successfully",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data: map[string]interface{}{
			"token": tokenString,
			"user":  user,
		},
	}
}

func (api *API) LoginWithGoogle(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	var req struct {
		AccessToken string `json:"access_token"`
	}
	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
	}

	token := &oauth2.Token{AccessToken: req.AccessToken}
	client := googleOauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return respondWithError(err, "failed to get user info", values.Error, &tc)
	}
	defer resp.Body.Close()

	var userInfo model.UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return respondWithError(err, "failed to decode user info", values.Error, &tc)
	}

	// Check if user exists
	user, err := api.GetUserByEmail(r.Context(), userInfo.Email)
	if err != nil {
		return respondWithError(err, "user does not exist", values.NotFound, &tc)
	}

	// Generate JWT token
	tokenString, _, err := api.createToken(user.ID.String())
	if err != nil {
		return respondWithError(err, "failed to create token", values.Error, &tc)
	}

	return &ServerResponse{
		Message:    "Login successful",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data: map[string]interface{}{
			"token": tokenString,
			"user":  user,
		},
	}
}

func (api *API) MobileGoogleLogin(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	var req struct {
		IDToken string `json:"id_token"`
	}
	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
	}

	user, status, message, err := api.GoogleLogin(req.IDToken)
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

func (api *API) RefreshTokenHandler(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return respondWithError(err, "Invalid request body", values.BadRequestBody, nil)
	}

	// Refresh the access token
	log.Println("refreshing token")
	accessToken, newRefreshToken, err := api.RefreshAccessToken(r.Context(), req.RefreshToken)
	if err != nil {
		log.Println("error refreshing token", err)
		return respondWithError(err, "Failed to refresh tokens", values.NotAuthorised, nil)
	}

	// Return the new tokens
	return &ServerResponse{
		Message:    "Tokens refreshed successfully",
		Status:     values.Success,
		StatusCode: http.StatusOK,
		Data: map[string]string{
			"access_token":  accessToken,
			"refresh_token": newRefreshToken,
		},
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
