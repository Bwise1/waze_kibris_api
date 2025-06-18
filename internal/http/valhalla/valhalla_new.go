package valhalla

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwise1/waze_kibris/util" // Assuming this provides DecodeValhallaPolyline6 and MapValhallaManeuverType
)

// --- Assume these Valhalla library structures (or similar) ---
// These are not explicitly defined in your snippet but are inferred for the logic.
// You'd typically get these from your Valhalla client library.

// LocationInfo represents a waypoint in the Valhalla trip.
type LocationInfo struct {
	Type          string  `json:"type"` // e.g., "break", "through", "start", "end"
	Lat           float64 `json:"lat"`
	Lon           float64 `json:"lon"`
	Name          string  `json:"name,omitempty"`   // User-defined name or derived name
	Street        string  `json:"street,omitempty"` // Street name at the location
	OriginalIndex int     `json:"original_index"`   // Index from the input request's locations array
	// ... other fields like SideOfStreet, etc.
}

// TripSummary is part of Valhalla's Trip.
type TripSummary struct {
	Time   float64 `json:"time"`
	Length float64 `json:"length"` // In units specified by Trip.Units
	MinLat float64 `json:"min_lat"`
	MinLon float64 `json:"min_lon"`
	MaxLat float64 `json:"max_lat"`
	MaxLon float64 `json:"max_lon"`
	// ... other fields
}

// LegSummary is part of Valhalla's Leg.
type LegSummary struct {
	Time   float64 `json:"time"`
	Length float64 `json:"length"` // In units specified by Trip.Units
	// ... other fields
}

// Maneuver is part of Valhalla's Leg.
type Maneuver struct {
	Type            int      `json:"type"`
	Instruction     string   `json:"instruction"`
	Time            float64  `json:"time"`
	Length          float64  `json:"length"` // In units specified by Trip.Units
	BeginShapeIndex int      `json:"begin_shape_index"`
	StreetNames     []string `json:"street_names,omitempty"`
	// ... other fields
}

// Leg is part of Valhalla's Trip.
type Leg struct { // Or TripLeg
	Summary   LegSummary `json:"summary"`
	Shape     string     `json:"shape"` // Encoded polyline
	Maneuvers []Maneuver `json:"maneuvers"`
	// ... other fields
}

// Trip is Valhalla's representation of a route or part of a route.
type Trip struct {
	Locations     []LocationInfo `json:"locations"` // Crucial for identifying via points
	Legs          []Leg          `json:"legs"`
	Summary       TripSummary    `json:"summary"`
	Units         string         `json:"units"`          // e.g., "kilometers" or "miles"
	Status        int            `json:"status"`         // Optional: Valhalla status code
	StatusMessage string         `json:"status_message"` // Optional: Valhalla status message
	// ... other fields
}

// AlternateRoute typically wraps a Trip for alternative routes.
type AlternateRoute struct {
	Trip Trip `json:"trip"`
	// ... potentially other metadata about the alternate
}

// RouteResponse is the raw response from Valhalla.
type RouteResponse struct {
	ID         *string          `json:"id,omitempty"`
	Trip       Trip             `json:"trip"`
	Alternates []AlternateRoute `json:"alternates,omitempty"` // Assuming this structure for alternates
	// ... other fields like error codes, etc.
}

// --- Mobile-Friendly Formatted Structures (Your existing structs with modifications) ---

// MobileRouteResponse is the top-level response optimized for mobile consumption
type MobileRouteResponse struct {
	ID           *string      `json:"id,omitempty"` // Optional: Echoes request ID
	Trip         MobileTrip   `json:"trip"`
	Alternatives []MobileTrip `json:"alternates,omitempty"`
	ErrorMessage *string      `json:"errorMessage,omitempty"` // Used if processing fails partially/fully
}

// MobileTrip represents a single processed route trip
type MobileTrip struct {
	Summary MobileTripSummary `json:"summary"`
	Legs    []MobileLeg       `json:"legs"`
}

// MobileTripSummary provides formatted overall trip details
type MobileTripSummary struct {
	TotalTimeSeconds    float64   `json:"totalTimeSeconds"`
	TotalDistanceMeters float64   `json:"totalDistanceMeters"`
	FormattedTime       string    `json:"formattedTime"`         // e.g., "1h 15m"
	FormattedDistance   string    `json:"formattedDistance"`     // e.g., "120.5 km" or "75.0 mi" (depends on desired output unit)
	Units               string    `json:"units"`                 // Indicate units used in FormattedDistance ("km" or "mi")
	BoundingBox         []float64 `json:"boundingBox,omitempty"` // Optional: [minLon, minLat, maxLon, maxLat]
}

// MobileLeg represents a processed leg of the trip
type MobileLeg struct {
	Summary     MobileLegSummary `json:"summary"`
	Coordinates [][]float64      `json:"coordinates"` // Decoded polyline as [[lon, lat], ...]
	Maneuvers   []MobileManeuver `json:"maneuvers"`
}

// MobileLegSummary provides formatted leg details
type MobileLegSummary struct {
	TimeSeconds             float64 `json:"timeSeconds"`
	DistanceMeters          float64 `json:"distanceMeters"`
	FormattedTime           string  `json:"formattedTime"`
	FormattedDistance       string  `json:"formattedDistance"`
	Units                   string  `json:"units"`
	DestinationWaypointType *string `json:"destinationWaypointType,omitempty"` // ADDED: e.g., "Stopover", "ViaPassThrough", "FinalDestination"
	DestinationWaypointName *string `json:"destinationWaypointName,omitempty"` // ADDED: Name of the destination waypoint for this leg
}

// MobileManeuver represents a simplified turn-by-turn instruction
type MobileManeuver struct {
	Type             string    `json:"type"` // String representation (e.g., "TurnLeft", "RoundaboutExit")
	Instruction      string    `json:"instruction"`
	DistanceMeters   float64   `json:"distanceMeters"`             // Distance for this step
	TimeSeconds      float64   `json:"timeSeconds"`                // Time for this step
	StartCoordinates []float64 `json:"startCoordinates,omitempty"` // [lon, lat]
	StreetName       string    `json:"streetName,omitempty"`
}

// --- Formatting Helper Functions ---

// formatDuration converts seconds into a "Xh Ym" or "Ym Zs" string
func formatDuration(seconds float64) string {
	if seconds < 0 {
		return "0s"
	}
	dur := time.Duration(seconds * float64(time.Second))
	h := int(dur.Hours())
	m := int(dur.Minutes()) % 60
	s := int(dur.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// formatDistance converts meters into a "X.Y km" or "X.Y mi" string
func formatDistance(meters float64, targetUnit string) (string, string) {
	unit := "km" // Default unit
	var value float64

	if targetUnit == "miles" { // Valhalla might use "miles" or "mi" in trip.Units
		unit = "mi"
		value = meters / 1609.34
		return fmt.Sprintf("%.1f %s", value, unit), unit
	} else { // Default to kilometers, Valhalla might use "kilometers" or "km"
		unit = "km"
		value = meters / 1000.0
		if value < 1.0 && value > 0 { // Show meters if less than 1 km and not zero
			return fmt.Sprintf("%.0f m", meters), "m"
		}
		return fmt.Sprintf("%.1f %s", value, unit), unit
	}
}

// metersPerUnit returns the conversion factor from the Valhalla unit to meters.
func metersPerUnit(unit string) float64 {
	// Valhalla typically uses "kilometers" or "miles"
	if strings.ToLower(unit) == "miles" || strings.ToLower(unit) == "mi" {
		return 1609.344
	}
	return 1000.0 // Default to kilometers
}

// formatTripForMobile processes a single Valhalla Trip into a MobileTrip
func formatTripForMobile(trip *Trip) (*MobileTrip, error) {
	if trip == nil {
		return nil, fmt.Errorf("cannot format nil trip")
	}
	if trip.Locations == nil {
		// This check is important as trip.Locations is used to determine via points.
		// If it's nil, we might not be able to correctly identify via points.
		// Depending on requirements, you might return an error or proceed with limited info.
		log.Println("Warning: trip.Locations is nil, cannot determine via point details accurately.")
		// return nil, fmt.Errorf("trip.Locations is nil, cannot process via points")
	}

	mobileTrip := MobileTrip{
		Legs: make([]MobileLeg, 0, len(trip.Legs)),
	}

	metersFactor := metersPerUnit(trip.Units)
	totalDistanceMeters := trip.Summary.Length * metersFactor
	formattedDistStr, distUnit := formatDistance(totalDistanceMeters, trip.Units)

	mobileTrip.Summary = MobileTripSummary{
		TotalTimeSeconds:    trip.Summary.Time,
		TotalDistanceMeters: totalDistanceMeters,
		FormattedTime:       formatDuration(trip.Summary.Time),
		FormattedDistance:   formattedDistStr,
		Units:               distUnit,
	}
	// Bounding box might be nil if summary doesn't provide it or if trip is minimal
	if trip.Summary.MinLon != 0 || trip.Summary.MinLat != 0 || trip.Summary.MaxLon != 0 || trip.Summary.MaxLat != 0 {
		mobileTrip.Summary.BoundingBox = []float64{trip.Summary.MinLon, trip.Summary.MinLat, trip.Summary.MaxLon, trip.Summary.MaxLat}
	}

	// --- Process Legs ---
	for legIdx, leg := range trip.Legs {
		coords, err := util.DecodeValhallaPolyline6(leg.Shape)
		if err != nil {
			return nil, fmt.Errorf("failed to decode polyline for leg %d: %w", legIdx, err)
		}
		mobileCoords := make([][]float64, len(coords))
		for j, p := range coords {
			mobileCoords[j] = []float64{p.Lon, p.Lat}
		}

		mobileLeg := MobileLeg{
			Coordinates: mobileCoords,
			Maneuvers:   make([]MobileManeuver, 0, len(leg.Maneuvers)),
		}

		// Process Leg Summary
		legDistMeters := leg.Summary.Length * metersFactor
		legFormattedDist, legDistUnit := formatDistance(legDistMeters, trip.Units)
		mobileLeg.Summary = MobileLegSummary{
			TimeSeconds:       leg.Summary.Time,
			DistanceMeters:    legDistMeters,
			FormattedTime:     formatDuration(leg.Summary.Time),
			FormattedDistance: legFormattedDist,
			Units:             legDistUnit,
		}

		// --- ADDED LOGIC for Destination Waypoint Type and Name ---
		// A leg `trip.Legs[legIdx]` goes from `trip.Locations[legIdx]` to `trip.Locations[legIdx+1]`.
		// So, `trip.Locations[legIdx+1]` is the destination waypoint for this current leg.
		if trip.Locations != nil && len(trip.Locations) > legIdx+1 {
			destWaypointInfo := trip.Locations[legIdx+1]
			var destWaypointType string

			// Check if this is the final destination of the entire trip
			if legIdx+1 == len(trip.Locations)-1 {
				destWaypointType = "FinalDestination"
			} else {
				// Otherwise, it's an intermediate waypoint (via, stopover)
				switch strings.ToLower(destWaypointInfo.Type) {
				case "break": // "break" locations are typically user-specified stops/waypoints.
					destWaypointType = "Stopover"
				case "through": // "through" locations are points the route must pass through.
					destWaypointType = "ViaPassThrough"
				// Add other Valhalla location types if needed for more specific categorization
				// e.g., "break_through" might also be "ViaPassThrough" or "Stopover"
				default:
					// If type is not 'break' or 'through', it might be an implicit point.
					// We only explicitly mark user-defined intermediate points here.
					// log.Printf("Leg %d destination waypoint type: %s (not marked as via/stopover)", legIdx, destWaypointInfo.Type)
				}
			}

			if destWaypointType != "" {
				mobileLeg.Summary.DestinationWaypointType = &destWaypointType
			}

			// Set destination waypoint name
			if destWaypointInfo.Name != "" {
				mobileLeg.Summary.DestinationWaypointName = &destWaypointInfo.Name
			} else if destWaypointInfo.Street != "" { // Fallback to street name
				mobileLeg.Summary.DestinationWaypointName = &destWaypointInfo.Street
			}
		} else if trip.Locations == nil {
			log.Printf("Warning: trip.Locations is nil, cannot determine destination waypoint type/name for leg %d", legIdx)
		} else {
			log.Printf("Warning: Not enough location info to determine destination waypoint for leg %d (locations: %d, legIdx+1: %d)", legIdx, len(trip.Locations), legIdx+1)
		}
		// --- END ADDED LOGIC ---

		// Process Maneuvers
		for _, maneuver := range leg.Maneuvers {
			maneuverDistMeters := maneuver.Length * metersFactor
			streetName := ""
			if len(maneuver.StreetNames) > 0 {
				streetName = strings.Join(maneuver.StreetNames, " ; ")
			}
			mobileManeuver := MobileManeuver{
				Type:           util.MapValhallaManeuverType(maneuver.Type),
				Instruction:    maneuver.Instruction,
				DistanceMeters: maneuverDistMeters,
				TimeSeconds:    maneuver.Time,
				StreetName:     streetName,
			}
			if len(mobileCoords) > maneuver.BeginShapeIndex && maneuver.BeginShapeIndex >= 0 {
				mobileManeuver.StartCoordinates = mobileCoords[maneuver.BeginShapeIndex]
			}

			mobileLeg.Maneuvers = append(mobileLeg.Maneuvers, mobileManeuver)
		}
		mobileTrip.Legs = append(mobileTrip.Legs, mobileLeg)
	}

	return &mobileTrip, nil
}

// FormatRouteForMobile takes a raw Valhalla response and converts it to mobile-friendly format
func FormatRouteForMobile(resp *RouteResponse) (*MobileRouteResponse, error) {
	if resp == nil {
		return nil, fmt.Errorf("received nil RouteResponse")
	}

	mobileResp := MobileRouteResponse{
		ID:           resp.ID,
		Alternatives: make([]MobileTrip, 0, len(resp.Alternates)),
	}

	// Process the main trip
	// Ensure resp.Trip is not nil before dereferencing, though formatTripForMobile handles nil trip.
	if resp.Trip.Legs != nil || resp.Trip.Summary.Time > 0 { // Basic check if trip has some data
		mainTrip, err := formatTripForMobile(&resp.Trip)
		if err != nil {
			errMsg := fmt.Sprintf("Error processing main trip: %v", err)
			mobileResp.ErrorMessage = &errMsg
		} else if mainTrip != nil {
			mobileResp.Trip = *mainTrip
		}
	} else {
		// Handle case where resp.Trip might be an empty struct
		log.Println("Main trip in RouteResponse appears to be empty or uninitialized.")
	}

	// Process alternatives
	for i, altRoute := range resp.Alternates { // Assuming resp.Alternates is []AlternateRoute
		// altRoute.Trip is the actual Trip object for the alternative
		if altRoute.Trip.Legs != nil || altRoute.Trip.Summary.Time > 0 { // Basic check
			formattedAlt, err := formatTripForMobile(&altRoute.Trip)
			if err != nil {
				log.Printf("Error processing alternative %d: %v", i, err)
				errMsgPart := fmt.Sprintf("Error processing alternative %d: %v", i, err)
				if mobileResp.ErrorMessage == nil {
					mobileResp.ErrorMessage = &errMsgPart
				} else {
					*mobileResp.ErrorMessage += "; " + errMsgPart
				}
				continue
			}
			if formattedAlt != nil {
				mobileResp.Alternatives = append(mobileResp.Alternatives, *formattedAlt)
			}
		} else {
			log.Printf("Alternative trip %d in RouteResponse appears to be empty or uninitialized.", i)
		}
	}

	// Check if we processed anything useful, especially if main trip was initially empty
	if (mobileResp.Trip.Legs == nil || len(mobileResp.Trip.Legs) == 0) && len(mobileResp.Alternatives) == 0 && mobileResp.ErrorMessage == nil {
		errMsg := "No valid route processed."
		// Check original Valhalla status message if available and trip was somewhat initialized
		if resp.Trip.StatusMessage != "" {
			errMsg = fmt.Sprintf("No valid route processed. Original status: %s", resp.Trip.StatusMessage)
		} else if resp.Trip.Legs == nil && resp.Trip.Locations == nil {
			errMsg = "No route data found in the Valhalla response."
		}
		mobileResp.ErrorMessage = &errMsg
	}

	return &mobileResp, nil
}
