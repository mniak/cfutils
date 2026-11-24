package codefresh

import (
	"net/http"

	cfsdk "github.com/codefresh-io/go-sdk/pkg/codefresh"
	"github.com/samber/lo"
)

const Host = "https://g.codefresh.io"

type Client struct {
	cfsdk.Codefresh
}

func New(host, token string) *Client {
	cf := cfsdk.New(&cfsdk.ClientOptions{
		Host:   lo.CoalesceOrEmpty(host, Host),
		Token:  token,
		Client: injectTokenHTTPClient(token),
	})
	return &Client{cf}
}

func injectTokenHTTPClient(token string) *http.Client {
	return &http.Client{
		Transport: newAccessTokenTransport(token, nil),
	}
}