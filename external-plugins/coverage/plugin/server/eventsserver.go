package server

import (
	"net/http"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/github"

	"kubevirt.io/project-infra/external-plugins/coverage/plugin/handler"
)

// GitHubEventsServer handles incoming GitHub webhook requests and routes them to the appropriate handler
type GitHubEventsServe struct {
	tokenGenerator func() []byte
	eventsChan     chan<- *handler.GitHubEvent //event channel to send events to the handler
}

// New GitHubEventsServer creates a new webhook server
func NewGitHubEventsServer(tokenGenerator func() []byte, eventsChan chan<- *handler.GitHubEvent) *GitHubEventsServer {
	return &GitHubEventsServe{
		tokenGenerator: tokenGenerator,
		eventsChan:     eventsChan,
	}
}

// Server HTTP handler for incoming webhooks
func (s *GitHubEventsServe) ServerHTTP(writer http.ResponseWriter, request *http.Request) {
	//Validate webhook signature
	eventType, eventGUID, eventPayload, eventOk, _ := github.ValidateWebhook(writer, request, s.tokenGenerator)

	//if validation fails return 400
	if !eventOk {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	//create event object
	event := &handler.GitHubEvent{
		Type:    eventType,
		GUID:    eventGUID,
		Payload: eventPayload,
	}

	//handle event asynchronously
	select {
	case s.eventsChan <- event:
		logrus.Debugf("Event %d queued for processing", eventGUID)
	default:
		logrus.Warnf("Event channel full, event s% dropped", eventGUID)
	}

	//return response
	writer.WriteHeader(http.StatusOK)
	writer.Write([]byte("Event received. Have a nice day"))
}
