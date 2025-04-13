package store

import (
    "context"
    "time"
)

type Store interface {
    // Put stores a key-value pair
    Put(ctx context.Context, key []byte, value []byte) error
    
    // Get retrieves the value for a given key
    Get(ctx context.Context, key []byte) ([]byte, error)
    
    // Delete removes a key-value pair
    Delete(ctx context.Context, key []byte) error
}

type VectorClock struct {
    NodeID string
    Counter uint64
    Timestamp time.Time
}

type StoreOptions struct {
    NodeID string
    RaftPort int
    StorageDir string
    PeerNodes []string
}