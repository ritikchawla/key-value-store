package raft

import (
    "context"
    pb "github.com/ritikchawla/key-value-store/proto"
    "google.golang.org/grpc"
    "net"
)

type RaftServer struct {
    pb.UnimplementedRaftServiceServer
    node *RaftNode
}

func NewRaftServer(node *RaftNode) *RaftServer {
    return &RaftServer{node: node}
}

func (s *RaftServer) RequestVote(ctx context.Context, req *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
    granted := s.node.HandleVoteRequest(ctx, req.CandidateId, req.Term, req.LastLogIndex, req.LastLogTerm)
    
    return &pb.RequestVoteResponse{
        Term:       s.node.currentTerm,
        VoteGranted: granted,
    }, nil
}

func (s *RaftServer) AppendEntries(ctx context.Context, req *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
    entries := make([]LogEntry, len(req.Entries))
    for i, e := range req.Entries {
        entries[i] = LogEntry{
            Term:    e.Term,
            Index:   e.Index,
            Command: e.Command,
        }
    }
    
    err := s.node.AppendEntries(ctx, req.Term, entries)
    success := err == nil
    
    return &pb.AppendEntriesResponse{
        Term:    s.node.currentTerm,
        Success: success,
    }, nil
}

func (s *RaftServer) InstallSnapshot(ctx context.Context, req *pb.InstallSnapshotRequest) (*pb.InstallSnapshotResponse, error) {
    snapshot := &Snapshot{
        LastIncludedIndex: req.LastIncludedIndex,
        LastIncludedTerm:  req.LastIncludedTerm,
        State:            req.Data,
    }
    
    if err := s.node.InstallSnapshot(snapshot); err != nil {
        return nil, err
    }
    
    return &pb.InstallSnapshotResponse{
        Term: s.node.currentTerm,
    }, nil
}

func StartRaftServer(node *RaftNode, addr string) error {
    lis, err := net.Listen("tcp", addr)
    if err != nil {
        return err
    }

    server := grpc.NewServer()
    pb.RegisterRaftServiceServer(server, &RaftServer{node: node})
    
    return server.Serve(lis)
}