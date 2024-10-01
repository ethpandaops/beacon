package types

import (
	"github.com/ethereum/go-ethereum/p2p/enode"
)

// Identity represents the node identity.
type Identity struct {
	PeerID             string   `json:"peer_id"`
	ENR                string   `json:"enr"`
	P2PAddresses       []string `json:"p2p_addresses"`
	DiscoveryAddresses []string `json:"discovery_addresses"`
	Metadata           struct {
		SeqNumber string `json:"seq_number"`
		Attnets   string `json:"attnets"`
		Syncnets  string `json:"syncnets"`
	} `json:"metadata"`
}

func (i *Identity) GetEnode() (*enode.Node, error) {
	var node enode.Node

	err := node.UnmarshalText([]byte(i.ENR))
	if err != nil {
		return nil, err
	}

	return &node, nil
}
