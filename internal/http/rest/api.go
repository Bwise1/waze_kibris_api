package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bwise1/waze_kibris/config"
	deps "github.com/bwise1/waze_kibris/internal/debs"
	smtp "github.com/bwise1/waze_kibris/util/email"
	"github.com/bwise1/waze_kibris/util/values"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultIdleTimeout    = time.Minute
	defaultReadTimeout    = 5 * time.Second
	defaultWriteTimeout   = 10 * time.Second
	defaultShutdownPeriod = 30 * time.Second
)

type Handler func(w http.ResponseWriter, r *http.Request) *ServerResponse

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := h(w, r)
	respByte, err := json.Marshal(resp)
	if err != nil {
		writeErrorResponse(w, err, values.Error, "unable to marshal server response")
		return
	}
	writeJSONResponse(w, respByte, resp.StatusCode)
}

type API struct {
	Server *http.Server
	Config *config.Config
	Deps   *deps.Dependencies
	Mailer *smtp.Mailer
	DB     *pgxpool.Pool
}

func (api *API) Serve() error {
	api.Server = &http.Server{
		Addr:         fmt.Sprintf(":%d", api.Config.Port),
		IdleTimeout:  defaultIdleTimeout,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		Handler:      api.setUpServerHandler(),
	}
	return api.Server.ListenAndServe()
}

func (api *API) setUpServerHandler() http.Handler {
	mux := chi.NewRouter()
	mux.Use(RequestTracing)

	mux.Get("/",
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello, World!"))
		},
	)

	mux.Mount("/auth", api.AuthRoutes())
	mux.Mount("/reports", api.ReportRoutes())
	mux.Mount("/saved-locations", api.SavedLocationRoutes())

	return mux
}

func (a *API) Shutdown() error {
	// err := a.Deps.DAL.DB.Close()
	// if err != nil {
	// 	return err
	// }

	err := a.Server.Shutdown(context.Background())
	if err != nil {
		return err
	}
	return nil
}
