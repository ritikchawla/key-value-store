package raft

import (
    "context"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    pb "github.com/ritikchawla/key-value-store/proto"
    "sync"
)

type RaftClient struct {
    mu      sync.Mutex
    clients map[string]pb.RaftServiceClient
    conns   map[string]*grpc.ClientConn
}

func NewRaftClient() *RaftClient {
    return &RaftClient{
        clients: make(map[string]pb.RaftServiceClient),
        conns:   make(map[string]*grpc.ClientConn),
    }
}

func (c *RaftClient) Connect(peerID, address string) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        return err
    }

    client := pb.NewRaftServiceClient(conn)
    c.clients[peerID] = client
    c.conns[peerID] = conn
    return nil
}

func (c *RaftClient) RequestVote(ctx context.Context, peerID string, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
    c.mu.Lock()
    client := c.clients[peerID]
    c.mu.Unlock()

    return client.RequestVote(ctx, req)
}

func (c *RaftClient) AppendEntries(ctx context.Context, peerID string, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
    c.mu.Lock()
    client := c.clients[peerID]
    c.mu.Unlock()

    return client.AppendEntries(ctx, req)
}

func (c *RaftClient) Close() {
    c.mu.Lock()
    defer c.mu.Unlock()

    for _, conn := range c.conns {
        conn.Close()
    }
}