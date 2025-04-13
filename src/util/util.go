package util

import (
    "crypto/sha256"
    "encoding/binary"
    "time"
    "math/rand"
)

// GenerateID generates a unique node ID
func GenerateID() string {
    timestamp := time.Now().UnixNano()
    random := rand.Int63()
    
    hash := sha256.New()
    binary.Write(hash, binary.BigEndian, timestamp)
    binary.Write(hash, binary.BigEndian, random)
    
    return string(hash.Sum(nil)[:16])
}

// RandomTimeout returns a random duration between min and max
func RandomTimeout(min, max time.Duration) time.Duration {
    delta := max - min
    random := rand.Int63n(int64(delta))
    return min + time.Duration(random)
}

// IsQuorum checks if count represents a quorum of total
func IsQuorum(count, total int) bool {
    return count > total/2
}