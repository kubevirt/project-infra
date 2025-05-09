package server

import (
	"net/http"

	"sigs.k8s.io/prow/pkg/github"

	"kubevirt.io/project-infra/external-plugins/test-subset/plugin/handler"
)

type GitHubEventsServer struct {
	tokenGenerator func() []byte
	eventsHandler  *handler.GitHubEventsHandler
}

func NewGitHubEventsServer(tokenGenerator func() []byte, eventsChan *handler.GitHubEventsHandler) *GitHubEventsServer {
	return &GitHubEventsServer{
		tokenGenerator: tokenGenerator,
		eventsHandler:  eventsChan,
	}
}

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
	if _, err := writer.Write([]byte("Event received. Have a nice day.")); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
	}
}
