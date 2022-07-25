package beacon

import (
	"context"
	"errors"
	"fmt"
	"time"

	eth2client "github.com/attestantio/go-eth2-client"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

func (n *node) ensureBeaconSubscription(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second * 5):
			if len(n.config.EventTopics) == 0 {
				continue
			}

			if n.client == nil {
				continue
			}

			// Only resubscribe if we haven't received an event after 5 minutes.
			if time.Since(n.lastEventTime) < (5 * time.Minute) {
				continue
			}

			// Don't resubscribe if we are pre-genesis.
			if n.genesis == nil {
				continue
			}

			timeTillGenesis := time.Until(n.genesis.GenesisTime)
			if timeTillGenesis > 0 {
				n.log.WithField("time_until_genesis", timeTillGenesis.String()).Info("We are pre-genesis, not subscribing upstream yet...")

				time.Sleep(timeTillGenesis)
			}

			n.log.
				WithField("last_event_time", n.lastEventTime.Local().String()).
				Info("Haven't received any events for 5 minutes, re-subscribing")

			if time.Since(n.lastEventTime) > time.Minute*5 {
				if err := n.subscribeToBeaconEvents(ctx); err != nil {
					n.log.WithError(err).Error("Failed to subscribe to beacon")
				}

				time.Sleep(time.Minute * 15)
			}
		}
	}
}

func (n *node) subscribeToBeaconEvents(ctx context.Context) error {
	provider, isProvider := n.client.(eth2client.EventsProvider)
	if !isProvider {
		return errors.New("client does not implement eth2client.Subscriptions")
	}

	topics := []string{}

	for key, supported := range v1.SupportedEventTopics {
		if key == "contribution_and_proof" {
			continue
		}

		if !supported {
			continue
		}

		if !n.config.EventTopics.Exists(key) {
			continue
		}

		topics = append(topics, key)
	}

	n.log.WithField("topics", topics).Info("Subscribing to events upstream")

	if err := provider.Events(ctx, topics, func(event *v1.Event) {
		n.lastEventTime = time.Now()

		if err := n.handleEvent(ctx, event); err != nil {
			n.log.Errorf("Failed to handle event: %v", err)
		}
	}); err != nil {
		return err
	}

	return nil
}

func (n *node) handleEvent(ctx context.Context, event *v1.Event) error {
	n.publishEvent(ctx, event)

	// If we are syncing, only forward on "head" and "block" events
	// TODO(sam.calder-mason): Revisit this - was only added for the metrics-exporter use-case.
	if n.status.Syncing() {
		if event.Topic != topicBlock && event.Topic != topicHead {
			return nil
		}
	}

	switch event.Topic {
	case topicAttestation:
		return n.handleAttestation(ctx, event)
	case topicBlock:
		return n.handleBlock(ctx, event)
	case topicChainReorg:
		return n.handleChainReorg(ctx, event)
	case topicFinalizedCheckpoint:
		return n.handleFinalizedCheckpoint(ctx, event)
	case topicHead:
		return n.handleHead(ctx, event)
	case topicVoluntaryExit:
		return n.handleVoluntaryExit(ctx, event)
	case topicContributionAndProof:
		return n.handleContributionAndProof(ctx, event)

	default:
		return fmt.Errorf("unknown event topic %s", event.Topic)
	}
}

func (n *node) handleAttestation(ctx context.Context, event *v1.Event) error {
	attestation, valid := event.Data.(*phase0.Attestation)
	if !valid {
		return errors.New("invalid attestation event")
	}

	n.publishAttestation(ctx, attestation)

	return nil
}

func (n *node) handleBlock(ctx context.Context, event *v1.Event) error {
	block, valid := event.Data.(*v1.BlockEvent)
	if !valid {
		return errors.New("invalid block event")
	}

	n.publishBlock(ctx, block)

	return nil
}

func (n *node) handleChainReorg(ctx context.Context, event *v1.Event) error {
	chainReorg, valid := event.Data.(*v1.ChainReorgEvent)
	if !valid {
		return errors.New("invalid chain reorg event")
	}

	n.publishChainReOrg(ctx, chainReorg)

	return nil
}

func (n *node) handleFinalizedCheckpoint(ctx context.Context, event *v1.Event) error {
	checkpoint, valid := event.Data.(*v1.FinalizedCheckpointEvent)
	if !valid {
		return errors.New("invalid checkpoint event")
	}

	n.publishFinalizedCheckpoint(ctx, checkpoint)
	return nil
}

func (n *node) handleHead(ctx context.Context, event *v1.Event) error {
	head, valid := event.Data.(*v1.HeadEvent)
	if !valid {
		return errors.New("invalid head event")
	}

	n.publishHead(ctx, head)

	return nil
}

func (n *node) handleVoluntaryExit(ctx context.Context, event *v1.Event) error {
	exit, valid := event.Data.(*phase0.VoluntaryExit)
	if !valid {
		return errors.New("invalid voluntary exit event")
	}

	n.publishVoluntaryExit(ctx, exit)

	return nil
}

func (n *node) handleContributionAndProof(ctx context.Context, event *v1.Event) error {
	// Do nothing for now
	return nil
}
