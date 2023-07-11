package sys

import (
	"errors"
	"strings"
	"testing"

	"github.com/nats-io/nats.go"
)

func TestConnz(t *testing.T) {
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
			sys, err := NewSysClient(sysConn)
			if err != nil {
				t.Fatalf("Error creating system client: %s", err)
			}

			connz, err := sys.Connz(test.id, ConnzEventOptions{})
			if test.withError != nil {
				if !errors.Is(err, test.withError) {
					t.Fatalf("Expected error; want: %s; got: %s", test.withError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unable to fetch CONNZ: %s", err)
			}
			if connz.Connz.ID != test.id {
				t.Fatalf("Invalid server CONNZ response: %+v", connz)
			}
		})
	}
}

func TestConnzPing(t *testing.T) {
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

	sys, err := NewSysClient(sysConn)
	if err != nil {
		t.Fatalf("Error creating system client: %s", err)
	}

	resp, err := sys.ConnzPing(ConnzEventOptions{})
	if err != nil {
		t.Fatalf("Unable to fetch CONNZ: %s", err)
	}
	if len(resp) != 3 {
		t.Fatalf("Invalid number of responses: %d; want: %d", len(resp), 3)
	}
	for _, s := range c.servers {
		var seen bool
		for _, connz := range resp {
			if s.ID() == connz.Connz.ID {
				seen = true
				break
			}
		}
		if !seen {
			t.Fatalf("Expected server %q in the response", s.Name())
		}
	}
}
