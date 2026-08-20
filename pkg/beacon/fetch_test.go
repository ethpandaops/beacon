package beacon

import (
	"context"
	"sync"
	"testing"

	"github.com/chuckpreslar/emission"
	"github.com/ethpandaops/beacon/pkg/beacon/api/types"
	"github.com/sirupsen/logrus"
)

type peersFakeAPI struct{}

func (f *peersFakeAPI) NodePeer(context.Context, string) (types.Peer, error) {
	return types.Peer{}, nil
}

func (f *peersFakeAPI) NodePeers(context.Context) (types.Peers, error) {
	return types.Peers{{PeerID: "a"}, {PeerID: "b"}}, nil
}

func (f *peersFakeAPI) NodePeerCount(context.Context) (types.PeerCount, error) {
	return types.PeerCount{}, nil
}

func (f *peersFakeAPI) RawBlock(context.Context, string, string) ([]byte, error) { return nil, nil }

func (f *peersFakeAPI) RawDebugBeaconState(context.Context, string, string) ([]byte, error) {
	return nil, nil
}

func (f *peersFakeAPI) DepositSnapshot(context.Context) (*types.DepositSnapshot, error) {
	return nil, nil
}

func (f *peersFakeAPI) NodeIdentity(context.Context) (*types.Identity, error) { return nil, nil }

// TestFetchPeersConcurrent exercises concurrent FetchPeers calls, the shape
// of the 60s peers cron running alongside a consumer calling FetchPeers
// directly. Should be clean under -race now that FetchPeers no longer
// writes to a shared, unguarded field.
func TestFetchPeersConcurrent(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.PanicLevel)

	n := &node{
		log:    log,
		broker: emission.NewEmitter(),
		api:    &peersFakeAPI{},
	}

	ctx := context.Background()

	var wg sync.WaitGroup

	for range 8 {
		wg.Go(func() {
			for range 50 {
				if _, err := n.FetchPeers(ctx); err != nil {
					t.Error(err)
				}
			}
		})
	}

	wg.Wait()
}
