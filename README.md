# httperr

RFC 9457 (Problem Details for HTTP APIs) compliant error response library for Go.

## Installation

```bash
go get github.com/mickamy/httperr
```

## Usage

### Define error mapping

```go
import "github.com/mickamy/httperr"

// Use case layer errors (HTTP-agnostic)
var (
    ErrUserNotFound = errors.New("user not found")
    ErrUnauthorized = errors.New("unauthorized")
)

// Handler layer mapping
var errMap = httperr.Map{
    ErrUserNotFound: httperr.New(
        "users/not-found",       // type (RFC 9457)
        "User not found",        // title
        http.StatusNotFound,     // status
    ),
    ErrUnauthorized: httperr.New(
        "auth/unauthorized",
        "Unauthorized",
        http.StatusUnauthorized,
    ),
}
```

### Resolve errors

```go
func (h *Handler) GetUser(c echo.Context) error {
    ctx := c.Request().Context()
    tag := language.Japanese // from Accept-Language

    user, err := h.uc.GetUser(ctx, id)
    if err != nil {
        resp := httperr.Resolve(err, errMap, tag)

        // Log level based on status
        if resp.Status >= 500 {
            logger.Error("failed", "err", err)
        } else {
            logger.Info("failed", "err", err)
        }

        return c.JSON(resp.Status, resp.ProblemDetail(c.Request().URL.Path))
    }

    return c.JSON(http.StatusOK, user)
}
```

### Response format

```json
{
  "type": "users/not-found",
  "title": "User not found",
  "status": 404,
  "detail": "ユーザーが見つかりません",
  "instance": "/api/v1/users/123"
}
```

Content-Type: `application/problem+json`

### With base URI (RFC 9457 recommends absolute URIs)

```go
resp.ProblemDetail("/api/v1/users/123", "https://api.example.com/problems")
```

```json
{
  "type": "https://api.example.com/problems/users/not-found",
  "title": "User not found",
  "status": 404
}
```

### Localization

Implement the `Localizable` interface:

```go
type LocalizableError struct {
    err      error
    messages map[language.Tag]string
}

func (e *LocalizableError) Error() string   { return e.err.Error() }
func (e *LocalizableError) Unwrap() error   { return e.err }
func (e *LocalizableError) Localize(tag language.Tag) string {
    if msg, ok := e.messages[tag]; ok {
        return msg
    }
    return e.messages[language.English]
}
```

`Resolve` automatically extracts localized messages via `errors.As`.

## Unmapped errors

Errors not in the map return RFC 9457 default:

```json
{
  "type": "about:blank",
  "title": "Internal Server Error",
  "status": 500
}
```

## Features

- **RFC 9457 compliant** - Problem Details for HTTP APIs
- **Layered architecture friendly** - Use case layer stays HTTP-agnostic
- **Localization support** - Via `Localizable` interface and `golang.org/x/text/language`
- **Framework agnostic** - Works with echo, gin, net/http, etc.
- **Logging is caller's responsibility** - Check `resp.Status` to decide log level

## References

- [RFC 9457: Problem Details for HTTP APIs](https://www.rfc-editor.org/rfc/rfc9457.html)

## License

[MIT](./LICENSE)
