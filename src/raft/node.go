package raft

import (
    "sync"
    "time"
    "math/rand"
    "context"
    "errors"
    "github.com/prometheus/client_golang/prometheus"
)

type NodeState int

const (
    Follower NodeState = iota
    Candidate
    Leader
)

// Add peers field to RaftNode struct
type RaftNode struct {
    mu sync.Mutex
    
    nodeID string
    peers []string    // Add this field
    state NodeState
    currentTerm uint64
    votedFor string
    
    // Log entries
    log []LogEntry
    commitIndex uint64
    lastApplied uint64
    
    // Leader specific
    nextIndex map[string]uint64
    matchIndex map[string]uint64
    
    // Election timer
    electionTimeout time.Duration
    lastHeartbeat time.Time
    storage *Storage
    lastSnapshot *Snapshot
    snapshotIndex uint64
}

// Add these methods after the existing ones

func (n *RaftNode) CreateSnapshot() error {
    n.mu.Lock()
    defer n.mu.Unlock()

    if len(n.log) == 0 {
        return nil
    }

    snapshot := &Snapshot{
        LastIncludedIndex: n.commitIndex,
        LastIncludedTerm:  n.log[n.commitIndex].Term,
        State:            nil, // State machine state would go here
    }

    // Truncate log
    n.log = n.log[n.commitIndex:]
    n.lastSnapshot = snapshot
    n.snapshotIndex = n.commitIndex

    return n.storage.SaveSnapshot(snapshot)
}

func (n *RaftNode) InstallSnapshot(snapshot *Snapshot) error {
    n.mu.Lock()
    defer n.mu.Unlock()

    if snapshot.LastIncludedIndex <= n.snapshotIndex {
        return nil
    }

    // Clear old log entries
    n.log = make([]LogEntry, 0)
    n.lastSnapshot = snapshot
    n.snapshotIndex = snapshot.LastIncludedIndex
    n.commitIndex = snapshot.LastIncludedIndex
    n.lastApplied = snapshot.LastIncludedIndex

    return nil
}

type LogEntry struct {
    Term uint64
    Index uint64
    Command []byte
}

// Update NewRaftNode to store peers
func NewRaftNode(nodeID string, peers []string) *RaftNode {
    return &RaftNode{
        nodeID: nodeID,
        peers: peers,    // Add this line
        state: Follower,
        currentTerm: 0,
        votedFor: "",
        log: make([]LogEntry, 0),
        nextIndex: make(map[string]uint64),
        matchIndex: make(map[string]uint64),
        electionTimeout: time.Duration(150+rand.Intn(150)) * time.Millisecond,
    }
}

// Add these new methods after the existing ones

// Add these fields to RaftNode struct
type RaftNode struct {
    mu sync.Mutex
    
    nodeID string
    peers []string    // Add this field
    state NodeState
    currentTerm uint64
    votedFor string
    
    // Log entries
    log []LogEntry
    commitIndex uint64
    lastApplied uint64
    
    // Leader specific
    nextIndex map[string]uint64
    matchIndex map[string]uint64
    
    // Election timer
    electionTimeout time.Duration
    lastHeartbeat time.Time
    storage *Storage
    lastSnapshot *Snapshot
    snapshotIndex uint64
    
    // Connection management
    reconnectTicker *time.Ticker
    disconnectedPeers map[string]time.Time
}

// Update NewRaftNode constructor
func NewRaftNode(config *NodeConfig) *RaftNode {
    node := &RaftNode{
        nodeID:           config.NodeID,
        peers:            make([]string, 0, len(config.PeerAddresses)),
        state:           Follower,
        currentTerm:     0,
        votedFor:        "",
        log:             make([]LogEntry, 0),
        nextIndex:       make(map[string]uint64),
        matchIndex:      make(map[string]uint64),
        config:          config,
        rpcClient:       NewRaftClient(),
        stopCh:          make(chan struct{}),
        disconnectedPeers: make(map[string]time.Time),
    }

    for peerID := range config.PeerAddresses {
        node.peers = append(node.peers, peerID)
    }

    go node.managePeerConnections()
    return node
}

func (n *RaftNode) managePeerConnections() {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-n.stopCh:
            return
        case <-ticker.C:
            n.reconnectFailedPeers()
        }
    }
}

func (n *RaftNode) reconnectFailedPeers() {
    n.mu.Lock()
    peers := make(map[string]time.Time, len(n.disconnectedPeers))
    for peerID, lastAttempt := range n.disconnectedPeers {
        peers[peerID] = lastAttempt
    }
    n.mu.Unlock()

    for peerID, lastAttempt := range peers {
        if time.Since(lastAttempt) < time.Duration(n.config.ConnectTimeout)*time.Millisecond {
            continue
        }

        if err := n.connectToPeer(peerID); err != nil {
            n.mu.Lock()
            n.disconnectedPeers[peerID] = time.Now()
            n.mu.Unlock()
        } else {
            n.mu.Lock()
            delete(n.disconnectedPeers, peerID)
            n.mu.Unlock()
        }
    }
}

func (n *RaftNode) connectToPeer(peerID string) error {
    address, ok := n.config.PeerAddresses[peerID]
    if !ok {
        return errors.New("unknown peer")
    }

    ctx, cancel := context.WithTimeout(context.Background(), 
        time.Duration(n.config.ConnectTimeout)*time.Millisecond)
    defer cancel()

    var lastErr error
    for i := 0; i < n.config.MaxRetries; i++ {
        if err := n.rpcClient.Connect(peerID, address); err != nil {
            lastErr = err
            time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
            continue
        }
        return nil
    }
    return lastErr
}

// Update requestVote to use RPC
func (n *RaftNode) requestVote(peerID string, term uint64) bool {
    n.mu.Lock()
    lastLogIndex := uint64(len(n.log))
    lastLogTerm := uint64(0)
    if lastLogIndex > 0 {
        lastLogTerm = n.log[lastLogIndex-1].Term
    }
    n.mu.Unlock()

    req := &pb.RequestVoteRequest{
        Term:         term,
        CandidateId:  n.nodeID,
        LastLogIndex: lastLogIndex,
        LastLogTerm:  lastLogTerm,
    }

    ctx, cancel := context.WithTimeout(context.Background(), 
        time.Duration(n.config.ConnectTimeout)*time.Millisecond)
    defer cancel()

    resp, err := n.rpcClient.RequestVote(ctx, peerID, req)
    if err != nil {
        n.mu.Lock()
        n.disconnectedPeers[peerID] = time.Now()
        n.mu.Unlock()
        return false
    }

    return resp.VoteGranted
}

func (n *RaftNode) sendHeartbeats() {
    n.mu.Lock()
    currentTerm := n.currentTerm
    entries := make([]LogEntry, 0)
    n.mu.Unlock()

    for _, peer := range n.peers {
        go func(peerID string) {
            n.mu.Lock()
            nextIdx := n.nextIndex[peerID]
            if nextIdx < uint64(len(n.log)) {
                entries = n.log[nextIdx:]
            }
            n.mu.Unlock()

            ctx := context.Background()
            if err := n.replicateLog(ctx, peerID, currentTerm, entries); err != nil {
                // Handle error (peer might be down)
                return
            }
        }(peer)
    }
}

// Add metrics fields to RaftNode
type RaftNode struct {
    mu sync.Mutex
    
    nodeID string
    peers []string    // Add this field
    state NodeState
    currentTerm uint64
    votedFor string
    
    // Log entries
    log []LogEntry
    commitIndex uint64
    lastApplied uint64
    
    // Leader specific
    nextIndex map[string]uint64
    matchIndex map[string]uint64
    
    // Election timer
    electionTimeout time.Duration
    lastHeartbeat time.Time
    storage *Storage
    lastSnapshot *Snapshot
    snapshotIndex uint64
    
    // Connection management
    reconnectTicker *time.Ticker
    disconnectedPeers map[string]time.Time
}

// Update NewRaftNode constructor
func NewRaftNode(config *NodeConfig) (*RaftNode, error) {
    storage, err := NewStorage(filepath.Join(config.DataDir, "raft"))
    if err != nil {
        return nil, err
    }

    node := &RaftNode{
        nodeID:           config.NodeID,
        peers:            make([]string, 0, len(config.PeerAddresses)),
        state:           Follower,
        currentTerm:     0,
        votedFor:        "",
        log:             make([]LogEntry, 0),
        nextIndex:       make(map[string]uint64),
        matchIndex:      make(map[string]uint64),
        config:          config,
        rpcClient:       NewRaftClient(),
        stopCh:          make(chan struct{}),
        disconnectedPeers: make(map[string]time.Time),
        storage:         storage,
        healthCheck:     NewHealthChecker(3),
        metrics:        NewRaftMetrics(config.NodeID),
    }

    // Load persistent state
    state, err := storage.LoadState()
    if err != nil {
        return nil, err
    }
    
    node.currentTerm = state.CurrentTerm
    node.votedFor = state.VotedFor
    node.log = state.Log

    // Load snapshot if exists
    snapshot, err := storage.LoadSnapshot()
    if err != nil {
        return nil, err
    }
    if snapshot != nil {
        node.lastSnapshot = snapshot
        node.snapshotIndex = snapshot.LastIncludedIndex
    }

    for peerID := range config.PeerAddresses {
        node.peers = append(node.peers, peerID)
    }

    // Start background tasks
    go node.managePeerConnections()
    go node.startHealthCheck()
    go node.startLogCompaction()

    return node, nil
}

func (n *RaftNode) managePeerConnections() {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-n.stopCh:
            return
        case <-ticker.C:
            n.reconnectFailedPeers()
        }
    }
}

func (n *RaftNode) reconnectFailedPeers() {
    n.mu.Lock()
    peers := make(map[string]time.Time, len(n.disconnectedPeers))
    for peerID, lastAttempt := range n.disconnectedPeers {
        peers[peerID] = lastAttempt
    }
    n.mu.Unlock()

    for peerID, lastAttempt := range peers {
        if time.Since(lastAttempt) < time.Duration(n.config.ConnectTimeout)*time.Millisecond {
            continue
        }

        if err := n.connectToPeer(peerID); err != nil {
            n.mu.Lock()
            n.disconnectedPeers[peerID] = time.Now()
            n.mu.Unlock()
        } else {
            n.mu.Lock()
            delete(n.disconnectedPeers, peerID)
            n.mu.Unlock()
        }
    }
}

func (n *RaftNode) connectToPeer(peerID string) error {
    address, ok := n.config.PeerAddresses[peerID]
    if !ok {
        return errors.New("unknown peer")
    }

    ctx, cancel := context.WithTimeout(context.Background(), 
        time.Duration(n.config.ConnectTimeout)*time.Millisecond)
    defer cancel()

    var lastErr error
    for i := 0; i < n.config.MaxRetries; i++ {
        if err := n.rpcClient.Connect(peerID, address); err != nil {
            lastErr = err
            time.Sleep(time.Duration(100*(i+1)) * time.Millisecond)
            continue
        }
        return nil
    }
    return lastErr
}

// Update requestVote to use RPC
func (n *RaftNode) requestVote(peerID string, term uint64) bool {
    n.mu.Lock()
    lastLogIndex := uint64(len(n.log))
    lastLogTerm := uint64(0)
    if lastLogIndex > 0 {
        lastLogTerm = n.log[lastLogIndex-1].Term
    }
    n.mu.Unlock()

    req := &pb.RequestVoteRequest{
        Term:         term,
        CandidateId:  n.nodeID,
        LastLogIndex: lastLogIndex,
        LastLogTerm:  lastLogTerm,
    }

    ctx, cancel := context.WithTimeout(context.Background(), 
        time.Duration(n.config.ConnectTimeout)*time.Millisecond)
    defer cancel()

    resp, err := n.rpcClient.RequestVote(ctx, peerID, req)
    if err != nil {
        n.mu.Lock()
        n.disconnectedPeers[peerID] = time.Now()
        n.mu.Unlock()
        return false
    }

    return resp.VoteGranted
}

func (n *RaftNode) sendHeartbeats() {
    n.mu.Lock()
    currentTerm := n.currentTerm
    entries := make([]LogEntry, 0)
    n.mu.Unlock()

    for _, peer := range n.peers {
        go func(peerID string) {
            n.mu.Lock()
            nextIdx := n.nextIndex[peerID]
            if nextIdx < uint64(len(n.log)) {
                entries = n.log[nextIdx:]
            }
            n.mu.Unlock()

            ctx := context.Background()
            if err := n.replicateLog(ctx, peerID, currentTerm, entries); err != nil {
                // Handle error (peer might be down)
                return
            }
        }(peer)
    }
}

func (n *RaftNode) startLogCompaction() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-n.shutdownCh:
            return
        case <-ticker.C:
            if len(n.log) > 1000 { // Configurable threshold
                if err := n.CreateSnapshot(); err != nil {
                    // Handle error, maybe retry
                    continue
                }
            }
        }
    }
}

func (n *RaftNode) replicateLog(ctx context.Context, peerID string, term uint64, entries []LogEntry) error {
    n.mu.Lock()
    prevLogIndex := n.nextIndex[peerID] - 1
    prevLogTerm := uint64(0)
    if prevLogIndex > 0 && int(prevLogIndex) < len(n.log) {
        prevLogTerm = n.log[prevLogIndex].Term
    }
    
    pbEntries := make([]*pb.LogEntry, len(entries))
    for i, entry := range entries {
        pbEntries[i] = &pb.LogEntry{
            Term:    entry.Term,
            Index:   entry.Index,
            Command: entry.Command,
        }
    }
    n.mu.Unlock()

    req := &pb.AppendEntriesRequest{
        Term:         term,
        LeaderId:     n.nodeID,
        PrevLogIndex: prevLogIndex,
        PrevLogTerm:  prevLogTerm,
        Entries:      pbEntries,
        LeaderCommit: n.commitIndex,
    }

    start := time.Now()
    resp, err := n.rpcClient.AppendEntries(ctx, peerID, req)
    n.metrics.appendEntriesLatency.Observe(time.Since(start).Seconds())

    if err != nil {
        n.metrics.appendEntriesFailures.Inc()
        n.healthCheck.RecordFailure(peerID)
        return err
    }

    n.mu.Lock()
    if resp.Success {
        if len(entries) > 0 {
            n.nextIndex[peerID] += uint64(len(entries))
            n.matchIndex[peerID] = n.nextIndex[peerID] - 1
            n.metrics.replicatedEntries.Add(float64(len(entries)))
        }
    } else {
        // If AppendEntries fails, decrement nextIndex and retry
        if n.nextIndex[peerID] > 0 {
            n.nextIndex[peerID]--
        }
    }
    n.mu.Unlock()

    return nil
}

// Add health checking
type HealthChecker struct {
    mu sync.Mutex
    failures map[string]int
    lastSeen map[string]time.Time
    threshold int
}

func NewHealthChecker(threshold int) *HealthChecker {
    return &HealthChecker{
        failures:  make(map[string]int),
        lastSeen:  make(map[string]time.Time),
        threshold: threshold,
    }
}

func (h *HealthChecker) RecordFailure(peerID string) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.failures[peerID]++
    h.lastSeen[peerID] = time.Now()
}

func (h *HealthChecker) RecordSuccess(peerID string) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.failures[peerID] = 0
    h.lastSeen[peerID] = time.Now()
}

func (h *HealthChecker) IsHealthy(peerID string) bool {
    h.mu.Lock()
    defer h.mu.Unlock()
    return h.failures[peerID] < h.threshold
}

// Add metrics
type RaftMetrics struct {
    appendEntriesLatency prometheus.Histogram
    appendEntriesFailures prometheus.Counter
    replicatedEntries prometheus.Counter
    leadershipTransitions prometheus.Counter
}

func NewRaftMetrics(nodeID string) *RaftMetrics {
    return &RaftMetrics{
        appendEntriesLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
            Name: "raft_append_entries_latency",
            Help: "Latency of AppendEntries RPC calls",
        }),
        appendEntriesFailures: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "raft_append_entries_failures",
            Help: "Number of failed AppendEntries RPC calls",
        }),
        replicatedEntries: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "raft_replicated_entries",
            Help: "Number of successfully replicated log entries",
        }),
        leadershipTransitions: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "raft_leadership_transitions",
            Help: "Number of leadership transitions",
        }),
    }
}

// Add graceful shutdown
func (n *RaftNode) Shutdown() error {
    n.mu.Lock()
    if n.shutdown {
        n.mu.Unlock()
        return nil
    }
    n.shutdown = true
    n.mu.Unlock()

    close(n.shutdownCh)
    
    // Wait for all goroutines to finish
    n.rpcClient.Close()
    
    // Save state before shutdown
    if err := n.storage.SaveState(&PersistentState{
        CurrentTerm: n.currentTerm,
        VotedFor:    n.votedFor,
        Log:         n.log,
    }); err != nil {
        return err
    }
    
    return nil
}

func (n *RaftNode) HandleVoteRequest(ctx context.Context, candidateID string, term uint64, lastLogIndex uint64, lastLogTerm uint64) bool {
    n.mu.Lock()
    defer n.mu.Unlock()

    if term < n.currentTerm {
        return false
    }

    if term > n.currentTerm {
        n.currentTerm = term
        n.state = Follower
        n.votedFor = ""
    }

    if n.votedFor != "" && n.votedFor != candidateID {
        return false
    }

    // Check if candidate's log is at least as up-to-date as ours
    ourLastIndex := uint64(len(n.log))
    ourLastTerm := uint64(0)
    if ourLastIndex > 0 {
        ourLastTerm = n.log[ourLastIndex-1].Term
    }

    if lastLogTerm < ourLastTerm || 
       (lastLogTerm == ourLastTerm && lastLogIndex < ourLastIndex) {
        return false
    }

    n.votedFor = candidateID
    return true
}

// StartElection initiates the leader election process
func (n *RaftNode) StartElection() error {
    n.mu.Lock()
    n.state = Candidate
    n.currentTerm++
    n.votedFor = n.nodeID
    currentTerm := n.currentTerm
    n.mu.Unlock()

    votes := 1  // Vote for self
    voteCh := make(chan bool)

    // Request votes from all peers
    for _, peer := range n.peers {
        go func(peerID string) {
            granted := n.requestVote(peerID, currentTerm)
            voteCh <- granted
        }(peer)
    }

    // Count votes
    for i := 0; i < len(n.peers); i++ {
        if <-voteCh {
            votes++
            if votes > len(n.peers)/2 {
                n.becomeLeader()
                return nil
            }
        }
    }
    
    n.mu.Lock()
    n.state = Follower
    n.mu.Unlock()
    return errors.New("failed to get majority votes")
}

func (n *RaftNode) becomeLeader() {
    n.mu.Lock()
    defer n.mu.Unlock()
    
    n.state = Leader
    for _, peer := range n.peers {
        n.nextIndex[peer] = uint64(len(n.log))
        n.matchIndex[peer] = 0
    }
    
    // Start sending heartbeats
    go n.heartbeatLoop()
}

func (n *RaftNode) heartbeatLoop() {
    ticker := time.NewTicker(50 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            if n.state != Leader {
                return
            }
            n.sendHeartbeats()
        }
    }
}

func (n *RaftNode) AppendEntries(ctx context.Context, term uint64, entries []LogEntry) error {
    n.mu.Lock()
    defer n.mu.Unlock()

    if term < n.currentTerm {
        return errors.New("term is outdated")
    }

    if term > n.currentTerm {
        n.currentTerm = term
        n.state = Follower
        n.votedFor = ""
    }

    n.lastHeartbeat = time.Now()

    // Append new entries
    n.log = append(n.log, entries...)
    return nil
}