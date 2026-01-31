// Package httperr provides RFC 9457 (Problem Details) compliant HTTP error responses.
package httperr

import (
	"errors"
	"net/http"
	"strings"

	"golang.org/x/text/language"
)

// ContentType is the MIME type for RFC 9457 Problem Details.
const ContentType = "application/problem+json"

// Config is the HTTP response configuration for an error.
type Config struct {
	Type   string // RFC 9457 type URI
	Title  string // Short description (fixed, English)
	Status int    // HTTP status code
}

// New creates a Config.
func New(typ, title string, status int) Config {
	return Config{
		Type:   typ,
		Title:  title,
		Status: status,
	}
}

// Map is a mapping from error to Config.
type Map map[error]Config

// Match finds a Config that matches the error using errors.Is.
// Returns default (500 Internal Server Error) if not found.
func (m Map) Match(err error) Config {
	for target, config := range m {
		if errors.Is(err, target) {
			return config
		}
	}
	return defaultConfig
}

var defaultConfig = Config{
	Type:   "about:blank",
	Title:  "Internal Server Error",
	Status: http.StatusInternalServerError,
}

// Response is the resolved response information.
type Response struct {
	Type   string // RFC 9457 type
	Title  string // Short description
	Status int    // HTTP status code
	Detail string // Localized message (from Localizable)
}

// ProblemDetail returns RFC 9457 compliant map.
// If baseURI is provided and Type is not an absolute URI, Type is resolved against baseURI.
func (r *Response) ProblemDetail(instance string, baseURI ...string) map[string]any {
	typeURI := r.Type
	if len(baseURI) > 0 && baseURI[0] != "" && !isAbsoluteURI(r.Type) {
		typeURI = resolveURI(baseURI[0], r.Type)
	}

	pd := map[string]any{
		"type":   typeURI,
		"title":  r.Title,
		"status": r.Status,
	}
	if r.Detail != "" {
		pd["detail"] = r.Detail
	}
	if instance != "" {
		pd["instance"] = instance
	}
	return pd
}

func isAbsoluteURI(uri string) bool {
	return strings.HasPrefix(uri, "http://") ||
		strings.HasPrefix(uri, "https://") ||
		strings.HasPrefix(uri, "about:")
}

func resolveURI(base, ref string) string {
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return base + ref
}

// Localizable is an error that can be localized.
type Localizable interface {
	Localize(tag language.Tag) string
}

// Resolve resolves an error to response information.
//   - Finds Config from Map using errors.Is
//   - Finds Localizable using errors.As and localizes Detail
func Resolve(err error, m Map, tag language.Tag) Response {
	config := m.Match(err)

	detail := ""
	var loc Localizable
	if errors.As(err, &loc) {
		detail = loc.Localize(tag)
	}

	return Response{
		Type:   config.Type,
		Title:  config.Title,
		Status: config.Status,
		Detail: detail,
	}
}
