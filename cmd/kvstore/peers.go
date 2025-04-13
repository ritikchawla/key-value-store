package main

import (
    "encoding/json"
    "os"
)

func loadPeers(filename string, config *raft.NodeConfig) error {
    data, err := os.ReadFile(filename)
    if err != nil {
        return err
    }

    return json.Unmarshal(data, &config.PeerAddresses)
}