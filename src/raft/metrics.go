package raft

import (
    "github.com/prometheus/client_golang/prometheus"
)

type RaftMetrics struct {
    // Existing metrics
    appendEntriesLatency   prometheus.Histogram
    appendEntriesFailures  prometheus.Counter
    replicatedEntries     prometheus.Counter
    leadershipTransitions prometheus.Counter
    
    // New metrics
    termGauge             prometheus.Gauge
    stateGauge           prometheus.Gauge
    commitIndexGauge     prometheus.Gauge
    logSize              prometheus.Gauge
    healthyPeers         prometheus.Gauge
    snapshotSize         prometheus.Histogram
    recoveryTime         prometheus.Histogram
    quorumSize           prometheus.Gauge
}

func NewRaftMetrics(nodeID string) *RaftMetrics {
    return &RaftMetrics{
        // ... existing metrics initialization ...
        
        termGauge: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "raft_current_term",
            Help: "Current term of the Raft node",
        }),
        stateGauge: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "raft_node_state",
            Help: "Current state of the Raft node (0: Follower, 1: Candidate, 2: Leader)",
        }),
        healthyPeers: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "raft_healthy_peers",
            Help: "Number of healthy peers in the cluster",
        }),
        snapshotSize: prometheus.NewHistogram(prometheus.HistogramOpts{
            Name: "raft_snapshot_size_bytes",
            Help: "Size of Raft snapshots",
            Buckets: prometheus.ExponentialBuckets(1024, 2, 10),
        }),
        recoveryTime: prometheus.NewHistogram(prometheus.HistogramOpts{
            Name: "raft_recovery_duration_seconds",
            Help: "Time taken for node recovery",
        }),
    }
}