package util

import (
	"bytes"
	"html/template"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var (
	RgxEmail = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
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
