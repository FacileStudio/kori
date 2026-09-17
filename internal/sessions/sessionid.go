package sessions

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// GenerateSessionID produces a random 8-character hex session identifier.
func GenerateSessionID() string {
	return generateSessionID("")
}

func generateSessionID(dir string) string {
	for {
		b := make([]byte, 4)
		if _, err := rand.Read(b); err != nil {
			return fmt.Sprintf("%08x", time.Now().UnixNano()&0xffffffff)
		}
		id := hex.EncodeToString(b)
		if dir == "" {
			return id
		}
		if _, err := os.Stat(filepath.Join(dir, id+".jsonl")); os.IsNotExist(err) {
			return id
		}
	}
}
