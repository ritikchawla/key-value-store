package util

import "errors"

var (
    ErrTimeout        = errors.New("operation timed out")
    ErrNotLeader      = errors.New("not the leader")
    ErrNodeExists     = errors.New("node already exists")
    ErrNoSuchMailbox  = errors.New("no such mailbox")
    ErrNodeNotFound   = errors.New("node not found")
    ErrInvalidTerm    = errors.New("invalid term")
    ErrStaleRead      = errors.New("stale read")
)