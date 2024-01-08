package redirect_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dkhrunov/url-shortener/internal/storage"
	"github.com/dkhrunov/url-shortener/internal/transport/http/common/response"
	"github.com/dkhrunov/url-shortener/internal/transport/http/handlers/redirect"
	"github.com/dkhrunov/url-shortener/internal/transport/http/handlers/redirect/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedirectHandler(t *testing.T) {
	cases := []struct {
		name      string
		alias     string
		url       string
		respError string
		mockError error
		status    int
	}{
		{
			name:   "Success",
			alias:  "test_alias",
			url:    "https://google.com",
			status: http.StatusFound,
		},
		{
			name:      "Empty alias",
			alias:     "",
			respError: "invalid request",
			status:    http.StatusBadRequest,
		},
		{
			name:      "Not found",
			alias:     "test_alias",
			respError: "not found",
			mockError: storage.ErrURLNotFound,
			status:    http.StatusNotFound,
		},
		{
			name:      "Failed",
			alias:     "test_alias",
			respError: "failed to get url",
			mockError: errors.New("unexpected error"),
			status:    http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlGetterMock := mocks.NewURLGetter(t)

			if tc.respError == "" || tc.mockError != nil {
				urlGetterMock.EXPECT().
					GetURL(tc.alias).
					Return(tc.url, tc.mockError).
					Once()
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodDelete, "/{alias}", nil)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("alias", tc.alias)

			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

			handler := redirect.New(urlGetterMock)
			handler.ServeHTTP(w, r)

			if tc.respError != "" {
				body := w.Body.String()

				var resp response.Response

				require.NoError(t, json.Unmarshal([]byte(body), &resp))

				assert.Equal(t, tc.respError, resp.Error)
			}

			assert.Equal(t, tc.status, w.Code)

			assert.Equal(t, tc.url, w.Header().Get("Location"))
		})
	}
}
