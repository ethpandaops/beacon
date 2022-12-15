package beacon

import (
	"context"
	"net/http"
	"time"

	eth2client "github.com/attestantio/go-eth2-client"
	ehttp "github.com/attestantio/go-eth2-client/http"
	"github.com/rs/zerolog"
	"github.com/samcm/beacon/pkg/beacon/api"
)

// ensureClients ensures that the node has a client and an API client.
func (n *node) ensureClients(ctx context.Context) error {
	failures := 0

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
			timeout := 90 * time.Second

			client, err := ehttp.New(ctx,
				ehttp.WithAddress(n.config.Addr),
				ehttp.WithLogLevel(zerolog.Disabled),
				ehttp.WithTimeout(timeout),
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

			n.api = api.NewConsensusClient(ctx, n.log, n.config.Addr, httpClient)

			break
		}
	}

	return nil
}
