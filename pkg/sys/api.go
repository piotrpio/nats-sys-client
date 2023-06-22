package sys

import (
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	srvVarzSubj    = "$SYS.REQ.SERVER.%s.VARZ"
	srvConnzSubj   = "$SYS.REQ.SERVER.%s.CONNZ"
	srvSubszSubj   = "$SYS.REQ.SERVER.%s.SUBSZ"
	srvHealthzSubj = "$SYS.REQ.SERVER.%s.HEALTHZ"
	srvJszSubj     = "$SYS.REQ.SERVER.%s.JSZ"
)

var (
	ErrValidation      = errors.New("validation error")
	ErrInvalidServerID = errors.New("sever with given ID does not exist")
)

// System can be used to request monitoring data from the server
type System struct {
	nc *nats.Conn
}

// ServerInfo identifies remote servers.
type ServerInfo struct {
	Name      string    `json:"name"`
	Host      string    `json:"host"`
	ID        string    `json:"id"`
	Cluster   string    `json:"cluster,omitempty"`
	Domain    string    `json:"domain,omitempty"`
	Version   string    `json:"ver"`
	Tags      []string  `json:"tags,omitempty"`
	Seq       uint64    `json:"seq"`
	JetStream bool      `json:"jetstream"`
	Time      time.Time `json:"time"`
}

func NewSysClient(nc *nats.Conn) System {
	return System{
		nc: nc,
	}
}

type requestManyOpts struct {
	maxWait     time.Duration
	maxInterval time.Duration
	count       int
}

type RequestManyOpt func(*requestManyOpts) error

func WithRequestManyMaxWait(maxWait time.Duration) RequestManyOpt {
	return func(opts *requestManyOpts) error {
		if maxWait <= 0 {
			return fmt.Errorf("%w: max wait has to be greater than 0", ErrValidation)
		}
		opts.maxWait = maxWait
		return nil
	}
}

func WithRequestManyMaxInterval(interval time.Duration) RequestManyOpt {
	return func(opts *requestManyOpts) error {
		if interval <= 0 {
			return fmt.Errorf("%w: max interval has to be greater than 0", ErrValidation)
		}
		opts.maxInterval = interval
		return nil
	}
}

func WithRequestManyCount(count int) RequestManyOpt {
	return func(opts *requestManyOpts) error {
		if count <= 0 {
			return fmt.Errorf("%w: expected request count has to be greater than 0", ErrValidation)
		}
		opts.count = count
		return nil
	}
}

func (s *System) RequestMany(subject string, data []byte, opts ...RequestManyOpt) ([]*nats.Msg, error) {
	if subject == "" {

	}

	conn := s.nc
	reqOpts := &requestManyOpts{
		maxWait:     DefaultRequestTimeout,
		maxInterval: 300 * time.Millisecond,
		count:       -1,
	}

	for _, opt := range opts {
		if err := opt(reqOpts); err != nil {
			return nil, err
		}
	}

	inbox := nats.NewInbox()
	res := make([]*nats.Msg, 0)
	msgsChan := make(chan *nats.Msg, 100)

	intervalTimer := time.NewTimer(reqOpts.maxInterval)
	sub, err := conn.Subscribe(inbox, func(msg *nats.Msg) {
		intervalTimer.Reset(reqOpts.maxInterval)
		msgsChan <- msg
	})
	defer sub.Unsubscribe()

	if err := conn.PublishRequest(subject, inbox, data); err != nil {
		return nil, err
	}

	for {
		select {
		case msg := <-msgsChan:
			if msg.Header.Get("Status") == "503" {
				return nil, fmt.Errorf("server request on subject %q failed: unauthorized", subject)
			}
			res = append(res, msg)
			if reqOpts.count != -1 && len(res) == reqOpts.count {
				return res, nil
			}
		case <-intervalTimer.C:
			return res, nil
		case <-time.After(reqOpts.maxWait):
			return res, nil
		}
	}
}

func jsonString(s string) string {
	return "\"" + s + "\""
}
