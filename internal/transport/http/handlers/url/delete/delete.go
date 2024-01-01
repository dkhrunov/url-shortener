package delete

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

type Response struct {
	response.Response
}

type URLDeleter interface {
	DeleteURL(alias string) error
}

func New(log *slog.Logger, urlDeleter URLDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.delete.New"

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

		err := urlDeleter.DeleteURL(alias)
		if errors.Is(err, storage.ErrURLNotFound) {
			log.Info("url not found", "alias", alias)

			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, response.Error("not found"))

			return
		}
		if err != nil {
			log.Error("failed to delete url", slogerr.Error(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error("failed to delete url"))

			return
		}

		log.Info("url deleted")

		render.JSON(w, r, Response{
			Response: response.OK(),
		})
	}
}
