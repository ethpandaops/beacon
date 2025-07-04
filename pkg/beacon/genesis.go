package beacon

import (
	"context"
	"errors"

	eth2client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/api"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
)

func (n *node) FetchGenesis(ctx context.Context) (*v1.Genesis, error) {
	provider, isProvider := n.client.(eth2client.GenesisProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.GenesisProvider")
	}

	rsp, err := provider.Genesis(ctx, &api.GenesisOpts{})
	if err != nil {
		return nil, err
	}

	n.genesisMu.Lock()
	n.genesis = rsp.Data
	n.genesisMu.Unlock()

	return rsp.Data, nil
}
