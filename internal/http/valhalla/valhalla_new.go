package valhalla

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwise1/waze_kibris/util"
)

// --- Mobile-Friendly Formatted Structures ---

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
	TimeSeconds       float64 `json:"timeSeconds"`
	DistanceMeters    float64 `json:"distanceMeters"`
	FormattedTime     string  `json:"formattedTime"`
	FormattedDistance string  `json:"formattedDistance"`
	Units             string  `json:"units"`
}

// MobileManeuver represents a simplified turn-by-turn instruction
type MobileManeuver struct {
	Type           string  `json:"type"` // String representation (e.g., "TurnLeft", "RoundaboutExit")
	Instruction    string  `json:"instruction"`
	DistanceMeters float64 `json:"distanceMeters"` // Distance for this step
	TimeSeconds    float64 `json:"timeSeconds"`    // Time for this step
	// Optional: Add coordinates for the start of the maneuver for easier map interaction
	StartCoordinates []float64 `json:"startCoordinates,omitempty"` // [lon, lat]
	StreetName       string    `json:"streetName,omitempty"`
	// Optional: Include original type if mobile needs it for specific logic
	// OriginalType    int       `json:"originalType,omitempty"`

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

	if targetUnit == "miles" {
		unit = "mi"
		value = meters / 1609.34
		return fmt.Sprintf("%.1f %s", value, unit), unit
	} else {
		// Default to kilometers
		unit = "km"
		value = meters / 1000.0
		// Show meters if less than 1 km
		if value < 1.0 && value > 0 {
			return fmt.Sprintf("%.0f m", meters), "m"
		}
		return fmt.Sprintf("%.1f %s", value, unit), unit
	}
}

// metersPerUnit returns the conversion factor from the Valhalla unit to meters.
func metersPerUnit(unit string) float64 {
	if unit == "miles" {
		return 1609.344
	}
	return 1000.0 // Default to kilometers
}

// formatTripForMobile processes a single Valhalla Trip into a MobileTrip
func formatTripForMobile(trip *Trip) (*MobileTrip, error) {
	if trip == nil {
		return nil, fmt.Errorf("cannot format nil trip")
	}

	mobileTrip := MobileTrip{
		Legs: make([]MobileLeg, 0, len(trip.Legs)),
	}

	// --- Process Summary ---
	metersFactor := metersPerUnit(trip.Units)
	totalDistanceMeters := trip.Summary.Length * metersFactor
	formattedDistStr, distUnit := formatDistance(totalDistanceMeters, trip.Units) // Keep original units for display consistency

	mobileTrip.Summary = MobileTripSummary{
		TotalTimeSeconds:    trip.Summary.Time,
		TotalDistanceMeters: totalDistanceMeters,
		FormattedTime:       formatDuration(trip.Summary.Time),
		FormattedDistance:   formattedDistStr,
		Units:               distUnit,
		BoundingBox:         []float64{trip.Summary.MinLon, trip.Summary.MinLat, trip.Summary.MaxLon, trip.Summary.MaxLat},
	}

	// --- Process Legs ---
	for i, leg := range trip.Legs {
		// Decode Polyline
		// Valhalla uses polyline6, precision 1e6
		coords, err := util.DecodeValhallaPolyline6(leg.Shape)
		if err != nil {
			// Log the error but potentially continue, maybe returning partial results?
			// Or return error immediately:
			return nil, fmt.Errorf("failed to decode polyline for leg %d: %w", i, err)
			// For now, we log and skip the leg, but return an error message later
			// log.Printf("Warning: failed to decode polyline for leg %d: %v", i, err)
			// continue // Skip this leg
		}
		// Convert to [[lon, lat], ...] format expected by many map libs
		mobileCoords := make([][]float64, len(coords))
		for j, p := range coords {
			mobileCoords[j] = []float64{p.Lon, p.Lat} // Access fields directly: polyline gives [lat, lon], maps usually want [lon, lat]
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

		// Process Maneuvers
		for _, maneuver := range leg.Maneuvers {
			maneuverDistMeters := maneuver.Length * metersFactor
			streetName := ""
			if len(maneuver.StreetNames) > 0 {
				streetName = strings.Join(maneuver.StreetNames, " ; ")
			}
			mobileManeuver := MobileManeuver{
				Type: util.MapValhallaManeuverType(maneuver.Type),
				// OriginalType: maneuver.Type // Uncomment if mobile needs the int too
				Instruction:    maneuver.Instruction,
				DistanceMeters: maneuverDistMeters,
				TimeSeconds:    maneuver.Time,
				StreetName:     streetName,
			}
			// Add start coordinates if possible
			if len(mobileCoords) > maneuver.BeginShapeIndex {
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
	// log.Println(resp.Alternates)

	mobileResp := MobileRouteResponse{
		ID:           resp.ID,
		Alternatives: make([]MobileTrip, 0, len(resp.Alternates)),
	}

	// Process the main trip
	mainTrip, err := formatTripForMobile(&resp.Trip)
	if err != nil {
		// Decide how to handle partial errors. Return error immediately?
		// Or return partial result with an error message?
		errMsg := fmt.Sprintf("Error processing main trip: %v", err)
		mobileResp.ErrorMessage = &errMsg
		// return nil, err // Option 1: Fail fast
		// For now, we'll allow returning partials if alternatives work
	} else if mainTrip != nil {
		mobileResp.Trip = *mainTrip
	}

	// Process alternatives
	for i, altTrip := range resp.Alternates {
		// log.Printf("alt trip %d", i)
		// log.Println(altTrip)
		formattedAlt, err := formatTripForMobile(&altTrip.Trip) // Process pointer to avoid copying large struct
		if err != nil {
			log.Println("In the alternatives")
			// Log and potentially add a note to ErrorMessage, but continue
			errMsg := fmt.Sprintf("Error processing alternative %d: %v", i, err)
			if mobileResp.ErrorMessage == nil {
				mobileResp.ErrorMessage = &errMsg
			} else {
				*mobileResp.ErrorMessage += "; " + errMsg
			}
			continue // Skip this alternative
		}
		if formattedAlt != nil {
			mobileResp.Alternatives = append(mobileResp.Alternatives, *formattedAlt)
		}
	}

	// Check if we processed anything useful
	if len(mobileResp.Trip.Legs) == 0 && len(mobileResp.Alternatives) == 0 && mobileResp.ErrorMessage == nil {
		// This means the original response was likely valid but empty, or processing failed silently
		errMsg := "No valid route processed."
		if resp.Trip.StatusMessage != "" {
			errMsg = fmt.Sprintf("No valid route processed. Original status: %s", resp.Trip.StatusMessage)
		}
		mobileResp.ErrorMessage = &errMsg
	}

	// Decide if an overall error should be returned if ErrorMessage is set
	// if mobileResp.ErrorMessage != nil {
	//     return mobileResp, fmt.Errorf(*mobileResp.ErrorMessage)
	// }

	return &mobileResp, nil // Return the potentially partial response
}
