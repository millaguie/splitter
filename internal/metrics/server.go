package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
)

type Server struct {
	addr    string
	handler *Handler
	server  *http.Server
}

func NewServer(addr string, handler *Handler) *Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", handler)
	mux.HandleFunc("/healthz", handler.Healthz)
	return &Server{
		addr:    addr,
		handler: handler,
		server:  &http.Server{Addr: addr, Handler: mux},
	}
}

func (s *Server) Start(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		if err := s.server.Shutdown(context.Background()); err != nil {
			slog.Error("metrics server shutdown", "error", err)
		}
	}()

	slog.Info("metrics server starting", "addr", s.addr)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("metrics server: %w", err)
	}
	return nil
}
