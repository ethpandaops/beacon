package types

import (
	"strings"
)

// Agent is a peer's agent.
type Agent string

const (
	// AgentUnknown is an unknown agent.
	AgentUnknown Agent = "unknown"
	// AgentLighthouse is a Lighthouse agent.
	AgentLighthouse Agent = "lighthouse"
	// AgentNimbus is a Nimbus agent.
	AgentNimbus Agent = "nimbus"
	// AgentTeku is a Teku agent.
	AgentTeku Agent = "teku"
	// AgentPrysm is a Prysm agent.
	AgentPrysm Agent = "prysm"
	// AgentLodestar is a Lodestar agent.
	AgentLodestar Agent = "lodestar"
	// AgentGrandine is a Grandine agent.
	AgentGrandine Agent = "grandine"
)

// AllAgents is a list of all agents.
var AllAgents = []Agent{
	AgentUnknown,
	AgentLighthouse,
	AgentNimbus,
	AgentTeku,
	AgentPrysm,
	AgentLodestar,
	AgentGrandine,
}

// AgentCount represents the number of peers with each agent.
type AgentCount struct {
	Unknown    int `json:"unknown"`
	Lighthouse int `json:"lighthouse"`
	Nimbus     int `json:"nimbus"`
	Teku       int `json:"teku"`
	Prysm      int `json:"prysm"`
	Lodestar   int `json:"lodestar"`
	Grandine   int `json:"grandine"`
}

// AgentFromString returns the agent from the given string.
func AgentFromString(agent string) Agent {
	asLower := strings.ToLower(agent)

	if strings.Contains(asLower, "lighthouse") {
		return AgentLighthouse
	}

	if strings.Contains(asLower, "nimbus") {
		return AgentNimbus
	}

	if strings.Contains(asLower, "teku") {
		return AgentTeku
	}

	if strings.Contains(asLower, "prysm") {
		return AgentPrysm
	}

	if strings.Contains(asLower, "lodestar") {
		return AgentLodestar
	}

	if strings.Contains(asLower, "grandine") {
		return AgentGrandine
	}

	return AgentUnknown
}
