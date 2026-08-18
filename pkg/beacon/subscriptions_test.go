package beacon

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	eapi "github.com/ethpandaops/go-eth2-client/api"
	"github.com/sirupsen/logrus"
)

// eventsFakeClient records how many times each topic was subscribed and
// fails deterministically for a configured bad topic, mirroring how
// go-eth2-client validates topics client-side before opening a stream.
type eventsFakeClient struct {
	mu       sync.Mutex
	calls    map[string]int
	badTopic string
}

func (f *eventsFakeClient) Name() string    { return "fake" }
func (f *eventsFakeClient) Address() string { return "fake://" }
func (f *eventsFakeClient) IsActive() bool  { return true }
func (f *eventsFakeClient) IsSynced() bool  { return true }

func (f *eventsFakeClient) Events(_ context.Context, opts *eapi.EventsOpts) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, topic := range opts.Topics {
		if topic == f.badTopic {
			return errors.New("unsupported event topic " + topic)
		}

		f.calls[topic]++
	}

	return nil
}

func (f *eventsFakeClient) counts() map[string]int {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make(map[string]int, len(f.calls))
	for k, v := range f.calls {
		out[k] = v
	}

	return out
}

func newSubscriptionTestNode(c *eventsFakeClient) *node {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)

	return &node{
		log:     log,
		options: DefaultOptions(),
		config:  &Config{},
		client:  c,
	}
}

func TestEnsureBeaconSubscription_SkipsBadTopicAndStaysIdempotent(t *testing.T) {
	c := &eventsFakeClient{
		calls:    map[string]int{},
		badTopic: "raw_event",
	}

	n := newSubscriptionTestNode(c)
	n.options.BeaconSubscription.Enabled = true
	n.options.BeaconSubscription.Topics = EventTopics{
		topicBlock,
		topicHead,
		"raw_event", // never supported, should be skipped forever
		topicChainReorg,
	}

	// Retries every 2s; give it enough time for several attempts.
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	// ensureBeaconSubscription only returns once every topic subscribes,
	// which never happens here because of the permanently bad topic, so it
	// will return ctx.Err() when the deadline hits. That's expected.
	_ = n.ensureBeaconSubscription(ctx)

	counts := c.counts()

	if counts[topicBlock] != 1 {
		t.Fatalf("expected %q to be subscribed exactly once, got %d", topicBlock, counts[topicBlock])
	}

	if counts[topicHead] != 1 {
		t.Fatalf("expected %q to be subscribed exactly once, got %d", topicHead, counts[topicHead])
	}

	if counts[topicChainReorg] != 1 {
		t.Fatalf("expected %q (after the bad topic) to be subscribed exactly once, got %d",
			topicChainReorg, counts[topicChainReorg])
	}

	if _, ok := counts["raw_event"]; ok {
		t.Fatalf("expected the bad topic to never succeed, but it recorded a call")
	}
}

func TestEnsureBeaconSubscription_ReturnsOnceAllTopicsSubscribe(t *testing.T) {
	c := &eventsFakeClient{calls: map[string]int{}}

	n := newSubscriptionTestNode(c)
	n.options.BeaconSubscription.Enabled = true
	n.options.BeaconSubscription.Topics = EventTopics{topicBlock, topicHead}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := n.ensureBeaconSubscription(ctx)
	if err != nil {
		t.Fatalf("expected ensureBeaconSubscription to return nil once all topics subscribe, got %v", err)
	}

	counts := c.counts()

	if counts[topicBlock] != 1 || counts[topicHead] != 1 {
		t.Fatalf("expected each topic subscribed exactly once, got %v", counts)
	}
}
