package sys

import (
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

	id := c.servers[1].ID()

	var urls []string
	for _, s := range c.servers {
		urls = append(urls, s.ClientURL())
	}

	sysConn, err := nats.Connect(strings.Join(urls, ","), nats.UserInfo("admin", "s3cr3t!"))
	if err != nil {
		t.Fatalf("Error establishing connection: %s", err)
	}

	sys := NewSysClient(sysConn)

	connz, err := sys.Connz(id, ConnzEventOptions{})
	if err != nil {
		t.Fatalf("Unable to fetch CONNZ: %s", err)
	}
	if connz.Connz.ID != id {
		t.Fatalf("Invalid server CONNZ response: %+v", connz)
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

	sys := NewSysClient(sysConn)

	resp, err := sys.ConnzPing(ConnzEventOptions{})
	if err != nil {
		t.Fatalf("Unable to fetch CONNZ: %s", err)
	}
	if len(resp) != 3 {
		t.Fatalf("Invalid number of responses: %d; want: %d", len(resp), 3)
	}

	// TODO: implement valid test check
}
