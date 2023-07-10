package sys

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type (
	ServerStatszResp struct {
		Server ServerInfo  `json:"server"`
		Statsz ServerStats `json:"statsz"`
	}

	// ServerStats hold various statistics that we will periodically send out.
	ServerStats struct {
		Start            time.Time      `json:"start"`
		Mem              int64          `json:"mem"`
		Cores            int            `json:"cores"`
		CPU              float64        `json:"cpu"`
		Connections      int            `json:"connections"`
		TotalConnections uint64         `json:"total_connections"`
		ActiveAccounts   int            `json:"active_accounts"`
		NumSubs          uint32         `json:"subscriptions"`
		Sent             DataStats      `json:"sent"`
		Received         DataStats      `json:"received"`
		SlowConsumers    int64          `json:"slow_consumers"`
		Routes           []*RouteStat   `json:"routes,omitempty"`
		Gateways         []*GatewayStat `json:"gateways,omitempty"`
		ActiveServers    int            `json:"active_servers,omitempty"`
		JetStream        *JetStreamVarz `json:"jetstream,omitempty"`
	}

	// DataStats reports how may msg and bytes. Applicable for both sent and received.
	DataStats struct {
		Msgs  int64 `json:"msgs"`
		Bytes int64 `json:"bytes"`
	}

	// RouteStat holds route statistics.
	RouteStat struct {
		ID       uint64    `json:"rid"`
		Name     string    `json:"name,omitempty"`
		Sent     DataStats `json:"sent"`
		Received DataStats `json:"received"`
		Pending  int       `json:"pending"`
	}

	// GatewayStat holds gateway statistics.
	GatewayStat struct {
		ID         uint64    `json:"gwid"`
		Name       string    `json:"name"`
		Sent       DataStats `json:"sent"`
		Received   DataStats `json:"received"`
		NumInbound int       `json:"inbound_connections"`
	}

	// StatszEventOptions are options passed to Statsz
	StatszEventOptions struct {
		EventFilterOptions
	}
)

// ServerStatsz returns server statistics for the Statsz structs
func (s *System) ServerStatsz(id string, opts StatszEventOptions) (*ServerStatszResp, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: server id cannot be empty", ErrValidation)
	}
	conn := s.nc
	subj := fmt.Sprintf(srvStatszSubj, id)
	payload, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	resp, err := conn.Request(subj, payload, s.opts.timeout)
	if err != nil {
		if errors.Is(err, nats.ErrNoResponders) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidServerID, id)
		}
		return nil, err
	}

	var statszResp ServerStatszResp
	if err := json.Unmarshal(resp.Data, &statszResp); err != nil {
		return nil, err
	}

	return &statszResp, nil
}

func (s *System) ServerStatszPing(opts StatszEventOptions) ([]ServerStatszResp, error) {
	subj := fmt.Sprintf(srvStatszSubj, "PING")
	payload, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	resp, err := s.RequestMany(subj, payload)
	if err != nil {
		return nil, err
	}
	srvStatsz := make([]ServerStatszResp, 0, len(resp))
	for _, msg := range resp {
		var statszResp ServerStatszResp
		if err := json.Unmarshal(msg.Data, &statszResp); err != nil {
			return nil, err
		}
		srvStatsz = append(srvStatsz, statszResp)
	}
	return srvStatsz, nil
}
