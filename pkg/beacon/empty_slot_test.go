package beacon

import (
	"context"
	"testing"
	"time"

	"github.com/chuckpreslar/emission"
	"github.com/ethpandaops/ethwallclock"
	eapi "github.com/ethpandaops/go-eth2-client/api"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/all"
	"github.com/sirupsen/logrus"
)

type emptySlotFakeClient struct {
	blockErr error
	block    *spec.VersionedSignedBeaconBlock
}

func (f *emptySlotFakeClient) Name() string    { return "fake" }
func (f *emptySlotFakeClient) Address() string { return "fake://" }
func (f *emptySlotFakeClient) IsActive() bool  { return true }
func (f *emptySlotFakeClient) IsSynced() bool  { return true }

func (f *emptySlotFakeClient) SignedBeaconBlock(
	_ context.Context, _ *eapi.SignedBeaconBlockOpts,
) (*eapi.Response[*spec.VersionedSignedBeaconBlock], error) {
	if f.blockErr != nil {
		return nil, f.blockErr
	}

	return &eapi.Response[*spec.VersionedSignedBeaconBlock]{Data: f.block}, nil
}

func (f *emptySlotFakeClient) AgnosticSignedBeaconBlock(
	_ context.Context, _ *eapi.SignedBeaconBlockOpts,
) (*eapi.Response[*all.SignedBeaconBlock], error) {
	return nil, f.blockErr
}

func newEmptySlotTestNode(c *emptySlotFakeClient) *node {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)

	return &node{
		log:     log,
		options: &Options{DetectEmptySlots: true},
		stat:    NewStatus(1, 1),
		broker:  emission.NewEmitter(),
		client:  c,
	}
}

func TestCheckEmptySlot_PublishesOnMissingBlock(t *testing.T) {
	n := newEmptySlotTestNode(&emptySlotFakeClient{
		blockErr: &eapi.Error{Method: "GET", Endpoint: "/eth/v2/beacon/blocks/41", StatusCode: 404},
	})

	fired := make(chan *EmptySlotEvent, 1)
	n.OnEmptySlot(context.Background(), func(_ context.Context, event *EmptySlotEvent) error {
		fired <- event
		return nil
	})

	n.checkEmptySlot(context.Background(), ethwallclock.NewSlot(42, time.Now(), time.Now()))

	select {
	case event := <-fired:
		if event.Slot != 42 {
			t.Fatalf("expected empty slot event for slot 42, got %d", event.Slot)
		}
	case <-time.After(time.Second):
		t.Fatal("expected an empty slot event to fire on a missing block, none did")
	}
}

func TestCheckEmptySlot_DoesNotPublishWhenBlockExists(t *testing.T) {
	n := newEmptySlotTestNode(&emptySlotFakeClient{
		block: &spec.VersionedSignedBeaconBlock{},
	})

	fired := make(chan *EmptySlotEvent, 1)
	n.OnEmptySlot(context.Background(), func(_ context.Context, event *EmptySlotEvent) error {
		fired <- event
		return nil
	})

	n.checkEmptySlot(context.Background(), ethwallclock.NewSlot(42, time.Now(), time.Now()))

	select {
	case event := <-fired:
		t.Fatalf("expected no empty slot event when a block exists, got one for slot %d", event.Slot)
	case <-time.After(200 * time.Millisecond):
		// Good, nothing fired.
	}
}

func TestCheckEmptySlot_DisabledByDefault(t *testing.T) {
	n := newEmptySlotTestNode(&emptySlotFakeClient{
		blockErr: &eapi.Error{Method: "GET", Endpoint: "/eth/v2/beacon/blocks/41", StatusCode: 404},
	})
	n.options.DetectEmptySlots = false

	fired := make(chan *EmptySlotEvent, 1)
	n.OnEmptySlot(context.Background(), func(_ context.Context, event *EmptySlotEvent) error {
		fired <- event
		return nil
	})

	n.checkEmptySlot(context.Background(), ethwallclock.NewSlot(42, time.Now(), time.Now()))

	select {
	case event := <-fired:
		t.Fatalf("expected no empty slot event when detection is disabled, got one for slot %d", event.Slot)
	case <-time.After(200 * time.Millisecond):
		// Good, nothing fired.
	}
}
