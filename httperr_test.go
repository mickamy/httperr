package httperr_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"golang.org/x/text/language"

	"github.com/mickamy/httperr"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
)

// localizableError implements httperr.Localizable.
type localizableError struct {
	err      error
	messages map[language.Tag]string
}

func (e *localizableError) Error() string {
	return e.err.Error()
}

func (e *localizableError) Unwrap() error {
	return e.err
}

func (e *localizableError) Localize(tag language.Tag) string {
	if msg, ok := e.messages[tag]; ok {
		return msg
	}
	return e.messages[language.English]
}

func newLocalizableError(err error, messages map[language.Tag]string) *localizableError {
	return &localizableError{err: err, messages: messages}
}

func TestNew(t *testing.T) {
	t.Parallel()

	config := httperr.New("test/error", "Test Error", http.StatusBadRequest)

	if config.Type != "test/error" {
		t.Errorf("Type = %q, want %q", config.Type, "test/error")
	}
	if config.Title != "Test Error" {
		t.Errorf("Title = %q, want %q", config.Title, "Test Error")
	}
	if config.Status != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", config.Status, http.StatusBadRequest)
	}
}

func TestMap_Match(t *testing.T) {
	t.Parallel()

	errMap := httperr.Map{
		ErrNotFound: httperr.New("resource/not-found", "Not Found", http.StatusNotFound),
	}

	t.Run("found", func(t *testing.T) {
		t.Parallel()

		config := errMap.Match(ErrNotFound)
		if config.Status != http.StatusNotFound {
			t.Errorf("Status = %d, want %d", config.Status, http.StatusNotFound)
		}
	})

	t.Run("wrapped error", func(t *testing.T) {
		t.Parallel()

		wrapped := fmt.Errorf("wrapped: %w", ErrNotFound)
		config := errMap.Match(wrapped)
		if config.Status != http.StatusNotFound {
			t.Errorf("Status = %d, want %d", config.Status, http.StatusNotFound)
		}
	})

	t.Run("not found returns default", func(t *testing.T) {
		t.Parallel()

		config := errMap.Match(ErrUnauthorized)
		if config.Status != http.StatusInternalServerError {
			t.Errorf("Status = %d, want %d", config.Status, http.StatusInternalServerError)
		}
	})
}

func TestResponse_ProblemDetail(t *testing.T) {
	t.Parallel()

	t.Run("with detail and instance", func(t *testing.T) {
		t.Parallel()

		resp := httperr.Response{
			Type:   "auth/unauthorized",
			Title:  "Unauthorized",
			Status: http.StatusUnauthorized,
			Detail: "Invalid credentials",
		}

		pd := resp.ProblemDetail("/api/v1/auth/login")

		if pd["type"] != "auth/unauthorized" {
			t.Errorf("type = %v, want %v", pd["type"], "auth/unauthorized")
		}
		if pd["title"] != "Unauthorized" {
			t.Errorf("title = %v, want %v", pd["title"], "Unauthorized")
		}
		if pd["status"] != http.StatusUnauthorized {
			t.Errorf("status = %v, want %v", pd["status"], http.StatusUnauthorized)
		}
		if pd["detail"] != "Invalid credentials" {
			t.Errorf("detail = %v, want %v", pd["detail"], "Invalid credentials")
		}
		if pd["instance"] != "/api/v1/auth/login" {
			t.Errorf("instance = %v, want %v", pd["instance"], "/api/v1/auth/login")
		}
	})

	t.Run("without detail", func(t *testing.T) {
		t.Parallel()

		resp := httperr.Response{
			Type:   "internal-error",
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
		}

		pd := resp.ProblemDetail("")

		if _, ok := pd["detail"]; ok {
			t.Error("detail should not be present when empty")
		}
		if _, ok := pd["instance"]; ok {
			t.Error("instance should not be present when empty")
		}
	})

	t.Run("with baseURI", func(t *testing.T) {
		t.Parallel()

		resp := httperr.Response{
			Type:   "auth/unauthorized",
			Title:  "Unauthorized",
			Status: http.StatusUnauthorized,
		}

		pd := resp.ProblemDetail("/api/v1/auth", "https://api.example.com/problems")

		if pd["type"] != "https://api.example.com/problems/auth/unauthorized" {
			t.Errorf("type = %v, want %v", pd["type"], "https://api.example.com/problems/auth/unauthorized")
		}
	})

	t.Run("with baseURI trailing slash", func(t *testing.T) {
		t.Parallel()

		resp := httperr.Response{
			Type:   "not-found",
			Title:  "Not Found",
			Status: http.StatusNotFound,
		}

		pd := resp.ProblemDetail("", "https://api.example.com/problems/")

		if pd["type"] != "https://api.example.com/problems/not-found" {
			t.Errorf("type = %v, want %v", pd["type"], "https://api.example.com/problems/not-found")
		}
	})

	t.Run("absolute URI ignores baseURI", func(t *testing.T) {
		t.Parallel()

		resp := httperr.Response{
			Type:   "https://other.example.com/problems/conflict",
			Title:  "Conflict",
			Status: http.StatusConflict,
		}

		pd := resp.ProblemDetail("", "https://api.example.com/problems/")

		if pd["type"] != "https://other.example.com/problems/conflict" {
			t.Errorf("type = %v, want %v", pd["type"], "https://other.example.com/problems/conflict")
		}
	})

	t.Run("about:blank ignores baseURI", func(t *testing.T) {
		t.Parallel()

		resp := httperr.Response{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
		}

		pd := resp.ProblemDetail("", "https://api.example.com/problems/")

		if pd["type"] != "about:blank" {
			t.Errorf("type = %v, want %v", pd["type"], "about:blank")
		}
	})
}

func TestResolve(t *testing.T) {
	t.Parallel()

	errMap := httperr.Map{
		ErrNotFound:     httperr.New("resource/not-found", "Not Found", http.StatusNotFound),
		ErrUnauthorized: httperr.New("auth/unauthorized", "Unauthorized", http.StatusUnauthorized),
	}

	t.Run("with localizable error", func(t *testing.T) {
		t.Parallel()

		err := newLocalizableError(ErrNotFound, map[language.Tag]string{
			language.English:  "Resource not found",
			language.Japanese: "リソースが見つかりません",
		})

		resp := httperr.Resolve(err, errMap, language.Japanese)

		if resp.Type != "resource/not-found" {
			t.Errorf("Type = %q, want %q", resp.Type, "resource/not-found")
		}
		if resp.Title != "Not Found" {
			t.Errorf("Title = %q, want %q", resp.Title, "Not Found")
		}
		if resp.Status != http.StatusNotFound {
			t.Errorf("Status = %d, want %d", resp.Status, http.StatusNotFound)
		}
		if resp.Detail != "リソースが見つかりません" {
			t.Errorf("Detail = %q, want %q", resp.Detail, "リソースが見つかりません")
		}
	})

	t.Run("without localizable error", func(t *testing.T) {
		t.Parallel()

		resp := httperr.Resolve(ErrNotFound, errMap, language.English)

		if resp.Detail != "" {
			t.Errorf("Detail = %q, want empty", resp.Detail)
		}
	})

	t.Run("unmapped error returns default", func(t *testing.T) {
		t.Parallel()

		unknownErr := errors.New("unknown error")
		resp := httperr.Resolve(unknownErr, errMap, language.English)

		if resp.Type != "about:blank" {
			t.Errorf("Type = %q, want %q", resp.Type, "about:blank")
		}
		if resp.Status != http.StatusInternalServerError {
			t.Errorf("Status = %d, want %d", resp.Status, http.StatusInternalServerError)
		}

		// about:blank should not be affected by baseURI (RFC 9457 default)
		pd := resp.ProblemDetail("/api", "https://api.example.com/problems")
		if pd["type"] != "about:blank" {
			t.Errorf("type = %v, want %v", pd["type"], "about:blank")
		}
	})

	t.Run("wrapped localizable error", func(t *testing.T) {
		t.Parallel()

		locErr := newLocalizableError(ErrUnauthorized, map[language.Tag]string{
			language.English: "Please log in",
		})
		wrapped := fmt.Errorf("handler: %w", locErr)

		resp := httperr.Resolve(wrapped, errMap, language.English)

		if resp.Status != http.StatusUnauthorized {
			t.Errorf("Status = %d, want %d", resp.Status, http.StatusUnauthorized)
		}
		if resp.Detail != "Please log in" {
			t.Errorf("Detail = %q, want %q", resp.Detail, "Please log in")
		}
	})
}
