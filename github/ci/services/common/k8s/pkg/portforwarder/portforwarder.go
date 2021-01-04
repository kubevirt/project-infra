package portforwarder

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/client"
)

// Buffer is a goroutine safe bytes.Buffer from https://gist.github.com/arkan/5924e155dbb4254b64614069ba0afd81
type Buffer struct {
	buffer bytes.Buffer
	mutex  sync.Mutex
}

// Write appends the contents of p to the buffer, growing the buffer as needed. It returns
// the number of bytes written.
func (s *Buffer) Write(p []byte) (n int, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.Write(p)
}

// String returns the contents of the unread portion of the buffer
// as a string.  If the Buffer is a nil pointer, it returns "<nil>".
func (s *Buffer) String() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.String()
}

// New creates a forwarder to podName's ports until stopChan is closed.
// Suggested in https://github.com/kubernetes/client-go/issues/51#issuecomment-436200428
func New(namespace, podName string, ports []string, stopChan <-chan struct{}) error {
	config, err := client.GetConfig()
	if err != nil {
		return err
	}
	roundTripper, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, podName)
	hostIP := strings.TrimLeft(config.Host, "htps:/")
	serverURL := url.URL{Scheme: "https", Path: path, Host: hostIP}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, &serverURL)

	readyChan := make(chan struct{}, 1)
	out, errOut := new(Buffer), new(Buffer)

	forwarder, err := portforward.New(dialer, ports, stopChan, readyChan, out, errOut)
	if err != nil {
		return err
	}

	go func() {
		for range readyChan { // Kubernetes will close this channel when it has something to tell us.
		}
		if len(errOut.String()) != 0 {
			panic(errOut.String())
		} else if len(out.String()) != 0 {
			fmt.Println(out.String())
		}
	}()

	if err = forwarder.ForwardPorts(); err != nil { // Locks until stopChan is closed.
		return err
	}
	return nil
}
