package util

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/twpayne/go-polyline"
)

var (
	RgxEmail         = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	shortCodeCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

func NotBlank(value string) bool {
	return strings.TrimSpace(value) != ""
}

func IsEmail(value string) bool {
	if len(value) > 254 {
		return false
	}

	return RgxEmail.MatchString(value)
}

func IsURL(value string) bool {
	u, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}

	return u.Scheme != "" && u.Host != ""
}

func safeHTML(s string) template.HTML {
	return template.HTML(s)
}

func formatTime(format string, t time.Time) string {
	return t.Format(format)
}

func slugify(s string) string {
	var buf bytes.Buffer

	for _, r := range s {
		switch {
		case r > unicode.MaxASCII:
			continue
		case unicode.IsLetter(r):
			buf.WriteRune(unicode.ToLower(r))
		case unicode.IsDigit(r), r == '_', r == '-':
			buf.WriteRune(r)
		case unicode.IsSpace(r):
			buf.WriteRune('-')
		}
	}

	return buf.String()
}

var TemplateFuncs = template.FuncMap{
	// Time functions
	"now":        time.Now,
	"timeSince":  time.Since,
	"timeUntil":  time.Until,
	"formatTime": formatTime,

	// String functions
	"uppercase": strings.ToUpper,
	"lowercase": strings.ToLower,
	"slugify":   slugify,
	"safeHTML":  safeHTML,

	// Slice functions
	"join": strings.Join,
}

func GenerateVerificationCode() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%04d", rand.Intn(10000))
}

func PointToLatLon(point pgtype.Point) (float64, float64) {
	return point.P.Y, point.P.X
}

// PointFromLatLon creates a pgtype.Point from latitude and longitude.
func PointFromLatLon(lat, lon float64) pgtype.Point {
	return pgtype.Point{
		P: pgtype.Vec2{
			X: lon,
			Y: lat,
		},
	}
}

func DecodePolyLines(shape string) ([][]float64, error) {
	decoded, _, err := polyline.DecodeCoords([]byte(shape))
	if err != nil {
		log.Println("error deocoding polyline: ", err)
		return nil, fmt.Errorf("failed to decode polyline %w", err)
	}
	return decoded, nil
}

func GenerateShortCode(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = shortCodeCharset[rand.Intn(len(shortCodeCharset))]
	}
	return string(b)
}

// Coordinate represents a latitude/longitude pair.
type Coordinate struct {
	Lat float64
	Lon float64
}

// DecodeValhallaPolyline6 decodes an encoded polyline string from Valhalla (precision 1e6)
// into a slice of Coordinate structs. It mirrors the logic commonly used for polyline6.
func DecodeValhallaPolyline6(encoded string) ([]Coordinate, error) {
	var coordinates []Coordinate // Use a slice to store coordinates
	index := 0
	lat, lon := 0, 0 // Use int for accumulated values before dividing

	for index < len(encoded) {
		var latResult, lonResult int
		var shift uint = 0
		var b byte

		// Decode Latitude
		var latChunk int
		for {
			if index >= len(encoded) {
				return nil, fmt.Errorf("polyline decode error: unexpected end of string while decoding latitude")
			}
			b = encoded[index]
			index++

			// Check for invalid characters
			if b < 63 {
				return nil, fmt.Errorf("polyline decode error: invalid character '%c' at index %d", b, index-1)
			}

			// Subtract 63 to get the value
			byteVal := int(b - 63)
			latChunk |= (byteVal & 0x1f) << shift // Mask out the continuation bit and shift
			shift += 5

			// Check the continuation bit (0x20)
			if (byteVal & 0x20) == 0 {
				break // Last chunk for latitude
			}
		}

		// Apply two's complement decoding for negative values
		if (latChunk & 1) != 0 {
			latResult = ^(latChunk >> 1) // Invert bits if negative flag is set
		} else {
			latResult = latChunk >> 1
		}
		lat += latResult // Accumulate latitude change

		// Decode Longitude (similar process)
		shift = 0 // Reset shift for longitude
		var lonChunk int
		for {
			if index >= len(encoded) {
				return nil, fmt.Errorf("polyline decode error: unexpected end of string while decoding longitude")
			}
			b = encoded[index]
			index++

			if b < 63 {
				return nil, fmt.Errorf("polyline decode error: invalid character '%c' at index %d", b, index-1)
			}

			byteVal := int(b - 63)
			lonChunk |= (byteVal & 0x1f) << shift
			shift += 5

			if (byteVal & 0x20) == 0 {
				break // Last chunk for longitude
			}
		}

		// Apply two's complement decoding
		if (lonChunk & 1) != 0 {
			lonResult = ^(lonChunk >> 1)
		} else {
			lonResult = lonChunk >> 1
		}
		lon += lonResult // Accumulate longitude change

		// Append the actual coordinate (divide by 1e6 for Valhalla's precision)
		coordinates = append(coordinates, Coordinate{
			Lat: float64(lat) / 1e6,
			Lon: float64(lon) / 1e6,
		})
	}

	// Check if the loop terminated exactly at the end of the string
	if index != len(encoded) {
		// This case should ideally be caught by the checks inside the loop,
		// but added as a final safeguard.
		return nil, fmt.Errorf("polyline decode error: unexpected characters remaining after decoding")
	}

	return coordinates, nil
}

// Helper function to convert Coordinate slice to [][]float64 format [lon, lat]
// commonly used by map libraries like Mapbox GL JS, MapLibre GL JS.
func CoordinatesToLonLatSlice(coords []Coordinate) [][]float64 {
	lonLatSlice := make([][]float64, len(coords))
	for i, coord := range coords {
		lonLatSlice[i] = []float64{coord.Lon, coord.Lat}
	}
	return lonLatSlice
}

// --- Maneuver Type Mapping ---

// MapValhallaManeuverType converts Valhalla integer type to a string representation.
func MapValhallaManeuverType(typeInt int) string {
	switch typeInt {
	case 0:
		return "None"
	case 1:
		return "Start"
	case 2:
		return "StartRight"
	case 3:
		return "StartLeft"
	case 4:
		return "Destination"
	case 5:
		return "DestinationRight"
	case 6:
		return "DestinationLeft"
	case 7:
		return "Becomes"
	case 8:
		return "Continue"
	case 9:
		return "SlightRight"
	case 10:
		return "Right"
	case 11:
		return "SharpRight"
	case 12:
		return "UturnRight"
	case 13:
		return "UturnLeft"
	case 14:
		return "SharpLeft"
	case 15:
		return "Left"
	case 16:
		return "SlightLeft"
	case 17:
		return "RampStraight"
	case 18:
		return "RampRight"
	case 19:
		return "RampLeft"
	case 20:
		return "ExitRight"
	case 21:
		return "ExitLeft"
	case 22:
		return "StayStraight"
	case 23:
		return "StayRight"
	case 24:
		return "StayLeft"
	case 25:
		return "Merge"
	case 26:
		return "RoundaboutEnter"
	case 27:
		return "RoundaboutExit"
	case 28:
		return "FerryEnter"
	case 29:
		return "FerryExit"
	case 30:
		return "Transit"
	case 31:
		return "TransitTransfer"
	case 32:
		return "TransitRemainOn"
	case 33:
		return "TransitConnectionStart"
	case 34:
		return "TransitConnectionTransfer"
	case 35:
		return "TransitConnectionDestination"
	case 36:
		return "PostTransitConnectionDestination"
	case 37:
		return "MergeRight"
	case 38:
		return "MergeLeft"
	case 39:
		return "ElevatorEnter"
	case 40:
		return "StepsEnter"
	case 41:
		return "EscalatorEnter"
	case 42:
		return "BuildingEnter"
	case 43:
		return "BuildingExit"
	default:
		return fmt.Sprintf("Unknown(%d)", typeInt)
	}
}

// IntPtr returns a pointer to the given integer.
func IntPtr(i int) *int {
	return &i
}
