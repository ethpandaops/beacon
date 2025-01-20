package beacon

import (
	"context"
	"net/http"
	"time"

	eth2client "github.com/attestantio/go-eth2-client"
	ehttp "github.com/attestantio/go-eth2-client/http"
	"github.com/ethpandaops/beacon/pkg/beacon/api"
)

// ensureClients ensures that the node has a client and an API client.
func (n *node) ensureClients(ctx context.Context) error {
	failures := 0

	zerologLevel := n.GetZeroLogLevel()

	for {
		if n.client != nil {
			_, isProvider := n.client.(eth2client.NodeSyncingProvider)
			if isProvider {
				break
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			timeout := 10 * time.Minute

			client, err := ehttp.New(ctx,
				ehttp.WithAddress(n.config.Addr),
				ehttp.WithLogLevel(zerologLevel),
				ehttp.WithTimeout(timeout),
				ehttp.WithExtraHeaders(n.config.Headers),
				ehttp.WithEnforceJSON(true), // TODO(sam): HACK: Remove this once we have a proper spec
			)
			if err != nil {
				failures++

				sleepFor := time.Duration(failures) * (time.Second * 5)

				// Clamp the sleep time to a maximum of 5 minutes.
				if sleepFor > time.Minute*5 {
					sleepFor = time.Minute * 5
				}

				n.log.WithError(err).Errorf("failed to bootstrap node.. will retry in %s", sleepFor.String())

				time.Sleep(sleepFor)

				continue
			}

			n.client = client

			httpClient := http.Client{
				Timeout: timeout,
			}

			n.api = api.NewConsensusClient(ctx, n.log, n.config.Addr, httpClient, n.config.Headers)

			break
		}
	}

	return nil
}
