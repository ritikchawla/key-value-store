package raft

type NodeConfig struct {
	NodeID        string
	ListenAddress string
	DataDir       string
	PeerAddresses map[string]string

	// Timeouts
	HeartbeatTimeout int
	ElectionTimeout  int
	ConnectTimeout   int
	MaxRetries       int
}

func DefaultConfig(nodeID string, listenAddr string) *NodeConfig {
	return &NodeConfig{
		NodeID:           nodeID,
		ListenAddress:    listenAddr,
		PeerAddresses:    make(map[string]string),
		HeartbeatTimeout: 50,   // milliseconds
		ElectionTimeout:  150,  // milliseconds
		ConnectTimeout:   5000, // milliseconds
		MaxRetries:       3,
	}
}
