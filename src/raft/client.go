package raft

import (
    "context"
    "errors"
)

type ClientRequest struct {
    Command []byte
}

type ClientResponse struct {
    Success bool
    Error   error
}

func (n *RaftNode) ProcessClientRequest(ctx context.Context, req *ClientRequest) (*ClientResponse, error) {
    if n.state != Leader {
        return &ClientResponse{
            Success: false,
            Error:   errors.New("not leader"),
        }, nil
    }

    entry := LogEntry{
        Term:    n.currentTerm,
        Index:   uint64(len(n.log)),
        Command: req.Command,
    }

    // Append to local log
    n.log = append(n.log, entry)

    // Wait for replication
    replicated := make(chan bool)
    go n.waitForReplication(entry.Index, replicated)

    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    case success := <-replicated:
        if !success {
            return &ClientResponse{Success: false, Error: errors.New("replication failed")}, nil
        }
        return &ClientResponse{Success: true}, nil
    }
}

func (n *RaftNode) waitForReplication(index uint64, done chan bool) {
    // Wait for majority of nodes to replicate
    majority := len(n.peers)/2 + 1
    ticker := time.NewTicker(50 * time.Millisecond)
    defer ticker.Stop()

    for {
        <-ticker.C
        count := 1 // Count self
        for _, peer := range n.peers {
            if n.matchIndex[peer] >= index {
                count++
            }
        }
        if count >= majority {
            done <- true
            return
        }
    }
}