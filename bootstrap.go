package beacon

import (
	"context"
	"time"

	eth2client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/http"
	"github.com/rs/zerolog"
	"github.com/samcm/beacon/api"
)

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
			client, err := http.New(ctx,
				http.WithAddress(n.config.Addr),
				http.WithLogLevel(zerolog.Disabled),
			)
			if err != nil {
				failures++

				sleepFor := time.Duration(failures) * (time.Second * 5)

				// Clamp the sleep time to a maximum of 5 minutes.
				if sleepFor > time.Minute*5 {
					sleepFor = time.Minute * 5
				}

				n.log.WithError(err).Error("failed to bootstrap node.. will retry in %s", sleepFor.String())

				time.Sleep(sleepFor)

				continue
			}

			n.client = client
			n.api = api.NewConsensusClient(ctx, n.log, n.config.Addr)

			break
		}
	}

	return nil
}
