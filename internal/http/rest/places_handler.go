package rest

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	googlemaps "github.com/bwise1/waze_kibris/internal/http/google"
	"github.com/bwise1/waze_kibris/internal/http/mapbox"
	stadiamaps "github.com/bwise1/waze_kibris/internal/http/stadia_maps" // Import stadia_maps
	"github.com/bwise1/waze_kibris/util"
	"github.com/bwise1/waze_kibris/util/tracing"
	"github.com/bwise1/waze_kibris/util/values"
	"github.com/go-chi/chi/v5"
)

// New function for Places/Geocoding routes
func (api *API) PlacesRoutes() chi.Router {
	mux := chi.NewRouter()

	mux.Group(func(r chi.Router) {
		r.Use(api.RequireLogin) // Authentication required for all Places API endpoints

		// Forward Geocoding (Search for an address/place)
		// Query Params: ?text=...&size=...&layers=...&boundary.country=...
		r.Method(http.MethodGet, "/search", Handler(api.SearchPlacesHandler))

		// Reverse Geocoding (Find address for lat/lon)
		// Query Params: ?point.lat=...&point.lon=...&size=...&layers=...
		r.Method(http.MethodGet, "/reverse", Handler(api.ReverseGeocodeHandler))

		// Autocomplete (Get suggestions for partial address/place)
		// Query Params: ?text=...&size=...&focus.point.lat=...&focus.point.lon=... (optional focus)
		r.Method(http.MethodGet, "/autocomplete", Handler(api.AutocompletePlaceHandler))

		// r.Method(http.MethodGet, "/placedetails", Handler(api.PlaceDetailHandler))
		r.Method(http.MethodGet, "/googleplacedetails", Handler(api.GooglePlaceDetailHandler))

		r.Method(http.MethodGet, "/googleautocomplete", Handler(api.GoogleAutocompleteHandler))

		r.Method(http.MethodGet, "/googledirections", Handler(api.GoogleDirectionsHandler))
		r.Method(http.MethodGet, "/mapboxdirections", Handler(api.MapboxDirectionsHandler))
	})
	return mux
}

// --- Places API Handlers ---

func (api *API) SearchPlacesHandler(w http.ResponseWriter, r *http.Request) *ServerResponse {
	tc, ok := r.Context().Value(values.ContextTracingKey).(tracing.Context)
	if !ok {
		return respondWithError(nil, "Missing tracing context", values.SystemErr, nil)
	}

	queryParams := r.URL.Query()
	text := strings.TrimSpace(queryParams.Get("text"))
	if text == "" {
		return respondWithError(nil, "Missing or empty 'text' query parameter", values.BadRequestBody, &tc)
	}

	// Build GeocodeQuery from URL parameters
	geocodeParams := &stadiamaps.GeocodeQuery{Text: text}
	if sizeStr := queryParams.Get("size"); sizeStr != "" {
		size, err := strconv.Atoi(sizeStr)
		if err != nil || size < 1 || size > 100 { // Stadia typically limits to 100
			return respondWithError(err, "Invalid 'size' parameter", values.BadRequestBody, &tc)
		}
		geocodeParams.Size = util.IntPtr(size)
	}
	if layers := queryParams["layers"]; len(layers) > 0 {
		validLayers := map[string]bool{"address": true, "venue": true, "street": true, "locality": true} // Add more as needed
		for _, layer := range layers {
			if !validLayers[layer] {
				return respondWithError(nil, "Invalid 'layers' parameter", values.BadRequestBody, &tc)
			}
		}
		geocodeParams.Layers = layers
	}
	if latStr, lonStr := queryParams.Get("focus.point.lat"), queryParams.Get("focus.point.lon"); latStr != "" && lonStr != "" {
		lat, err1 := strconv.ParseFloat(latStr, 64)
		lon, err2 := strconv.ParseFloat(lonStr, 64)
		if err1 != nil || err2 != nil || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			return respondWithError(nil, "Invalid 'focus.point' coordinates", values.BadRequestBody, &tc)
		}
		geocodeParams.FocusPointLat = &lat
		geocodeParams.FocusPointLon = &lon
	}

	// Perform search using Stadia Maps v2 API
	results, err := api.StadiaClient.Search(r.Context(), text, geocodeParams)
	if err != nil {
		// Check for specific API errors (e.g., rate limits)
		if strings.Contains(err.Error(), "429") {
			return respondWithError(err, "Rate limit exceeded", values.SystemErr, &tc)
		}
		return respondWithError(err, "Failed to search places", values.SystemErr, &tc)
	}

	// Optionally transform results for the frontend
	type simplifiedResult struct {
		Name        string    `json:"name"`
		Address     string    `json:"address"`
		Coordinates []float64 `json:"coordinates"`
		GID         string    `json:"gid"`
	}
	var simplified []simplifiedResult
	for _, feature := range results.Features {
		coords := []float64{}
		if feature.Geometry != nil {
			coords = feature.Geometry.Coordinates
		}
		simplified = append(simplified, simplifiedResult{
			Name:        feature.Properties["name"].(string),
			Address:     feature.Properties["label"].(string),
			Coordinates: coords,
			GID:         feature.Properties["gid"].(string),
		})
	}

	response := &ServerResponse{
		Message:    "Places searched successfully",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       simplified,
	}
	// if err := json.NewEncoder(w).Encode(response); err != nil {
	// 	log.Printf("Error encoding response [%s]: %v", tc.RequestID, err)
	// 	return respondWithError(err, "Failed to encode response", values.SystemErr, &tc)
	// }

	return response
}

func (api *API) ReverseGeocodeHandler(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)
	queryParams := r.URL.Query()

	latStr := queryParams.Get("point.lat")
	lonStr := queryParams.Get("point.lon")

	if latStr == "" || lonStr == "" {
		return respondWithError(nil, "Missing 'point.lat' or 'point.lon' query parameters", values.BadRequestBody, &tc)
	}

	lat, errLat := strconv.ParseFloat(latStr, 64)
	lon, errLon := strconv.ParseFloat(lonStr, 64)

	if errLat != nil || errLon != nil {
		return respondWithError(nil, "Invalid latitude or longitude format", values.BadRequestBody, &tc)
	}

	geocodeParams := &stadiamaps.GeocodeQuery{} // Initialize empty or parse other params
	if sizeStr := queryParams.Get("size"); sizeStr != "" {
		if size, err := strconv.Atoi(sizeStr); err == nil {
			geocodeParams.Size = util.IntPtr(size)
		}
	}
	if layers := queryParams["layers"]; len(layers) > 0 {
		geocodeParams.Layers = layers
	}
	// Add more params as needed

	results, err := api.StadiaClient.ReverseGeocode(r.Context(), lat, lon, geocodeParams)
	if err != nil {
		log.Printf("Error reverse geocoding with Stadia: %v", err)
		return respondWithError(err, "Failed to reverse geocode", values.Error, &tc)
	}

	return &ServerResponse{
		Message:    "Reverse geocoding successful",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       results,
	}
}

func (api *API) AutocompletePlaceHandler(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)
	queryParams := r.URL.Query()
	text := queryParams.Get("text")

	if text == "" {
		return respondWithError(nil, "Missing 'text' query parameter for autocomplete", values.BadRequestBody, &tc)
	}

	geocodeParams := &stadiamaps.GeocodeQuery{Text: text}
	if sizeStr := queryParams.Get("size"); sizeStr != "" {
		if size, err := strconv.Atoi(sizeStr); err == nil {
			geocodeParams.Size = util.IntPtr(size)
		}
	}
	// Stadia Autocomplete can also take focus.point.lat/lon, boundary.rect etc.
	// Example for focus point:
	// if latStr := queryParams.Get("focus.point.lat"); latStr != "" {
	// 	if lat, err := strconv.ParseFloat(latStr, 64); err == nil {
	// 		// You'd need to add FocusPointLat to your stadia_maps.GeocodeQuery struct
	//      // and handle it in stadiaClient.buildURL or pass it appropriately.
	// 	}
	// }
	// Add more params as needed

	results, err := api.StadiaClient.Autocomplete(r.Context(), text, geocodeParams)
	if err != nil {
		log.Printf("Error autocompleting place with Stadia: %v", err)
		return respondWithError(err, "Failed to autocomplete place", values.Error, &tc)
	}

	return &ServerResponse{
		Message:    "Autocomplete successful",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       results,
	}
}

// ... (other existing code in places_handler.go)

func (api *API) PlaceDetailHandler(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc, ok := r.Context().Value(values.ContextTracingKey).(tracing.Context)
	if !ok {
		log.Println("Warning: Missing tracing context in PlaceDetailHandler")
		// Consider returning a default error or handling as appropriate
	}
	queryParams := r.URL.Query()
	gid := strings.TrimSpace(queryParams.Get("gid"))

	if gid == "" {
		return respondWithError(nil, "Missing 'gid' query parameter", values.BadRequestBody, &tc)
	}

	// The PlaceDetail method in stadia.go now takes only the GID
	// and returns *stadiamaps.PlaceDetailResponse
	placeData, err := api.StadiaClient.PlaceDetail(r.Context(), gid)
	if err != nil {
		log.Printf("Error fetching place details from Stadia (v2) for GID %s: %v", gid, err)
		// Check for specific API errors (e.g., rate limits, not found)
		// The error message from c.do in stadia.go will include the status code.
		if strings.Contains(err.Error(), "status 404") { // Example check for 404
			return respondWithError(err, "Place details not found for the given GID", values.NotFound, &tc)
		}
		if strings.Contains(err.Error(), "status 429") { // Example check for 429
			return respondWithError(err, "Rate limit exceeded", values.SystemErr, &tc)
		}
		return respondWithError(err, "Failed to fetch place details", values.SystemErr, &tc)
	}

	// The placeData is now directly *stadiamaps.PlaceDetailResponse,
	// which represents a single GeoJSON Feature.
	if placeData == nil { // Should not happen if err is nil, but good for robustness
		log.Printf("No data returned for GID %s from PlaceDetail (v2), though API call succeeded.", gid)
		return respondWithError(nil, "Place details not found for the given GID (no data)", values.NotFound, &tc)
	}

	return &ServerResponse{
		Message:    "Place details retrieved successfully",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       placeData, // Return the PlaceDetailResponse directly
	}
}

func (api *API) GooglePlaceDetailHandler(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc, ok := r.Context().Value(values.ContextTracingKey).(tracing.Context)
	if !ok {
		log.Println("Warning: Missing tracing context in GooglePlaceDetailHandler")
	}
	queryParams := r.URL.Query()
	placeID := strings.TrimSpace(queryParams.Get("place_id"))

	if placeID == "" {
		return respondWithError(nil, "Missing 'place_id' query parameter", values.BadRequestBody, &tc)
	}

	// Specify the fields you want from Google (excluding atmosphere data)
	fields := []string{
		"name", "formatted_address", "geometry", "rating", "opening_hours", "photos", "reviews", "place_id",
	}

	placeData, err := api.GoogleMapsClient.GetPlaceDetails(r.Context(), placeID, fields)
	if err != nil {
		log.Printf("Error fetching place details from Google for PlaceID %s: %v", placeID, err)
		return respondWithError(err, "Failed to fetch place details", values.SystemErr, &tc)
	}

	if placeData == nil {
		log.Printf("No data returned for PlaceID %s from Google Place Details.", placeID)
		return respondWithError(nil, "No place details found", values.NotFound, &tc)
	}

	return &ServerResponse{
		Message:    "Place details fetched successfully (Google)",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       placeData,
	}
}

// GoogleAutocompleteHandler provides autocomplete suggestions using Google Places API.
// Query Params: ?text=...&focus.point.lat=...&focus.point.lon=...&radius=...
func (api *API) GoogleAutocompleteHandler(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)
	queryParams := r.URL.Query()
	text := queryParams.Get("text")

	if text == "" {
		return respondWithError(nil, "Missing 'text' query parameter", values.BadRequestBody, &tc)
	}

	// --- MODIFIED SECTION START ---

	// Parse origin coordinates from "lat" and "lon" query parameters.
	var origin *googlemaps.LatLng
	latStr := queryParams.Get("lat")
	lonStr := queryParams.Get("lon")

	if latStr != "" && lonStr != "" {
		lat, err1 := strconv.ParseFloat(latStr, 64)
		lon, err2 := strconv.ParseFloat(lonStr, 64)
		if err1 == nil && err2 == nil {
			origin = &googlemaps.LatLng{Lat: lat, Lng: lon}
		} else {
			// Optional: return an error for invalid coordinates
			log.Printf("Invalid latitude/longitude format: lat=%s, lon=%s", latStr, lonStr)
			return respondWithError(nil, "Invalid 'lat' or 'lon' query parameter format", values.BadRequestBody, &tc)
		}
	}

	// --- MODIFIED SECTION END ---

	// Optional: parse radius
	radius := 0
	if radiusStr := queryParams.Get("radius"); radiusStr != "" {
		if r, err := strconv.Atoi(radiusStr); err == nil {
			radius = r
		}
	}

	// Pass the parsed 'origin' to your client function.
	results, err := api.GoogleMapsClient.PlaceAutocomplete(r.Context(), text, origin, radius)
	if err != nil {
		log.Printf("Error autocompleting place with Google: %v", err)
		return respondWithError(err, "Failed to autocomplete place (Google)", values.Error, &tc)
	}

	return &ServerResponse{
		Message:    "Autocomplete successful (Google)",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       results,
	}
}
func (api *API) GoogleDirectionsHandler(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)
	q := r.URL.Query()
	origin := q.Get("origin")
	destination := q.Get("destination")
	mode := q.Get("mode")
	waypoints := q["waypoint"] // e.g. ?waypoint=Benin&waypoint=Ibadan

	if origin == "" || destination == "" {
		return respondWithError(nil, "Missing 'origin' or 'destination'", values.BadRequestBody, &tc)
	}

	result, err := api.GoogleMapsClient.Directions(r.Context(), origin, destination, waypoints, mode, true)
	if err != nil {
		return respondWithError(err, "Failed to get directions", values.SystemErr, &tc)
	}
	return &ServerResponse{
		Message:    "Directions fetched successfully",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       result,
	}
}

// MapboxDirectionsHandler provides road-snapped directions using Mapbox Directions API
// This gives PROFESSIONAL ROAD-ALIGNED polylines for navigation
func (api *API) MapboxDirectionsHandler(_ http.ResponseWriter, r *http.Request) *ServerResponse {
	tc := r.Context().Value(values.ContextTracingKey).(tracing.Context)
	q := r.URL.Query()
	origin := q.Get("origin")           // e.g., "35.1856,33.3823"
	destination := q.Get("destination") // e.g., "35.1951,33.3662"
	profile := q.Get("profile")         // "driving", "walking", "cycling", "driving-traffic"
	waypoints := q["waypoint"]          // Optional waypoints

	if origin == "" || destination == "" {
		return respondWithError(nil, "Missing 'origin' or 'destination'", values.BadRequestBody, &tc)
	}

	// Build coordinates array for Mapbox (format: lng,lat)
	coordinates := []string{
		mapbox.FormatCoordinate(origin), // Convert lat,lng to lng,lat
	}
	
	// Add waypoints if provided
	for _, wp := range waypoints {
		coordinates = append(coordinates, mapbox.FormatCoordinate(wp))
	}
	
	// Add destination
	coordinates = append(coordinates, mapbox.FormatCoordinate(destination))

	// Get road-snapped directions from Mapbox
	result, err := api.MapboxClient.Directions(r.Context(), coordinates, profile, true, true, "geojson")
	if err != nil {
		log.Printf("Error getting Mapbox directions: %v", err)
		return respondWithError(err, "Failed to get Mapbox directions", values.SystemErr, &tc)
	}

	if len(result.Routes) == 0 {
		return respondWithError(nil, "No routes found", values.NotFound, &tc)
	}

	return &ServerResponse{
		Message:    "Mapbox directions fetched successfully with road-snapped coordinates",
		Status:     values.Success,
		StatusCode: util.StatusCode(values.Success),
		Data:       result,
	}
}
