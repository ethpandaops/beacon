package api

import "github.com/ethpandaops/beacon/pkg/beacon/api/types/lightclient"

type Response[T any] struct {
	Data     T              `json:"data"`
	Metadata map[string]any `json:"metadata"`
}

type LightClientUpdatesResponse struct {
	Response[*lightclient.Updates]
}

type LightClientBootstrapResponse struct {
	Response[*lightclient.Bootstrap]
}

type LightClientFinalityUpdateResponse struct {
	Response[*lightclient.FinalityUpdate]
}

type LightClientOptimisticUpdateResponse struct {
	Response[*lightclient.OptimisticUpdate]
}
