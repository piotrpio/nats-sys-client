package sys

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type (
	SubszResp struct {
		Server ServerInfo `json:"server"`
		Subsz  Subsz      `json:"data"`
	}

	Subsz struct {
		ID  string    `json:"server_id"`
		Now time.Time `json:"now"`
		*SublistStats
		Total  int         `json:"total"`
		Offset int         `json:"offset"`
		Limit  int         `json:"limit"`
		Subs   []SubDetail `json:"subscriptions_list,omitempty"`
	}

	SublistStats struct {
		NumSubs      uint32  `json:"num_subscriptions"`
		NumCache     uint32  `json:"num_cache"`
		NumInserts   uint64  `json:"num_inserts"`
		NumRemoves   uint64  `json:"num_removes"`
		NumMatches   uint64  `json:"num_matches"`
		CacheHitRate float64 `json:"cache_hit_rate"`
		MaxFanout    uint32  `json:"max_fanout"`
		AvgFanout    float64 `json:"avg_fanout"`
	}

	// SubszOptions are the options passed to Subsz.
	SubszOptions struct {
		// Offset is used for pagination. Subsz() only returns connections starting at this
		// offset from the global results.
		Offset int `json:"offset"`

		// Limit is the maximum number of subscriptions that should be returned by Subsz().
		Limit int `json:"limit"`

		// Subscriptions indicates if subscription details should be included in the results.
		Subscriptions bool `json:"subscriptions"`

		// Filter based on this account name.
		Account string `json:"account,omitempty"`

		// Test the list against this subject. Needs to be literal since it signifies a publish subject.
		// We will only return subscriptions that would match if a message was sent to this subject.
		Test string `json:"test,omitempty"`
	}
)

// ServerSubsz returns server subscriptions data
func (s *System) ServerSubsz(id string, opts SubszOptions) (*SubszResp, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: server id cannot be empty", ErrValidation)
	}
	conn := s.nc
	subj := fmt.Sprintf(srvSubszSubj, id)
	payload, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	resp, err := conn.Request(subj, payload, DefaultRequestTimeout)
	if err != nil {
		if errors.Is(err, nats.ErrNoResponders) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidServerID, id)
		}
		return nil, err
	}

	var subszResp SubszResp
	if err := json.Unmarshal(resp.Data, &subszResp); err != nil {
		return nil, err
	}

	return &subszResp, nil
}

func (s *System) ServerSubszPing(opts SubszOptions) ([]SubszResp, error) {
	subj := fmt.Sprintf(srvSubszSubj, "PING")
	payload, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	resp, err := s.RequestMany(subj, payload)
	if err != nil {
		return nil, err
	}
	srvSubsz := make([]SubszResp, 0, len(resp))
	for _, msg := range resp {
		var subszResp SubszResp
		if err := json.Unmarshal(msg.Data, &subszResp); err != nil {
			return nil, err
		}
		srvSubsz = append(srvSubsz, subszResp)
	}
	return srvSubsz, nil
}
