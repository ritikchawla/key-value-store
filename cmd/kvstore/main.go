package main

import (
    "flag"
    "log"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/ritikchawla/key-value-store/src/raft"
)

func main() {
    nodeID := flag.String("id", "", "Node ID")
    listenAddr := flag.String("addr", ":7000", "Listen address")
    dataDir := flag.String("data", "data", "Data directory")
    peersFile := flag.String("peers", "peers.json", "Peers configuration file")
    flag.Parse()

    if *nodeID == "" {
        log.Fatal("Node ID is required")
    }

    config := &raft.NodeConfig{
        NodeID:        *nodeID,
        ListenAddress: *listenAddr,
        DataDir:      *dataDir,
        PeerAddresses: make(map[string]string),
        HeartbeatTimeout: 50,
        ElectionTimeout:  150,
        ConnectTimeout:   5000,
        MaxRetries:      3,
    }

    // Load peers from configuration
    if err := loadPeers(*peersFile, config); err != nil {
        log.Fatal(err)
    }

    node, err := raft.NewRaftNode(config)
    if err != nil {
        log.Fatal(err)
    }

    // Start the Raft server
    go func() {
        if err := raft.StartRaftServer(node, *listenAddr); err != nil {
            log.Fatal(err)
        }
    }()

    // Wait for interrupt signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh

    // Graceful shutdown
    if err := node.Shutdown(); err != nil {
        log.Printf("Error during shutdown: %v", err)
    }
}