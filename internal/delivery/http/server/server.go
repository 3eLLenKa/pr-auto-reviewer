package server

import (
	"context"
	"log/slog"
	"net/http"
)

type Server struct {
	server *http.Server
}

func New(addr string, handler http.Handler) *Server {
	s := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	return &Server{
		server: s,
	}
}

func (s *Server) Run() error {

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("failed to run server", slog.Any("err", err))
		return err
	}

	return nil
}

func (s *Server) MustRun() {
	if err := s.Run(); err != nil {
		panic(err)
	}
}

func (s *Server) Stop(ctx context.Context) {

	if err := s.server.Shutdown(ctx); err != nil {
		slog.Error("failed to stop http server gracefully", slog.Any("err", err))
	}
}
