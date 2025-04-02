package rest

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/bwise1/waze_kibris/internal/http/valhalla"
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

	})

	return mux
}

func (api *API) GetRouteHandler(_ http.ResponseWriter, r *http.Request) *ServerResponse {

	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)
	// Parse request parameters
	var req struct {
		Locations []valhalla.Location `json:"locations"`
		Costing   string              `json:"costing"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body: %v", err)
		return respondWithError(err, "Invalid request payload", values.BadRequestBody, &tc)
	}

	// Create a Valhalla client
	valhallaClient := valhalla.NewValhallaClient(api.Config.ValhallaURL)

	// Fetch the route
	routeResponse, err := valhallaClient.GetRoute(valhalla.RouteRequest{
		Locations: req.Locations,
		Costing:   req.Costing,
	})
	if err != nil {
		log.Println("Error fetching route:", err)
		return respondWithError(err, "Invalid request payload", values.SystemErr, &tc)
	}

	return &ServerResponse{
		Message:    "Votes retrieved successfully",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       routeResponse,
	}

}
