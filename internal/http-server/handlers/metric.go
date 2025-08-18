package handlers

import (
    "net/http"
    "github.com/go-chi/render"
    "log/slog"
    "clic-metric/internal/lib/api/response"
    "clic-metric/internal/lib/logger/sl"
    "clic-metric/internal/storage"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/go-chi/chi/v5"
)

type MetricGetter interface {
    GetClicks(alias string) (int, error)
}

func New(log *slog.Logger, mg MetricGetter) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        const op = "handlers.url.metric.New"
        log := log.With(
            slog.String("op", op),
            slog.String("request_id", middleware.GetReqID(r.Context())),
        )
        alias := chi.URLParam(r, "alias")
        if alias == "" {
            render.JSON(w, r, response.Error("alias is required"))
            return
        }
        clicks, err := mg.GetClicks(alias)
        if err == storage.ErrUrlNotFound {
            render.JSON(w, r, response.Error("not found"))
            return
        }
        if err != nil {
            log.Error("failed to get clicks", sl.Err(err))
            render.JSON(w, r, response.Error("internal error"))
            return
        }
        render.JSON(w, r, map[string]interface{}{
            "alias":  alias,
            "clicks": clicks,
        })
    }
}