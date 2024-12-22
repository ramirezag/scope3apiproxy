package api

import (
	"context"
	"errors"
	"go.uber.org/zap"
	"net/http"
	v1 "scope3apiproxy/api/v1"
	"scope3apiproxy/internal"
	"strconv"
)

type APIServer struct {
	srv    *http.Server
	logger *zap.Logger
}

func NewAPIServer(port int, logger *zap.Logger, emissionService *internal.EmissionService) *APIServer {
	handler := v1.NewHandler(logger, emissionService)
	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: handler,
	}
	return &APIServer{
		srv:    srv,
		logger: logger,
	}
}

func (s *APIServer) Run() {
	s.logger.Info("HTTP server listening on " + s.srv.Addr)
	if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.Error("Error starting HTTP server on "+s.srv.Addr, zap.Error(err))
	}
}

func (s *APIServer) Shutdown(ctx context.Context, done chan bool) {
	if err := s.srv.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP APIServer shutdown exited with error", zap.Error(err))
		done <- true
	} else {
		s.logger.Info("HTTP APIServer shutdown normally")
		done <- true
	}
}
