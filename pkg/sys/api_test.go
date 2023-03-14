package sys

import (
	"fmt"
	"os"
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
