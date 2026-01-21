package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ethpandaops/beacon/pkg/beacon/api/types"
	"github.com/ethpandaops/beacon/pkg/beacon/api/types/lightclient"
	"github.com/sirupsen/logrus"
)

// ConsensusClient is an interface for executing RPC calls to the Ethereum node.
type ConsensusClient interface {
	NodePeer(ctx context.Context, peerID string) (types.Peer, error)
	NodePeers(ctx context.Context) (types.Peers, error)
	NodePeerCount(ctx context.Context) (types.PeerCount, error)
	RawBlock(ctx context.Context, stateID string, contentType string) ([]byte, error)
	RawDebugBeaconState(ctx context.Context, stateID string, contentType string) ([]byte, error)
	DepositSnapshot(ctx context.Context) (*types.DepositSnapshot, error)
	NodeIdentity(ctx context.Context) (*types.Identity, error)
	LightClientBootstrap(ctx context.Context, blockRoot string) (*LightClientBootstrapResponse, error)
	LightClientUpdates(ctx context.Context, startPeriod, count int) (*LightClientUpdatesResponse, error)
	LightClientFinalityUpdate(ctx context.Context) (*LightClientFinalityUpdateResponse, error)
	LightClientOptimisticUpdate(ctx context.Context) (*LightClientOptimisticUpdateResponse, error)
}

type consensusClient struct {
	url     string
	log     logrus.FieldLogger
	client  http.Client
	headers map[string]string
}

// NewConsensusClient creates a new ConsensusClient.
func NewConsensusClient(ctx context.Context, log logrus.FieldLogger, url string, client http.Client, headers map[string]string) ConsensusClient {
	return &consensusClient{
		url:     url,
		log:     log,
		client:  client,
		headers: headers,
	}
}

type BeaconAPIResponse struct {
	Data    json.RawMessage `json:"data"`
	Version string          `json:"version"`
}

type BeaconAPIResponses[T any] []BeaconAPIResponse

//nolint:unused // this is used in the future
func (c *consensusClient) post(ctx context.Context, path string, body map[string]interface{}) (*BeaconAPIResponse, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+path, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	// Set headers from c.headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	rsp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", rsp.StatusCode)
	}

	data, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	resp := new(BeaconAPIResponse)
	if err := json.Unmarshal(data, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

//nolint:unparam // ctx will probably be used in the future
func (c *consensusClient) get(ctx context.Context, path string, contentType string, rspType any) error {
	if contentType == "" {
		contentType = "application/json"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.url+path, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", contentType)

	// Set headers from c.headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	rsp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d", rsp.StatusCode)
	}

	// Parse the content type header to handle parameters like charset
	contentTypeHeader := rsp.Header.Get("Content-Type")
	if contentTypeHeader != "" {
		if !strings.Contains(contentTypeHeader, contentType) {
			return fmt.Errorf("unexpected content type: wanted (%s): got (%s)", contentType, contentTypeHeader)
		}
	}

	data, err := io.ReadAll(rsp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, rspType); err != nil {
		return err
	}

	return nil
}

func (c *consensusClient) getRaw(ctx context.Context, path string, contentType string) ([]byte, error) {
	if contentType == "" {
		contentType = "application/json"
	}

	u, err := url.Parse(c.url + path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	// Set headers from c.headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	req.Header.Set("Accept", contentType)

	rsp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", rsp.StatusCode)
	}

	return io.ReadAll(rsp.Body)
}

// NodePeers returns the list of peers connected to the node.
func (c *consensusClient) NodePeers(ctx context.Context) (types.Peers, error) {
	data := new(BeaconAPIResponse)
	if err := c.get(ctx, "/eth/v1/node/peers", ContentTypeJSON, data); err != nil {
		return nil, err
	}

	rsp := types.Peers{}
	if err := json.Unmarshal(data.Data, &rsp); err != nil {
		return nil, err
	}

	return rsp, nil
}

// NodePeer returns the peer with the given peer ID.
func (c *consensusClient) NodePeer(ctx context.Context, peerID string) (types.Peer, error) {
	data := new(BeaconAPIResponse)
	if err := c.get(ctx, fmt.Sprintf("/eth/v1/node/peers/%s", peerID), ContentTypeJSON, data); err != nil {
		return types.Peer{}, err
	}

	rsp := types.Peer{}
	if err := json.Unmarshal(data.Data, &rsp); err != nil {
		return types.Peer{}, err
	}

	return rsp, nil
}

// NodePeerCount returns the number of peers connected to the node.
func (c *consensusClient) NodePeerCount(ctx context.Context) (types.PeerCount, error) {
	data := new(BeaconAPIResponse)
	if err := c.get(ctx, "/eth/v1/node/peer_count", ContentTypeJSON, data); err != nil {
		return types.PeerCount{}, err
	}

	rsp := types.PeerCount{}
	if err := json.Unmarshal(data.Data, &rsp); err != nil {
		return types.PeerCount{}, err
	}

	return rsp, nil
}

// RawDebugBeaconState returns the beacon state in the requested format.
func (c *consensusClient) RawDebugBeaconState(ctx context.Context, stateID string, contentType string) ([]byte, error) {
	data, err := c.getRaw(ctx, fmt.Sprintf("/eth/v2/debug/beacon/states/%s", stateID), contentType)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// RawBlock returns the block in the requested format.
func (c *consensusClient) RawBlock(ctx context.Context, stateID string, contentType string) ([]byte, error) {
	data, err := c.getRaw(ctx, fmt.Sprintf("/eth/v2/beacon/blocks/%s", stateID), contentType)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// DepositSnapshot returns the deposit snapshot in the requested format.
func (c *consensusClient) DepositSnapshot(ctx context.Context) (*types.DepositSnapshot, error) {
	data := new(BeaconAPIResponse)
	if err := c.get(ctx, "/eth/v1/beacon/deposit_snapshot", ContentTypeJSON, data); err != nil {
		return nil, err
	}

	rsp := types.DepositSnapshot{}
	if err := json.Unmarshal(data.Data, &rsp); err != nil {
		return nil, err
	}

	return &rsp, nil
}

func (c *consensusClient) NodeIdentity(ctx context.Context) (*types.Identity, error) {
	data := new(BeaconAPIResponse)
	if err := c.get(ctx, "/eth/v1/node/identity", ContentTypeJSON, data); err != nil {
		return nil, err
	}

	rsp := types.Identity{}
	if err := json.Unmarshal(data.Data, &rsp); err != nil {
		return nil, err
	}

	return &rsp, nil
}

func (c *consensusClient) LightClientBootstrap(ctx context.Context, blockRoot string) (*LightClientBootstrapResponse, error) {
	data := new(BeaconAPIResponse)
	if err := c.get(ctx, fmt.Sprintf("/eth/v1/beacon/light_client/bootstrap/%s", blockRoot), ContentTypeJSON, data); err != nil {
		return nil, err
	}

	rsp := LightClientBootstrapResponse{
		Response: Response[*lightclient.Bootstrap]{
			Data: &lightclient.Bootstrap{},
			Metadata: map[string]any{
				"version": data.Version,
			},
		},
	}
	if err := json.Unmarshal(data.Data, &rsp.Response.Data); err != nil {
		return nil, err
	}

	return &rsp, nil
}

func (c *consensusClient) LightClientUpdates(ctx context.Context, startPeriod, count int) (*LightClientUpdatesResponse, error) {
	if count == 0 {
		return nil, errors.New("count must be greater than 0")
	}

	params := url.Values{}
	params.Add("start_period", fmt.Sprintf("%d", startPeriod))
	params.Add("count", fmt.Sprintf("%d", count))

	data := new(BeaconAPIResponses[*lightclient.Updates])
	if err := c.get(ctx, "/eth/v1/beacon/light_client/updates?"+params.Encode(), ContentTypeJSON, data); err != nil {
		return nil, err
	}

	rsp := LightClientUpdatesResponse{
		Response: Response[*lightclient.Updates]{
			Data:     &lightclient.Updates{},
			Metadata: map[string]any{},
		},
	}

	updates := make(lightclient.Updates, 0)
	for _, resp := range *data {
		update := lightclient.Update{}
		if err := json.Unmarshal(resp.Data, &update); err != nil {
			return nil, err
		}

		updates = append(updates, &update)
	}

	rsp.Response.Data = &updates

	return &rsp, nil
}

func (c *consensusClient) LightClientFinalityUpdate(ctx context.Context) (*LightClientFinalityUpdateResponse, error) {
	data := new(BeaconAPIResponse)
	if err := c.get(ctx, "/eth/v1/beacon/light_client/finality_update", ContentTypeJSON, data); err != nil {
		return nil, err
	}

	rsp := LightClientFinalityUpdateResponse{
		Response: Response[*lightclient.FinalityUpdate]{
			Data: &lightclient.FinalityUpdate{},
			Metadata: map[string]any{
				"version": data.Version,
			},
		},
	}
	if err := json.Unmarshal(data.Data, &rsp.Data); err != nil {
		return nil, err
	}

	return &rsp, nil
}

func (c *consensusClient) LightClientOptimisticUpdate(ctx context.Context) (*LightClientOptimisticUpdateResponse, error) {
	data := new(BeaconAPIResponse)
	if err := c.get(ctx, "/eth/v1/beacon/light_client/optimistic_update", ContentTypeJSON, data); err != nil {
		return nil, err
	}

	rsp := LightClientOptimisticUpdateResponse{
		Response: Response[*lightclient.OptimisticUpdate]{
			Data: &lightclient.OptimisticUpdate{},
			Metadata: map[string]any{
				"version": data.Version,
			},
		},
	}
	if err := json.Unmarshal(data.Data, &rsp.Data); err != nil {
		return nil, err
	}

	return &rsp, nil
}
