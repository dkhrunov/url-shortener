package save_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dkhrunov/url-shortener/internal/storage"
	"github.com/dkhrunov/url-shortener/internal/transport/http/handlers/url/save"
	"github.com/dkhrunov/url-shortener/internal/transport/http/handlers/url/save/mocks"
)

func TestSaveHandler(t *testing.T) {
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
			status: http.StatusOK,
		},
		{
			name:   "Empty alias",
			alias:  "",
			url:    "https://google.com",
			status: http.StatusOK,
		},
		{
			name:      "Empty URL",
			alias:     "some_alias",
			url:       "",
			respError: "field URL is a required field",
			status:    http.StatusBadRequest,
		},
		{
			name:      "Invalid URL",
			alias:     "some_alias",
			url:       "some invalid URL",
			respError: "field URL is not a valid URL",
			status:    http.StatusBadRequest,
		},
		{
			name:      "Alias exist",
			alias:     "test_alias",
			url:       "https://google.com",
			respError: "url already exists",
			mockError: storage.ErrURLExist,
			status:    http.StatusConflict,
		},
		{
			name:      "SaveURL Error",
			alias:     "test_alias",
			url:       "https://google.com",
			respError: "failed to add url",
			mockError: errors.New("unexpected error"),
			status:    http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlSaverMock := mocks.NewURLSaver(t)

			if tc.respError == "" || tc.mockError != nil {
				urlSaverMock.EXPECT().
					SaveURL(tc.url, mock.AnythingOfType("string")).
					Return(int64(1), tc.mockError).
					Once()
			}

			handler := save.New(urlSaverMock)

			input := fmt.Sprintf(`{"url": "%s", "alias": "%s"}`, tc.url, tc.alias)

			req, err := http.NewRequest(http.MethodPost, "/save", bytes.NewReader([]byte(input)))
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.status, rr.Code)

			body := rr.Body.String()

			var resp save.Response

			require.NoError(t, json.Unmarshal([]byte(body), &resp))

			require.Equal(t, tc.respError, resp.Error)

			// TODO: add more checks
		})
	}
}
