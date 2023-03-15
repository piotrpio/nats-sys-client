package sys

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/jwt"
	"github.com/nats-io/nats.go"
)

type (
	ConnzResp struct {
		Server ServerInfo `json:"server"`
		Connz  Connz      `json:"data"`
	}

	Connz struct {
		ID       string      `json:"server_id"`
		Now      time.Time   `json:"now"`
		NumConns int         `json:"num_connections"`
		Total    int         `json:"total"`
		Offset   int         `json:"offset"`
		Limit    int         `json:"limit"`
		Conns    []*ConnInfo `json:"connections"`
	}

	// ConnInfo has detailed information on a per connection basis.
	ConnInfo struct {
		Cid            uint64         `json:"cid"`
		Kind           string         `json:"kind,omitempty"`
		Type           string         `json:"type,omitempty"`
		IP             string         `json:"ip"`
		Port           int            `json:"port"`
		Start          time.Time      `json:"start"`
		LastActivity   time.Time      `json:"last_activity"`
		Stop           *time.Time     `json:"stop,omitempty"`
		Reason         string         `json:"reason,omitempty"`
		RTT            string         `json:"rtt,omitempty"`
		Uptime         string         `json:"uptime"`
		Idle           string         `json:"idle"`
		Pending        int            `json:"pending_bytes"`
		InMsgs         int64          `json:"in_msgs"`
		OutMsgs        int64          `json:"out_msgs"`
		InBytes        int64          `json:"in_bytes"`
		OutBytes       int64          `json:"out_bytes"`
		NumSubs        uint32         `json:"subscriptions"`
		Name           string         `json:"name,omitempty"`
		Lang           string         `json:"lang,omitempty"`
		Version        string         `json:"version,omitempty"`
		TLSVersion     string         `json:"tls_version,omitempty"`
		TLSCipher      string         `json:"tls_cipher_suite,omitempty"`
		TLSPeerCerts   []*TLSPeerCert `json:"tls_peer_certs,omitempty"`
		AuthorizedUser string         `json:"authorized_user,omitempty"`
		Account        string         `json:"account,omitempty"`
		Subs           []string       `json:"subscriptions_list,omitempty"`
		SubsDetail     []SubDetail    `json:"subscriptions_list_detail,omitempty"`
		JWT            string         `json:"jwt,omitempty"`
		IssuerKey      string         `json:"issuer_key,omitempty"`
		NameTag        string         `json:"name_tag,omitempty"`
		Tags           jwt.TagList    `json:"tags,omitempty"`
		MQTTClient     string         `json:"mqtt_client,omitempty"` // This is the MQTT client id
	}

	// TLSPeerCert contains basic information about a TLS peer certificate
	TLSPeerCert struct {
		Subject          string `json:"subject,omitempty"`
		SubjectPKISha256 string `json:"spki_sha256,omitempty"`
		CertSha256       string `json:"cert_sha256,omitempty"`
	}

	SubDetail struct {
		Account string `json:"account,omitempty"`
		Subject string `json:"subject"`
		Queue   string `json:"qgroup,omitempty"`
		Sid     string `json:"sid"`
		Msgs    int64  `json:"msgs"`
		Max     int64  `json:"max,omitempty"`
		Cid     uint64 `json:"cid"`
	}

	ConnzEventOptions struct {
		ConnzOptions
		EventFilterOptions
	}

	ConnzOptions struct {
		// Sort indicates how the results will be sorted. Check SortOpt for possible values.
		// Only the sort by connection ID (ByCid) is ascending, all others are descending.
		Sort SortOpt `json:"sort"`

		// Username indicates if user names should be included in the results.
		Username bool `json:"auth"`

		// Subscriptions indicates if subscriptions should be included in the results.
		Subscriptions bool `json:"subscriptions"`

		// SubscriptionsDetail indicates if subscription details should be included in the results
		SubscriptionsDetail bool `json:"subscriptions_detail"`

		// Offset is used for pagination. Connz() only returns connections starting at this
		// offset from the global results.
		Offset int `json:"offset"`

		// Limit is the maximum number of connections that should be returned by Connz().
		Limit int `json:"limit"`

		// Filter for this explicit client connection.
		CID uint64 `json:"cid"`

		// Filter for this explicit client connection based on the MQTT client ID
		MQTTClient string `json:"mqtt_client"`

		// Filter by connection state.
		State ConnState `json:"state"`

		// The below options only apply if auth is true.

		// Filter by username.
		User string `json:"user"`

		// Filter by account.
		Account string `json:"acc"`

		// Filter by subject interest
		FilterSubject string `json:"filter_subject"`
	}

	SortOpt string

	ConnState int
)

// Possible sort options
const (
	ByCid      SortOpt = "cid"        // By connection ID
	ByStart    SortOpt = "start"      // By connection start time, same as CID
	BySubs     SortOpt = "subs"       // By number of subscriptions
	ByPending  SortOpt = "pending"    // By amount of data in bytes waiting to be sent to client
	ByOutMsgs  SortOpt = "msgs_to"    // By number of messages sent
	ByInMsgs   SortOpt = "msgs_from"  // By number of messages received
	ByOutBytes SortOpt = "bytes_to"   // By amount of bytes sent
	ByInBytes  SortOpt = "bytes_from" // By amount of bytes received
	ByLast     SortOpt = "last"       // By the last activity
	ByIdle     SortOpt = "idle"       // By the amount of inactivity
	ByUptime   SortOpt = "uptime"     // By the amount of time connections exist
	ByStop     SortOpt = "stop"       // By the stop time for a closed connection
	ByReason   SortOpt = "reason"     // By the reason for a closed connection
)

const (
	// ConnOpen filters on open clients.
	ConnOpen = ConnState(iota)
	// ConnClosed filters on closed clients.
	ConnClosed
	// ConnAll returns all clients.
	ConnAll
)

// Connz returns server connection details
func (s *System) Connz(id string, opts ConnzEventOptions) (*ConnzResp, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: server id cannot be empty", ErrValidation)
	}
	conn := s.nc
	subj := fmt.Sprintf(srvConnzSubj, id)
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

	var connzResp ConnzResp
	if err := json.Unmarshal(resp.Data, &connzResp); err != nil {
		return nil, err
	}

	return &connzResp, nil
}

func (s *System) ConnzPing(opts ConnzEventOptions) ([]ConnzResp, error) {
	subj := fmt.Sprintf(srvConnzSubj, "PING")
	payload, err := json.Marshal(opts)
	if err != nil {
		return nil, err
	}
	resp, err := s.RequestMany(subj, payload)
	if err != nil {
		return nil, err
	}
	srvConnz := make([]ConnzResp, 0, len(resp))
	for _, msg := range resp {
		var connzResp ConnzResp
		if err := json.Unmarshal(msg.Data, &connzResp); err != nil {
			return nil, err
		}
		srvConnz = append(srvConnz, connzResp)
	}
	return srvConnz, nil
}
