package sys

import (
	"errors"
	"strings"
	"testing"

	"github.com/nats-io/nats.go"
)

func TestJsz(t *testing.T) {
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
		options   JszEventOptions
		withError error
	}{
		{
			name: "with valid id",
			id:   c.servers[1].ID(),
		},
		{
			name: "with stream details",
			id:   c.servers[1].ID(),
			options: JszEventOptions{
				JszOptions: JszOptions{Accounts: true},
			},
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

			jsz, err := sys.Jsz(test.id, test.options)
			if test.withError != nil {
				if !errors.Is(err, test.withError) {
					t.Fatalf("Expected error; want: %s; got: %s", test.withError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unable to fetch JSZ: %s", err)
			}
			if jsz.Server.ID != test.id {
				t.Fatalf("Invalid server JSZ response: %+v", jsz)
			}
			if !test.options.Accounts {
				if len(jsz.JSInfo.AccountDetails) != 0 {
					t.Fatalf("Expected no account details, got: %+v", jsz.JSInfo.AccountDetails)
				}
				return
			}
			if len(jsz.JSInfo.AccountDetails) == 0 {
				t.Fatalf("Expected account details in the response")
			}
		})
	}
}

func TestJszPing(t *testing.T) {
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

	nc, err := nats.Connect(strings.Join(urls, ","))
	if err != nil {
		t.Fatalf("Error establishing connection: %s", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("Error getting JetStream context: %v", err)
	}
	_, err = js.AddStream(&nats.StreamConfig{Name: "s1", Subjects: []string{"foo"}})
	if err != nil {
		t.Fatalf("Error creating stream: %v", err)
	}

	sys := NewSysClient(sysConn)

	resp, err := sys.JszPing(JszEventOptions{JszOptions: JszOptions{Streams: true}})
	if err != nil {
		t.Fatalf("Unable to fetch JSZ: %s", err)
	}
	if len(resp) != 3 {
		t.Fatalf("Invalid number of responses: %d; want: %d", len(resp), 3)
	}
	var streamsNum int
	for _, s := range c.servers {
		var seen bool
		for _, jsz := range resp {
			if s.ID() == jsz.Server.ID {
				seen = true
			}
			streamsNum += len(jsz.JSInfo.AccountDetails[0].Streams)
		}
		if !seen {
			t.Fatalf("Expected server %q in the response", s.Name())
		}
		if streamsNum == 0 {
			t.Fatalf("Expected stream details in the response")
		}
	}
}
