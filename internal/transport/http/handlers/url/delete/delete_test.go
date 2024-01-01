package delete_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dkhrunov/url-shortener/internal/lib/logger/slog/handlers/slogdiscard"
	"github.com/dkhrunov/url-shortener/internal/storage"
	"github.com/dkhrunov/url-shortener/internal/transport/http/handlers/url/delete"
	"github.com/dkhrunov/url-shortener/internal/transport/http/handlers/url/delete/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestDeleteHandler(t *testing.T) {
	cases := []struct {
		name      string
		alias     string
		respError string
		mockError error
		status    int
	}{
		{
			name:   "Success",
			alias:  "test_alias",
			status: http.StatusOK,
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
			respError: "failed to delete url",
			mockError: errors.New("unexpected error"),
			status:    http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlDeleterMock := mocks.NewURLDeleter(t)

			if tc.respError == "" || tc.mockError != nil {
				urlDeleterMock.EXPECT().
					DeleteURL(tc.alias).
					Return(tc.mockError).
					Once()
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodDelete, "/url/{alias}", nil)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("alias", tc.alias)

			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

			handler := delete.New(slogdiscard.NewDiscardLogger(), urlDeleterMock)
			handler.ServeHTTP(w, r)

			require.Equal(t, tc.status, w.Code)

			body := w.Body.String()

			var resp delete.Response

			require.NoError(t, json.Unmarshal([]byte(body), &resp))

			require.Equal(t, tc.respError, resp.Error)
		})
	}
}
