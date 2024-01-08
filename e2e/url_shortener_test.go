package e2e

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/dkhrunov/url-shortener/internal/config"
	"github.com/dkhrunov/url-shortener/internal/lib/random"
	"github.com/dkhrunov/url-shortener/internal/transport/http/common/response"
	"github.com/dkhrunov/url-shortener/internal/transport/http/handlers/url/save"
	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	cfg *config.Config
)

func init() {
	os.Setenv("CONFIG_PATH", "../config/local.yaml")
	cfg = config.MustLoad()

}

func TestURLShortener_SaveHappyPath(t *testing.T) {
	u := url.URL{
		Scheme: "http",
		Host:   cfg.HTTPServer.Address,
	}
	c := httpexpect.Default(t, u.String())

	c.POST("/url").
		WithJSON(save.Request{
			URL:   gofakeit.URL(),
			Alias: random.RandomString(10),
		}).
		WithBasicAuth(cfg.User, cfg.Password).
		Expect().
		Status(http.StatusOK).
		JSON().
		Object().
		ContainsKey("alias")
}

func TestURLShortener_SaveRedirectDelete(t *testing.T) {

	testCases := []struct {
		name   string
		url    string
		alias  string
		error  string
		status int
	}{
		{
			name:   "Valid URL",
			url:    gofakeit.URL(),
			alias:  gofakeit.Word() + gofakeit.Word(),
			status: http.StatusOK,
		},
		{
			name:   "Invalid URL",
			url:    "invalid_url",
			alias:  gofakeit.Word(),
			error:  "field URL is not a valid URL",
			status: http.StatusBadRequest,
		},
		{
			name:   "Empty Alias",
			url:    gofakeit.URL(),
			alias:  "",
			status: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			u := url.URL{
				Scheme: "http",
				Host:   cfg.HTTPServer.Address,
			}

			c := httpexpect.Default(t, u.String())

			// Save
			res := c.POST("/url").
				WithJSON(save.Request{
					URL:   tc.url,
					Alias: tc.alias,
				}).
				WithBasicAuth(cfg.User, cfg.Password).
				Expect().
				Status(tc.status).
				JSON().
				Object()

			if tc.error != "" {
				res.NotContainsKey("alias")
				res.Value("error").String().IsEqual(tc.error)
				return
			}

			alias := tc.alias

			if tc.alias != "" {
				res.Value("alias").String().IsEqual(tc.alias)
			} else {
				res.Value("alias").String().NotEmpty()

				alias = res.Value("alias").String().Raw()
			}

			// Redirect
			testRedirect(t, alias, tc.url)

			// Delete
			delRes := c.DELETE("/"+path.Join("url", alias)).
				WithBasicAuth(cfg.User, cfg.Password).
				Expect().
				Status(http.StatusOK).
				JSON().
				Object()

			delRes.Value("status").String().IsEqual("OK")

			// Redirect again
			testRedirectNotFound(t, alias)
		})
	}
}

func testRedirect(t *testing.T, alias string, urlToRedirect string) {
	u := url.URL{
		Scheme: "http",
		Host:   cfg.HTTPServer.Address,
		Path:   alias,
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // stop after 1st redirect
		},
	}

	res, err := client.Get(u.String())
	require.NoError(t, err)
	defer res.Body.Close()

	redirectedToURL := res.Header.Get("Location")

	assert.Equal(t, http.StatusFound, res.StatusCode)
	require.Equal(t, urlToRedirect, redirectedToURL)
}

func testRedirectNotFound(t *testing.T, alias string) {
	u := url.URL{
		Scheme: "http",
		Host:   cfg.HTTPServer.Address,
		Path:   alias,
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // stop after 1st redirect
		},
	}

	res, err := client.Get(u.String())
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, res.StatusCode, http.StatusNotFound)

	var response response.Response
	err = json.NewDecoder(res.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, "not found", response.Error)
}
