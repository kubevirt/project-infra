package server

import (
	"net/http"

	"sigs.k8s.io/prow/pkg/github"
	"kubevirt.io/project-infra/external-plugins/coverage/plugin/handler"
)

// GitHubEventsServer handles incoming GitHub webhook requests and routes them to the appropriate handler
type GitHubEventsServer struct {
	tokenGenerator func() []byte
	eventsHandler  *handler.GitHubEventsHandler
}

// New GitHubEventsServer creates a new webhook server
func NewGitHubEventsServer(tokenGenerator func() []byte, eventsHandler *handler.GitHubEventsHandler) *GitHubEventsServer {
	return &GitHubEventsServer{
		tokenGenerator: tokenGenerator,
		eventsHandler:  eventsHandler,
	}
}

// Serve HTTP handler for incoming webhooks
func (s *GitHubEventsServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	eventType, eventGUID, eventPayload, eventOk, _ := github.ValidateWebhook(writer, request, s.tokenGenerator)

	if !eventOk {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	event := &handler.GitHubEvent{
		Type:    eventType,
		GUID:    eventGUID,
		Payload: eventPayload,
	}

	go s.eventsHandler.Handle(event)
	writer.Write([]byte("Event received. Have a nice day."))
}
