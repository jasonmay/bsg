package id

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

func Generate(name string, exists func(string) bool) string {
	raw := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", name, time.Now().UnixNano())))
	full := hex.EncodeToString(raw[:])

	id := "bsg-" + full[:4]
	if !exists(id) {
		return id
	}
	id = "bsg-" + full[:6]
	if !exists(id) {
		return id
	}
	return "bsg-" + full[:8]
}
