package wait

import (
	"fmt"
	"net"
	"time"
)

func ForPortOpen(host, port string) error {
	timeout := time.After(20 * time.Second)
	tick := time.Tick(1 * time.Second)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("port %s was not open in time", port)
		case <-tick:
			conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), time.Second)
			if err != nil {
				return err
			}
			if conn != nil {
				defer func() { _ = conn.Close() }()
				return nil
			}
		}
	}
}
