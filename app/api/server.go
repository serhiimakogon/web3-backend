package api

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	addr       string
	httpServer *http.Server
	store      *proposalStore
}

func NewServer(port string) *Server {
	return &Server{
		addr:  net.JoinHostPort("0.0.0.0", port),
		store: newProposalStore(),
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.httpServer = &http.Server{
		Addr:         s.addr,
		Handler:      s.routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.httpServer.Shutdown(shutdownCtx)
	}()

	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)

	r.Get("/health", s.handleHealth)

	r.Route("/proposals", func(r chi.Router) {
		r.Get("/", s.handleListProposals)
		r.Get("/{proposalID}", s.handleGetProposal)
	})

	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListProposals(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, s.store.ListProposals())
}

func (s *Server) handleGetProposal(w http.ResponseWriter, r *http.Request) {
	proposalID, err := strconv.ParseUint(chi.URLParam(r, "proposalID"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid proposal id")
		return
	}

	proposal, ok := s.store.GetProposal(proposalID)
	if !ok {
		respondError(w, http.StatusNotFound, "proposal not found")
		return
	}

	respondJSON(w, http.StatusOK, proposal)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}
