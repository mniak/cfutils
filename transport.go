package cfutils

import (
	"net/http"
	"strings"
)

const (
	_HeaderAuthorization = "Authorization"
	_HeaderAccessToken   = "X-Access-Token"
)

type accessTokenTransport struct {
	Token string
	Base  http.RoundTripper
}

func newAccessTokenTransport(token string, base http.RoundTripper) *accessTokenTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &accessTokenTransport{
		Token: token,
		Base:  base,
	}
}

func (t *accessTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())

	token := t.Token
	if token == "" {
		token = bearerToken(req.Header.Get(_HeaderAuthorization))
	}

	req.Header.Del(_HeaderAuthorization)
	if token != "" {
		req.Header.Set(_HeaderAccessToken, token)
	} else {
		req.Header.Del(_HeaderAccessToken)
	}

	return t.Base.RoundTrip(req)
}

func bearerToken(value string) string {
	value = strings.TrimSpace(value)
	scheme, token, found := strings.Cut(value, " ")
	if found && strings.EqualFold(scheme, "Bearer") {
		return strings.TrimSpace(token)
	}
	return value
}