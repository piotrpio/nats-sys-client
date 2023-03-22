package sys

import (
	"errors"
	"strings"
	"testing"

	"github.com/nats-io/nats.go"
)

func TestHealthz(t *testing.T) {
	c := SetupCluster(t)
	defer c.Shutdown()

	if len(c.servers) != 3 {
		t.Fatalf("Unexpected number of servers started: %d; want: %d", len(c.servers), 3)
	}

	var urls []string
	for _, s := range c.servers {
		urls = append(urls, s.ClientURL())
	}

	sysConn, err := nats.Connect(strings.Join(urls, ","), nats.UserInfo("admin", "s3cr3t!"))
	if err != nil {
		t.Fatalf("Error establishing connection: %s", err)
	}

	tests := []struct {
		name      string
		id        string
		withError error
	}{
		{
			name: "with valid id",
			id:   c.servers[1].ID(),
		},
		{
			name:      "with empty id",
			id:        "",
			withError: ErrValidation,
		},
		{
			name:      "with invalid id",
			id:        "asd",
			withError: ErrInvalidServerID,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sys := NewSysClient(sysConn)

			healthz, err := sys.Healthz(test.id, HealthzOptions{})
			if test.withError != nil {
				if !errors.Is(err, test.withError) {
					t.Fatalf("Expected error; want: %s; got: %s", test.withError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unable to fetch HEALTHZ: %s", err)
			}
			if healthz.Healthz.Status != StatusOK {
				t.Fatalf("Invalid server HEALTHZ response: %+v", healthz)
			}
		})
	}
}

func TestHealthzPing(t *testing.T) {
	c := SetupCluster(t)
	defer c.Shutdown()

	if len(c.servers) != 3 {
		t.Fatalf("Unexpected number of servers started: %d; want: %d", len(c.servers), 3)
	}

	var urls []string
	for _, s := range c.servers {
		urls = append(urls, s.ClientURL())
	}

	sysConn, err := nats.Connect(strings.Join(urls, ","), nats.UserInfo("admin", "s3cr3t!"))
	if err != nil {
		t.Fatalf("Error establishing connection: %s", err)
	}

	sys := NewSysClient(sysConn)

	resp, err := sys.HealthzPing(HealthzOptions{})
	if err != nil {
		t.Fatalf("Unable to fetch HEALTHZ: %s", err)
	}
	if len(resp) != 3 {
		t.Fatalf("Invalid number of responses: %d; want: %d", len(resp), 3)
	}
	for _, healthz := range resp {
		if healthz.Healthz.Status != StatusOK {
			t.Fatalf("Invalid server HEALTHZ response: %+v", healthz)
		}
	}
}
