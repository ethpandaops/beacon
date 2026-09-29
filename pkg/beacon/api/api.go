package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/ethpandaops/beacon/pkg/beacon/api/types"
	"github.com/sirupsen/logrus"
)

// ErrNotFound is returned when the node responds with HTTP 404 for the
// requested resource, e.g. an execution payload envelope that has not been
// revealed or a block that does not exist.
var ErrNotFound = errors.New("not found")

// ConsensusClient is an interface for executing RPC calls to the Ethereum node.
type ConsensusClient interface {
	NodePeer(ctx context.Context, peerID string) (types.Peer, error)
	NodePeers(ctx context.Context) (types.Peers, error)
	NodePeerCount(ctx context.Context) (types.PeerCount, error)
	RawBlock(ctx context.Context, blockID string, contentType string) ([]byte, error)
	RawExecutionPayloadEnvelope(ctx context.Context, blockID string, contentType string) ([]byte, error)
	RawDebugBeaconState(ctx context.Context, stateID string, contentType string) ([]byte, error)
	// OpenRawBlock opens a successful raw block response without consuming its body.
	OpenRawBlock(ctx context.Context, blockID string, contentType string) (*RawResponse, error)
	// OpenRawExecutionPayloadEnvelope opens a successful raw envelope response without consuming its body.
	OpenRawExecutionPayloadEnvelope(ctx context.Context, blockID string, contentType string) (*RawResponse, error)
	// OpenRawDebugBeaconState opens a successful raw beacon state response without consuming its body.
	OpenRawDebugBeaconState(ctx context.Context, stateID string, contentType string) (*RawResponse, error)
	DepositSnapshot(ctx context.Context) (*types.DepositSnapshot, error)
	NodeIdentity(ctx context.Context) (*types.Identity, error)
}

type consensusClient struct {
	url                 string
	log                 logrus.FieldLogger
	client              *http.Client
	rawClient           *http.Client
	headers             map[string]string
	rawResponseObserver RawResponseObserver
}

// NewConsensusClient creates a ConsensusClient. Both HTTP clients must be non-nil.
func NewConsensusClient(
	log logrus.FieldLogger,
	url string,
	client *http.Client,
	rawClient *http.Client,
	headers map[string]string,
	rawResponseObserver RawResponseObserver,
) ConsensusClient {
	if client == nil {
		panic("api: nil HTTP client")
	}

	if rawClient == nil {
		panic("api: nil raw HTTP client")
	}

	return &consensusClient{
		url:                 url,
		log:                 log,
		client:              client,
		rawClient:           rawClient,
		headers:             headers,
		rawResponseObserver: rawResponseObserver,
	}
}

type apiResponse struct {
	Data json.RawMessage `json:"data"`
}

//nolint:unused // this is used in the future
func (c *consensusClient) post(ctx context.Context, path string, body map[string]any) (json.RawMessage, error) {
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

	resp := new(apiResponse)
	if err := json.Unmarshal(data, resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

//nolint:unparam // ctx will probably be used in the future
func (c *consensusClient) get(ctx context.Context, path string) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.url+path, nil)
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

	resp := new(apiResponse)
	if err := json.Unmarshal(data, resp); err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func (c *consensusClient) getRaw(ctx context.Context, path string, contentType string) ([]byte, error) {
	rsp, err := c.doRaw(ctx, path, contentType)
	if err != nil {
		return nil, err
	}

	defer rsp.Body.Close()

	data, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// NodePeers returns the list of peers connected to the node.
func (c *consensusClient) NodePeers(ctx context.Context) (types.Peers, error) {
	data, err := c.get(ctx, "/eth/v1/node/peers")
	if err != nil {
		return nil, err
	}

	rsp := types.Peers{}
	if err := json.Unmarshal(data, &rsp); err != nil {
		return nil, err
	}

	return rsp, nil
}

// NodePeer returns the peer with the given peer ID.
func (c *consensusClient) NodePeer(ctx context.Context, peerID string) (types.Peer, error) {
	data, err := c.get(ctx, fmt.Sprintf("/eth/v1/node/peers/%s", peerID))
	if err != nil {
		return types.Peer{}, err
	}

	rsp := types.Peer{}
	if err := json.Unmarshal(data, &rsp); err != nil {
		return types.Peer{}, err
	}

	return rsp, nil
}

// NodePeerCount returns the number of peers connected to the node.
func (c *consensusClient) NodePeerCount(ctx context.Context) (types.PeerCount, error) {
	data, err := c.get(ctx, "/eth/v1/node/peer_count")
	if err != nil {
		return types.PeerCount{}, err
	}

	rsp := types.PeerCount{}
	if err := json.Unmarshal(data, &rsp); err != nil {
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
func (c *consensusClient) RawBlock(ctx context.Context, blockID string, contentType string) ([]byte, error) {
	data, err := c.getRaw(ctx, fmt.Sprintf("/eth/v2/beacon/blocks/%s", blockID), contentType)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// RawExecutionPayloadEnvelope returns the signed execution payload envelope
// for the given block id in the requested format (gloas onwards).
func (c *consensusClient) RawExecutionPayloadEnvelope(ctx context.Context, blockID string, contentType string) ([]byte, error) {
	data, err := c.getRaw(ctx, fmt.Sprintf("/eth/v1/beacon/execution_payload_envelopes/%s", blockID), contentType)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// DepositSnapshot returns the deposit snapshot in the requested format.
func (c *consensusClient) DepositSnapshot(ctx context.Context) (*types.DepositSnapshot, error) {
	data, err := c.get(ctx, "/eth/v1/beacon/deposit_snapshot")
	if err != nil {
		return nil, err
	}

	rsp := types.DepositSnapshot{}
	if err := json.Unmarshal(data, &rsp); err != nil {
		return nil, err
	}

	return &rsp, nil
}

func (c *consensusClient) NodeIdentity(ctx context.Context) (*types.Identity, error) {
	data, err := c.get(ctx, "/eth/v1/node/identity")
	if err != nil {
		return nil, err
	}

	rsp := types.Identity{}
	if err := json.Unmarshal(data, &rsp); err != nil {
		return nil, err
	}

	return &rsp, nil
}
