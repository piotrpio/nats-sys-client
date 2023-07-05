package sys

import (
	"errors"
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

type Cluster struct {
	servers []*server.Server
}

// StartJetStreamServer starts a NATS server
func StartJetStreamServer(t *testing.T, confFile string) *server.Server {
	opts, err := server.ProcessConfigFile(confFile)

	if err != nil {
		t.Fatalf("Error processing config file: %v", err)
	}

	opts.NoLog = true
	opts.StoreDir = t.TempDir()

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

func SetupCluster(t *testing.T) *Cluster {
	t.Helper()
	cluster := Cluster{
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

func (c *Cluster) Shutdown() {
	for _, s := range c.servers {
		s.Shutdown()
	}
}

func TestRequestMany(t *testing.T) {
	c := SetupCluster(t)
	defer c.Shutdown()

	if len(c.servers) != 3 {
		t.Fatalf("Unexpected number of servers started: %d; want: %d", len(c.servers), 3)
	}

	tests := []struct {
		name              string
		subject           string
		opts              []RequestManyOpt
		expectedResponses int
		withError         error
	}{
		{
			name:              "default opts",
			subject:           "$SYS.REQ.SERVER.PING",
			expectedResponses: 3,
		},
		{
			name:              "expected responses count",
			subject:           "$SYS.REQ.SERVER.PING",
			opts:              []RequestManyOpt{WithRequestManyCount(2)},
			expectedResponses: 2,
		},
		{
			name:              "large interval, small timeout",
			subject:           "$SYS.REQ.SERVER.PING",
			opts:              []RequestManyOpt{WithRequestManyMaxInterval(10 * time.Second), WithRequestManyMaxWait(200 * time.Millisecond)},
			expectedResponses: 3,
		},
		{
			name:              "small interval, large timeout",
			subject:           "$SYS.REQ.SERVER.PING",
			opts:              []RequestManyOpt{WithRequestManyMaxInterval(100 * time.Millisecond), WithRequestManyMaxWait(5 * time.Second)},
			expectedResponses: 3,
		},
		{
			name:      "invalid count",
			subject:   "$SYS.REQ.SERVER.PING",
			opts:      []RequestManyOpt{WithRequestManyCount(-1)},
			withError: ErrValidation,
		},
		{
			name:      "invalid interval",
			subject:   "$SYS.REQ.SERVER.PING",
			opts:      []RequestManyOpt{WithRequestManyMaxInterval(-1)},
			withError: ErrValidation,
		},
		{
			name:      "invalid wait",
			subject:   "$SYS.REQ.SERVER.PING",
			opts:      []RequestManyOpt{WithRequestManyMaxWait(-1)},
			withError: ErrValidation,
		},
		{
			name:      "invalid subject",
			subject:   "",
			withError: nats.ErrBadSubject,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var urls []string
			for _, s := range c.servers {
				urls = append(urls, s.ClientURL())
			}

			sysConn, err := nats.Connect(strings.Join(urls, ","), nats.UserInfo("admin", "s3cr3t!"))
			if err != nil {
				t.Fatalf("Error establishing connection: %s", err)
			}
			sys := NewSysClient(sysConn)

			start := time.Now()
			resp, err := sys.RequestMany(test.subject, nil, test.opts...)
			dur := time.Since(start)
			if dur > time.Second {
				t.Fatalf("Request did not terminate in time and took %s", dur)
			}
			if test.withError != nil {
				if !errors.Is(err, test.withError) {
					t.Fatalf("Expected error; want: %s; got: %s", test.withError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unable to fetch CONNZ: %s", err)
			}

			if len(resp) != test.expectedResponses {
				t.Fatalf("Invalid number of responses; want: %d; got: %d", test.expectedResponses, len(resp))
			}
		})
	}
}
