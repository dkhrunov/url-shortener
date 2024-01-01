package redirect

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/dkhrunov/url-shortener/internal/lib/logger/slog/slogerr"
	"github.com/dkhrunov/url-shortener/internal/storage"
	"github.com/dkhrunov/url-shortener/internal/transport/http/common/response"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type URLGetter interface {
	GetURL(alias string) (string, error)
}

func New(log *slog.Logger, urlGetter URLGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handler.redirect.New"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			log.Info("alias is empty")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error("invalid request"))

			return
		}

		resURL, err := urlGetter.GetURL(alias)
		if errors.Is(err, storage.ErrURLNotFound) {
			log.Info("url not found", "alias", alias)

			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, response.Error("not found"))

			return
		}
		if err != nil {
			log.Error("failed to get url", slogerr.Error(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to get url"))

			return
		}

		log.Info("got url", slog.String("url", resURL))

		// redirect to URL
		http.Redirect(w, r, resURL, http.StatusFound)
	}
}
