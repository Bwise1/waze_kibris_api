package rest

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/bwise1/waze_kibris/util"
	"github.com/bwise1/waze_kibris/util/tracing"
	"github.com/bwise1/waze_kibris/util/values"
)

type registerFCMRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

type unregisterFCMRequest struct {
	Token string `json:"token,omitempty"`
}

// RegisterFCMToken POST /user/fcm-token — store device token for push (after login).
func (api *API) RegisterFCMToken(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	userID, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		return respondWithError(err, "unable to get user ID from context", values.NotAuthorised, &tc)
	}

	var req registerFCMRequest
	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
	}

	req.Token = strings.TrimSpace(req.Token)
	req.Platform = strings.ToLower(strings.TrimSpace(req.Platform))
	if req.Token == "" || req.Platform == "" {
		return respondWithError(nil, "token and platform are required", values.BadRequestBody, &tc)
	}
	if req.Platform != "android" && req.Platform != "ios" && req.Platform != "web" {
		return respondWithError(nil, "platform must be android, ios, or web", values.BadRequestBody, &tc)
	}

	if err := api.UpsertFCMToken(r.Context(), userID.String(), req.Token, req.Platform); err != nil {
		return respondWithError(err, "failed to save FCM token", values.Error, &tc)
	}

	return &ServerResponse{
		Message:    "FCM token registered",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
	}
}

// UnregisterFCMToken DELETE /user/fcm-token — body `{"token":"..."}` removes one device; empty body removes all.
func (api *API) UnregisterFCMToken(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	userID, err := util.GetUserIDFromContext(r.Context())
	if err != nil {
		return respondWithError(err, "unable to get user ID from context", values.NotAuthorised, &tc)
	}

	bodyBytes, readErr := io.ReadAll(io.LimitReader(r.Body, 8192))
	if readErr != nil {
		return respondWithError(readErr, "unable to read request body", values.BadRequestBody, &tc)
	}
	var req unregisterFCMRequest
	if len(strings.TrimSpace(string(bodyBytes))) > 0 {
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			return respondWithError(err, "unable to decode request", values.BadRequestBody, &tc)
		}
	}

	req.Token = strings.TrimSpace(req.Token)
	if req.Token == "" {
		if err := api.DeleteAllFCMTokensForUser(r.Context(), userID.String()); err != nil {
			return respondWithError(err, "failed to remove FCM tokens", values.Error, &tc)
		}
		return &ServerResponse{
			Message:    "All FCM tokens removed",
			Status:     values.Success,
			StatusCode: util.StatusCode(values.Success),
		}
	}

	if err := api.DeleteFCMToken(r.Context(), userID.String(), req.Token); err != nil {
		return respondWithError(err, "failed to remove FCM token", values.Error, &tc)
	}

	return &ServerResponse{
		Message:    "FCM token removed",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
	}
}
