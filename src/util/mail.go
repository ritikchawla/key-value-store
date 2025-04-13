package util

import (
    "context"
    "sync"
)

type Mailbox struct {
    mu      sync.RWMutex
    boxes   map[string]chan interface{}
}

func NewMailbox() *Mailbox {
    return &Mailbox{
        boxes: make(map[string]chan interface{}),
    }
}

func (m *Mailbox) CreateBox(id string, bufferSize int) chan interface{} {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    ch := make(chan interface{}, bufferSize)
    m.boxes[id] = ch
    return ch
}

func (m *Mailbox) Send(ctx context.Context, to string, msg interface{}) error {
    m.mu.RLock()
    ch, ok := m.boxes[to]
    m.mu.RUnlock()
    
    if !ok {
        return ErrNoSuchMailbox
    }
    
    select {
    case ch <- msg:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (m *Mailbox) Close(id string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if ch, ok := m.boxes[id]; ok {
        close(ch)
        delete(m.boxes, id)
    }
}