package sys

import (
	"strings"
	"testing"

	"github.com/nats-io/nats.go"
)

func TestVarz(t *testing.T) {
	c := SetupCluster(t)
	defer c.Shutdown()

	if len(c.servers) != 3 {
		t.Fatalf("Unexpected number of servers started: %d; want: %d", len(c.servers), 3)
	}

	id := c.servers[1].ID()

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

	varz, err := sys.Varz(id, VarzEventOptions{})
	if err != nil {
		t.Fatalf("Unable to fetch VARZ: %s", err)
	}
	if varz.Varz.ID != id {
		t.Fatalf("Invalid server varz response: %+v", varz)
	}
}

func TestVarzPing(t *testing.T) {
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

	resp, err := sys.VarzPing(VarzEventOptions{})
	if err != nil {
		t.Fatalf("Unable to fetch VARZ: %s", err)
	}
	if len(resp) != 3 {
		t.Fatalf("Invalid number of responses: %d; want: %d", len(resp), 3)
	}
	for _, s := range c.servers {
		var seen bool
		for _, varz := range resp {
			if s.Name() == varz.Varz.Name {
				seen = true
				break
			}
		}
		if !seen {
			t.Fatalf("Expected server %q in the response", s.Name())
		}
	}
}
