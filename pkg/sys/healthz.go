package sys

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
)

type (
	HealthzResp struct {
		Server  ServerInfo `json:"server"`
		Healthz Healthz    `json:"data"`
	}

	Healthz struct {
		Status HealthStatus `json:"status"`
		Error  string       `json:"error,omitempty"`
	}

	HealthStatus int

	// HealthzOptions are options passed to Healthz
	HealthzOptions struct {
		JSEnabledOnly bool `json:"js-enabled-only,omitempty"`
		JSServerOnly  bool `json:"js-server-only,omitempty"`
	}
)

const (
	StatusOK HealthStatus = iota
	StatusUnavailable
	StatusError
)

func (hs *HealthStatus) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case jsonString("ok"):
		*hs = StatusOK
	case jsonString("na"):
		*hs = StatusUnavailable
	case jsonString("error"):
		*hs = StatusError
	default:
		return fmt.Errorf("cannot unmarshal %q", data)
	}

	return nil
}

func (hs HealthStatus) MarshalJSON() ([]byte, error) {
	switch hs {
	case StatusOK:
		return json.Marshal("ok")
	case StatusUnavailable:
		return json.Marshal("na")
	case StatusError:
		return json.Marshal("error")
	default:
		return nil, fmt.Errorf("unknown health status: %v", hs)
	}
}

func (hs HealthStatus) String() string {
	switch hs {
	case StatusOK:
		return "ok"
	case StatusUnavailable:
		return "na"
	case StatusError:
		return "error"
	default:
		return "unknown health status"
	}
}

// Healthz checks server health status
func (s *System) Healthz(id string, opts HealthzOptions) (*HealthzResp, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: server id cannot be empty", ErrValidation)
	}
	conn := s.nc
	subj := fmt.Sprintf(srvHealthzSubj, id)
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

	var healthzResp HealthzResp
	if err := json.Unmarshal(resp.Data, &healthzResp); err != nil {
		return nil, err
	}

	return &healthzResp, nil
}

func (s *System) HealthzPing(opts HealthzOptions) ([]HealthzResp, error) {
	subj := fmt.Sprintf(srvHealthzSubj, "PING")
	payload, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	resp, err := s.RequestMany(subj, payload)
	if err != nil {
		return nil, err
	}
	srvHealthz := make([]HealthzResp, 0, len(resp))
	for _, msg := range resp {
		var healthzResp HealthzResp
		if err := json.Unmarshal(msg.Data, &healthzResp); err != nil {
			return nil, err
		}
		srvHealthz = append(srvHealthz, healthzResp)
	}
	return srvHealthz, nil
}
