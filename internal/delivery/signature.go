
package delivery

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateSignature creates an HMAC-SHA256 signature.
func GenerateSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))

	mac.Write(payload)

	signature := hex.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("sha256=%s", signature)
}
