package util

import (
	"crypto/sha256"
	"encoding/hex"
)

// ComputeDeviceFingerprint hashes signals that stay stable across requests
// from the same browser/device (but differ across browsers/devices) into a
// single opaque value an account can be bound to.
func ComputeDeviceFingerprint(userAgent, acceptLanguage, acceptEncoding string) string {
	sum := sha256.Sum256([]byte(userAgent + "|" + acceptLanguage + "|" + acceptEncoding))
	return hex.EncodeToString(sum[:])
}
