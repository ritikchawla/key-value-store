package raft

import (
    "context"
    "io"
    "time"
)

type SnapshotTransfer struct {
    LastIncludedIndex uint64
    LastIncludedTerm  uint64
    Data              []byte
    Size              int64
}

func (n *RaftNode) transferSnapshot(peerID string) error {
    n.mu.Lock()
    snapshot := n.lastSnapshot
    if snapshot == nil {
        snapshot = &Snapshot{
            LastIncludedIndex: n.commitIndex,
            LastIncludedTerm:  n.currentTerm,
            State:            nil,
        }
    }
    n.mu.Unlock()

    transfer := &SnapshotTransfer{
        LastIncludedIndex: snapshot.LastIncludedIndex,
        LastIncludedTerm:  snapshot.LastIncludedTerm,
        Data:              snapshot.State,
        Size:              int64(len(snapshot.State)),
    }

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    n.metrics.snapshotSize.Observe(float64(transfer.Size))
    
    return n.rpcClient.InstallSnapshot(ctx, peerID, transfer)
}