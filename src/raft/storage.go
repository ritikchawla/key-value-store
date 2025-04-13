package raft

import (
    "encoding/json"
    "os"
    "path/filepath"
)

type Storage struct {
    dir string
    snapshot *Snapshot
}

type Snapshot struct {
    LastIncludedIndex uint64
    LastIncludedTerm  uint64
    State            []byte
}

type PersistentState struct {
    CurrentTerm uint64
    VotedFor    string
    Log         []LogEntry
}

func NewStorage(dir string) (*Storage, error) {
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, err
    }
    return &Storage{dir: dir}, nil
}

func (s *Storage) SaveState(state *PersistentState) error {
    data, err := json.Marshal(state)
    if err != nil {
        return err
    }
    return os.WriteFile(filepath.Join(s.dir, "state.json"), data, 0644)
}

func (s *Storage) LoadState() (*PersistentState, error) {
    data, err := os.ReadFile(filepath.Join(s.dir, "state.json"))
    if err != nil {
        if os.IsNotExist(err) {
            return &PersistentState{}, nil
        }
        return nil, err
    }
    
    var state PersistentState
    if err := json.Unmarshal(data, &state); err != nil {
        return nil, err
    }
    return &state, nil
}

// Add these methods to the Storage struct

func (s *Storage) SaveSnapshot(snapshot *Snapshot) error {
    data, err := json.Marshal(snapshot)
    if err != nil {
        return err
    }
    return os.WriteFile(filepath.Join(s.dir, "snapshot"), data, 0644)
}

func (s *Storage) LoadSnapshot() (*Snapshot, error) {
    data, err := os.ReadFile(filepath.Join(s.dir, "snapshot"))
    if err != nil {
        if os.IsNotExist(err) {
            return nil, nil
        }
        return nil, err
    }
    
    var snapshot Snapshot
    if err := json.Unmarshal(data, &snapshot); err != nil {
        return nil, err
    }
    return &snapshot, nil
}