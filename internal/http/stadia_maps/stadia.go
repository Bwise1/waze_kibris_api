package stadiamaps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/pkg/errors"
)

const (
	defaultStadiaBaseURL = "https://api.stadiamaps.com"
)

// Client handles communication with the Stadia Maps API.
type Client struct {
	BaseURL    *url.URL
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new Stadia Maps API client with default timeout.
func NewClient(apiKey string) *Client {
	baseURL, _ := url.Parse(defaultStadiaBaseURL)
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				TLSHandshakeTimeout: 5 * time.Second,
			},
		},
	}
}

// --- Geocoding Request/Response Structures ---

// GeocodeQuery represents parameters for geocoding requests.
type GeocodeQuery struct {
	Text          string   `url:"text,omitempty"`            // For search and autocomplete
	PointLat      *float64 `url:"point.lat,omitempty"`       // For reverse geocoding
	PointLon      *float64 `url:"point.lon,omitempty"`       // For reverse geocoding
	Size          *int     `url:"size,omitempty"`            // Number of results
	Layers        []string `url:"layers,omitempty,comma"`    // e.g., "address", "venue"
	FocusPointLat *float64 `url:"focus.point.lat,omitempty"` // For proximity-based search
	FocusPointLon *float64 `url:"focus.point.lon,omitempty"` // For proximity-based search
}

// GeoJSONFeatureCollection is the response structure for geocoding APIs.
type GeoJSONFeatureCollection struct {
	Type     string `json:"type"` // "FeatureCollection"
	Features []struct {
		Type     string    `json:"type"` // "Feature"
		Geometry *struct { // Nil for v2 autocomplete
			Type        string    `json:"type"`        // "Point"
			Coordinates []float64 `json:"coordinates"` // [lon, lat]
		} `json:"geometry"`
		Properties map[string]interface{} `json:"properties"` // Address, confidence, gid, etc.
	} `json:"features"`
}

// PlaceDetailResponse is the response for the /place_detail endpoint.
type PlaceDetailResponse struct {
	Type     string `json:"type"` // "Feature"
	Geometry struct {
		Type        string    `json:"type"`        // "Point"
		Coordinates []float64 `json:"coordinates"` // [lon, lat]
	} `json:"geometry"`
	Properties map[string]interface{} `json:"properties"` // Full place details
}

// --- Geocoding API Functions ---

// buildURL constructs the API URL with query parameters.
func (c *Client) buildURL(endpoint string, queryParams interface{}) (string, error) {
	rel, err := url.Parse(endpoint)
	if err != nil {
		return "", errors.Wrap(err, "parse endpoint")
	}
	u := c.BaseURL.ResolveReference(rel)

	q := u.Query()
	q.Set("api_key", c.APIKey)

	if queryParams != nil {
		v, err := query.Values(queryParams)
		if err != nil {
			return "", errors.Wrap(err, "encode query parameters")
		}
		for k, vals := range v {
			for _, val := range vals {
				q.Add(k, val)
			}
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// Search performs forward geocoding using v2 API.
// Endpoint: /geocoding/v2/search
func (c *Client) Search(ctx context.Context, text string, params *GeocodeQuery) (*GeoJSONFeatureCollection, error) {
	if params == nil {
		params = &GeocodeQuery{}
	}
	params.Text = text
	endpoint := "/geocoding/v1/search"

	reqURL, err := c.buildURL(endpoint, params)
	if err != nil {
		return nil, errors.Wrap(err, "build search URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "create search request")
	}

	var result GeoJSONFeatureCollection
	if err := c.do(req, &result); err != nil {
		return nil, errors.Wrap(err, "execute search request")
	}
	return &result, nil
}

// Autocomplete provides address suggestions using v2 API.
// Endpoint: /geocoding/v2/autocomplete
func (c *Client) Autocomplete(ctx context.Context, text string, params *GeocodeQuery) (*GeoJSONFeatureCollection, error) {
	if params == nil {
		params = &GeocodeQuery{}
	}
	params.Text = text
	endpoint := "/geocoding/v2/autocomplete"

	reqURL, err := c.buildURL(endpoint, params)
	if err != nil {
		return nil, errors.Wrap(err, "build autocomplete URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "create autocomplete request")
	}

	var result GeoJSONFeatureCollection
	if err := c.do(req, &result); err != nil {
		return nil, errors.Wrap(err, "execute autocomplete request")
	}
	return &result, nil
}

// ...existing code...
// PlaceDetail fetches detailed place information using v2 API.
// Endpoint: /geocoding/v2/place_detail
func (c *Client) PlaceDetail(ctx context.Context, gid string) (*PlaceDetailResponse, error) {
	endpoint := "/geocoding/v2/place_detail" // Correct endpoint is used here
	queryParams := struct {
		GID string `url:"gid"`
	}{GID: gid}

	reqURL, err := c.buildURL(endpoint, queryParams)
	if err != nil {
		return nil, errors.Wrap(err, "build place detail URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "create place detail request")
	}

	var result PlaceDetailResponse
	if err := c.do(req, &result); err != nil {
		return nil, errors.Wrap(err, "execute place detail request")
	}
	return &result, nil
}

// ...existing code...

// ReverseGeocode performs reverse geocoding using v1 API.
// Endpoint: /geocoding/v1/reverse
func (c *Client) ReverseGeocode(ctx context.Context, lat, lon float64, params *GeocodeQuery) (*GeoJSONFeatureCollection, error) {
	if params == nil {
		params = &GeocodeQuery{}
	}
	params.PointLat = &lat
	params.PointLon = &lon
	endpoint := "/geocoding/v1/reverse"

	reqURL, err := c.buildURL(endpoint, params)
	if err != nil {
		return nil, errors.Wrap(err, "build reverse geocode URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "create reverse geocode request")
	}

	var result GeoJSONFeatureCollection
	if err := c.do(req, &result); err != nil {
		return nil, errors.Wrap(err, "execute reverse geocode request")
	}
	return &result, nil
}

// --- Routing API Structures and Functions ---

// RouteLocation represents a location in a route request.
type RouteLocation struct {
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Type *string `json:"type,omitempty"` // "break", "through", "via"
}

// RouteRequest is the payload for the /route endpoint.
type RouteRequest struct {
	Locations      []RouteLocation `json:"locations"`
	Costing        string          `json:"costing"`                   // e.g., "auto", "pedestrian"
	Units          *string         `json:"units,omitempty"`           // "km" or "mi"
	Language       *string         `json:"language,omitempty"`        // e.g., "en-US"
	DirectionsType *string         `json:"directions_type,omitempty"` // "instructions" or "none"
}

// Maneuver represents a turn-by-turn instruction.
type Maneuver struct {
	Instruction string   `json:"instruction"`
	StreetNames []string `json:"street_names,omitempty"`
	Time        float64  `json:"time"`
	Length      float64  `json:"length"` // In units specified
}

// RouteLeg represents a segment of the trip.
type RouteLeg struct {
	Maneuvers []Maneuver `json:"maneuvers"`
	Summary   struct {
		Time   float64 `json:"time"`
		Length float64 `json:"length"`
	} `json:"summary"`
	Shape string `json:"shape"` // Polyline6 encoded
}

// RouteResponse is the response from the /route endpoint.
type RouteResponse struct {
	Trip RouteTrip `json:"trip"`
}

// RouteTrip represents a single route.
type RouteTrip struct {
	Legs    []RouteLeg `json:"legs"`
	Summary struct {
		Time   float64 `json:"time"`
		Length float64 `json:"length"`
	} `json:"summary"`
	Shape string `json:"shape"` // Polyline6-encoded shape
}

// GetRoute fetches a route using Valhalla.
// Endpoint: /route/v1
func (c *Client) GetRoute(ctx context.Context, routeReq RouteRequest) (*RouteResponse, error) {
	endpoint := "/route/v1"

	bodyBytes, err := json.Marshal(routeReq)
	if err != nil {
		return nil, errors.Wrap(err, "marshal route request")
	}

	reqURL, err := c.buildURL(endpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "build route URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, errors.Wrap(err, "create route request")
	}
	req.Header.Set("Content-Type", "application/json")

	var result RouteResponse
	if err := c.do(req, &result); err != nil {
		return nil, errors.Wrap(err, "execute route request")
	}
	return &result, nil
}

// do executes HTTP requests and decodes JSON responses.
func (c *Client) do(req *http.Request, v interface{}) error {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "execute HTTP request")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return errors.Wrap(err, "decode response")
		}
	}
	return nil
}
