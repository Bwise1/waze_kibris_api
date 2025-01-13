// internal/http/rest/reports_handler.go
package rest

import (
	"log"
	"net/http"
	"strconv"

	"github.com/bwise1/waze_kibris/internal/model"
	"github.com/bwise1/waze_kibris/util"
	"github.com/bwise1/waze_kibris/util/tracing"
	"github.com/bwise1/waze_kibris/util/values"
	"github.com/go-chi/chi/v5"
)

func (api *API) ReportRoutes() chi.Router {
	mux := chi.NewRouter()

	mux.Method(http.MethodPost, "/create", Handler(api.CreateReport))
	mux.Method(http.MethodGet, "/nearby", Handler(api.GetNearbyReports))
	mux.Method(http.MethodGet, "/all", Handler(api.GetAllReports))
	return mux
}

func (api *API) CreateReport(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	var req model.Report
	if decodeErr := util.DecodeJSONBody(&tc, r.Body, &req); decodeErr != nil {
		return respondWithError(decodeErr, "unable to decode request", values.BadRequestBody, &tc)
	}

	reportID, status, message, err := api.CreateReportHelper(r.Context(), req)
	if err != nil {
		return respondWithError(err, message, status, &tc)
	}

	return &ServerResponse{
		Message:    message,
		Status:     status,
		StatusCode: util.StatusCode(status),
		Data:       map[string]string{"report_id": reportID},
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
	if err != nil {
		return respondWithError(err, "invalid radius", values.BadRequestBody, &tc)
	}

	reports, status, message, err := api.GetNearbyReportsHelper(r.Context(), longitude, latitude, radius)
	if err != nil {
		return respondWithError(err, message, status, &tc)
	}

	return &ServerResponse{
		Message:    message,
		Status:     status,
		StatusCode: util.StatusCode(status),
		Data:       reports,
	}
}

func (api *API) GetAllReports(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)

	reports, status, message, err := api.GetAllReportsHelper(r.Context())
	if err != nil {
		log.Println("error getting all reports", err)
		return respondWithError(err, message, status, &tc)
	}

	return &ServerResponse{
		Message:    message,
		Status:     status,
		StatusCode: util.StatusCode(status),
		Data:       reports,
	}
}
