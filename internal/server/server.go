package server

import (
	"context"
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"

	"github.com/tnguyen21/kestral-tui/internal/app"
	"github.com/tnguyen21/kestral-tui/internal/config"
)

// Server wraps a wish SSH server that serves the Kestral TUI.
type Server struct {
	config *config.Config
	wish   *ssh.Server
}

// New creates a Server configured from cfg.
func New(cfg *config.Config) (*Server, error) {
	teaHandler := func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
		model := app.New(*cfg)
		return model, []tea.ProgramOption{
			tea.WithAltScreen(),
			tea.WithMouseCellMotion(),
		}
	}

	s, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf(":%d", cfg.Port)),
		wish.WithHostKeyPath(filepath.Join(cfg.HostKeyDir, "kestral_host_key")),
		wish.WithPublicKeyAuth(publicKeyHandler),
		wish.WithMiddleware(
			bubbletea.Middleware(teaHandler),
			activeterm.Middleware(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating wish server: %w", err)
	}

	return &Server{config: cfg, wish: s}, nil
}

// Start begins listening for SSH connections. It blocks until the server
// is shut down or encounters a fatal error. Returns nil on graceful shutdown.
func (s *Server) Start() error {
	if err := s.wish.ListenAndServe(); err != nil && err != ssh.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.wish.Shutdown(ctx)
}

// publicKeyHandler accepts all SSH public keys. Kestral is designed for
// local/VPS use behind a firewall; key-based access control can be
// added later via ~/.ssh/authorized_keys.
func publicKeyHandler(_ ssh.Context, _ ssh.PublicKey) bool {
	return true
}
