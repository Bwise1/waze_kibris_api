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
