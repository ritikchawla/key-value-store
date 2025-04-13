package raft

import (
    "context"
    "time"
    "sync"
)

type HealthStatus struct {
    IsHealthy bool
    LastSeen time.Time
    TermBehind uint64
    NeedsSnapshot bool
}

func (n *RaftNode) startHealthCheck() {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-n.shutdownCh:
            return
        case <-ticker.C:
            n.checkPeersHealth()
        }
    }
}

func (n *RaftNode) checkPeersHealth() {
    healthyCount := 0
    for _, peer := range n.peers {
        if n.healthCheck.IsHealthy(peer) {
            healthyCount++
        } else {
            go n.recoverPeer(peer)
        }
    }
    n.metrics.healthyPeers.Set(float64(healthyCount))
}

func (n *RaftNode) recoverPeer(peerID string) {
    start := time.Now()
    defer func() {
        n.metrics.recoveryTime.Observe(time.Since(start).Seconds())
    }()

    status, err := n.getPeerStatus(peerID)
    if err != nil {
        return
    }

    if status.NeedsSnapshot {
        if err := n.transferSnapshot(peerID); err != nil {
            return
        }
    }

    if err := n.syncPeerLog(peerID, status); err != nil {
        return
    }
}