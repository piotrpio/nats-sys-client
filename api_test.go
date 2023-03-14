package sys

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

var configFiles = []string{
	"./testdata/s1.conf",
	"./testdata/s2.conf",
	"./testdata/s3.conf",
}

type cluster struct {
	servers []*server.Server
}

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

	sys := NewSysClient(sysConn)

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

	sys := NewSysClient(sysConn)

	resp, err := sys.VarzPing(VarzEventOptions{})
	if err != nil {
		t.Fatalf("Unable to fetch VARZ: %s", err)
	}
	if len(resp) != 3 {
		t.Fatalf("Invalid number of responses: %d; want: %d", len(resp), 3)
	}
	// fmt.Printf("%+v", resp)
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

// StartJetStreamServer starts a a NATS server
func StartJetStreamServer(t *testing.T, confFile string) *server.Server {
	opts, err := server.ProcessConfigFile(confFile)

	if err != nil {
		t.Fatalf("Error processing config file: %v", err)
	}

	opts.NoLog = true
	tdir, err := os.MkdirTemp(os.TempDir(), fmt.Sprintf("%s_%s-", opts.ServerName, opts.Cluster.Name))
	if err != nil {
		t.Fatalf("Error creating jetstream store directory: %s", err)
	}
	opts.StoreDir = tdir

	s, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("Error creating server: %s", err)
	}

	s.Start()

	if !s.ReadyForConnections(10 * time.Second) {
		t.Fatal("Unable to start NATS Server")
	}
	return s
}

func SetupCluster(t *testing.T) *cluster {
	t.Helper()
	cluster := cluster{
		servers: make([]*server.Server, 0),
	}
	for _, confFile := range configFiles {
		srv := StartJetStreamServer(t, confFile)
		cluster.servers = append(cluster.servers, srv)
	}
	for _, s := range cluster.servers {
		nc, err := nats.Connect(s.ClientURL())
		if err != nil {
			t.Fatalf("Unable to connect to server %q: %s", s.Name(), err)
		}

		// wait until JetStream is ready
		timeout := time.Now().Add(10 * time.Second)
		for time.Now().Before(timeout) {
			jsm, err := nc.JetStream()
			if err != nil {
				t.Fatal(err)
			}
			_, err = jsm.AccountInfo()
			if err != nil {
				time.Sleep(500 * time.Millisecond)
			}
			break
		}
		if err != nil {
			t.Fatalf("Unexpected error creating stream: %v", err)
		}
	}
	return &cluster
}

func (c *cluster) Shutdown() {
	for _, s := range c.servers {
		s.Shutdown()
	}
}
