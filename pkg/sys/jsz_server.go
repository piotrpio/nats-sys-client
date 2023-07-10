package sys

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type (
	JSZResp struct {
		Server ServerInfo `json:"server"`
		JSInfo JSInfo     `json:"data"`
	}

	JSInfo struct {
		ID       string          `json:"server_id"`
		Now      time.Time       `json:"now"`
		Disabled bool            `json:"disabled,omitempty"`
		Config   JetStreamConfig `json:"config,omitempty"`
		JetStreamStats
		Streams   int              `json:"streams"`
		Consumers int              `json:"consumers"`
		Messages  uint64           `json:"messages"`
		Bytes     uint64           `json:"bytes"`
		Meta      *MetaClusterInfo `json:"meta_cluster,omitempty"`

		// aggregate raft info
		AccountDetails []*AccountDetail `json:"account_details,omitempty"`
	}

	AccountDetail struct {
		Name string `json:"name"`
		Id   string `json:"id"`
		JetStreamStats
		Streams []StreamDetail `json:"stream_detail,omitempty"`
	}

	StreamDetail struct {
		Name               string                   `json:"name"`
		Created            time.Time                `json:"created"`
		Cluster            *nats.ClusterInfo        `json:"cluster,omitempty"`
		Config             *nats.StreamConfig       `json:"config,omitempty"`
		State              nats.StreamState         `json:"state,omitempty"`
		Consumer           []*nats.ConsumerInfo     `json:"consumer_detail,omitempty"`
		Mirror             *nats.StreamSourceInfo   `json:"mirror,omitempty"`
		Sources            []*nats.StreamSourceInfo `json:"sources,omitempty"`
		RaftGroup          string                   `json:"stream_raft_group,omitempty"`
		ConsumerRaftGroups []*RaftGroupDetail       `json:"consumer_raft_groups,omitempty"`
	}

	RaftGroupDetail struct {
		Name      string `json:"name"`
		RaftGroup string `json:"raft_group,omitempty"`
	}

	JszEventOptions struct {
		JszOptions
		EventFilterOptions
	}

	JszOptions struct {
		Account    string `json:"account,omitempty"`
		Accounts   bool   `json:"accounts,omitempty"`
		Streams    bool   `json:"streams,omitempty"`
		Consumer   bool   `json:"consumer,omitempty"`
		Config     bool   `json:"config,omitempty"`
		LeaderOnly bool   `json:"leader_only,omitempty"`
		Offset     int    `json:"offset,omitempty"`
		Limit      int    `json:"limit,omitempty"`
		RaftGroups bool   `json:"raft,omitempty"`
	}
)

// Jsz returns server jetstream details
func (s *System) Jsz(id string, opts JszEventOptions) (*JSZResp, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: server id cannot be empty", ErrValidation)
	}
	conn := s.nc
	subj := fmt.Sprintf(srvJszSubj, id)
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

	var jszResp JSZResp
	if err := json.Unmarshal(resp.Data, &jszResp); err != nil {
		return nil, err
	}

	return &jszResp, nil
}

func (s *System) JszPing(opts JszEventOptions) ([]JSZResp, error) {
	subj := fmt.Sprintf(srvJszSubj, "PING")
	payload, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	resp, err := s.RequestMany(subj, payload)
	if err != nil {
		return nil, err
	}
	srvJsz := make([]JSZResp, 0, len(resp))
	for _, msg := range resp {
		var jszResp JSZResp
		if err := json.Unmarshal(msg.Data, &jszResp); err != nil {
			return nil, err
		}
		srvJsz = append(srvJsz, jszResp)
	}
	return srvJsz, nil
}
