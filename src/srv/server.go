package srv

import (
	"fmt"
	"net/http"

	c "../cloning"
	"../log"

	"github.com/gorilla/mux"
)

type Config struct {
	VerificationToken string `yaml:"verificationToken"`
	Port              uint   `yaml:"port"`
}

type Server struct {
	Config  *Config
	Cloning *c.Cloning
}

type Route struct {
	Route   string   `json:"route"`
	Methods []string `json:"methods"`
}

// Initializes Server instance with provided configuration.
func NewServer(cfg *Config, cloning *c.Cloning) *Server {
	// TODO(anatoly): Stop using mock data.
	server := &Server{
		Config:  cfg,
		Cloning: cloning,
	}

	return server
}

// Starts HTTP server on specified port in configuration.
func (s *Server) Run() error {
	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/status",
		s.authorized(s.getInstanceStatus())).Methods(http.MethodGet)
	r.HandleFunc("/snapshots",
		s.authorized(s.getSnapshots())).Methods(http.MethodGet)
	r.HandleFunc("/clone",
		s.authorized(s.startClone())).Methods(http.MethodPost)
	r.HandleFunc("/clone/{id}/reset",
		s.authorized(s.resetClone())).Methods(http.MethodPost)
	r.HandleFunc("/clone/{id}",
		s.authorized(s.getClone())).Methods(http.MethodGet)
	r.HandleFunc("/clone/{id}",
		s.authorized(s.patchClone())).Methods(http.MethodPatch)
	r.HandleFunc("/clone/{id}",
		s.authorized(s.stopClone())).Methods(http.MethodDelete)

	// Show available routes on index page.
	helpRoutes, err := getHelpRoutes(r)
	if err != nil {
		return err
	}
	r.HandleFunc("/", getHelp(helpRoutes))

	// Show not found error for all other possible routes.
	r.NotFoundHandler = http.HandlerFunc(failNotFound)

	// Start server.
	port := s.Config.Port
	log.Msg(fmt.Sprintf("Server started listening on localhost:%d.", port))
	err = http.ListenAndServe(fmt.Sprintf(":%d", port), logging(r))
	log.Err("HTTP server error:", err)

	return err
}
