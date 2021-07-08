package metrics

import (
	"fmt"
	"log"
	"strings"

	prowConfig "k8s.io/test-infra/prow/config"
)

var (
	exporters []switchable
)

type Config struct {
	ProwConfig *prowConfig.Config
}

type Handler struct{}

type switchable interface {
	Start(Config) error
	Stop() error
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Start(c Config) error {
	log.Println("Starting exporters...")
	errors := []string{}

	for _, e := range exporters {
		if err := e.Start(c); err != nil {
			errors = append(errors, err.Error())
		}
	}
	if len(errors) != 0 {
		return fmt.Errorf("Could not start exporters: %q", strings.Join(errors, ";"))
	}
	return nil
}

func (h *Handler) Stop() error {
	log.Println("Stopping exporters...")
	errors := []string{}

	for _, e := range exporters {
		if err := e.Stop(); err != nil {
			errors = append(errors, err.Error())
		}
	}
	if len(errors) != 0 {
		return fmt.Errorf("Could not stop exporters: %q", strings.Join(errors, ";"))
	}
	return nil
}
