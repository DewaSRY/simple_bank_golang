package auth

import (
	"slices"

	"github.com/DewaSRY/core-service/internal/util"
	"github.com/gin-gonic/gin"
)

// trustedIPMockFingerprint is the fixed value returned for any request from
// an IP in Handler.TrustedIPs, instead of one derived from headers. It's a
// constant input through the same hashing function real fingerprints use, so
// it's shaped identically (64 hex chars) and stored the same way.
var trustedIPMockFingerprint = util.ComputeDeviceFingerprint("trusted-ip-dev-bypass", "", "")

// requestDeviceFingerprint derives a stable identifier for the caller's
// browser/device from request headers, unless the request comes from a
// configured trusted IP (h.TrustedIPs), in which case it returns a fixed
// mock fingerprint so local tooling (Swagger UI, curl) never gets rejected
// as "bound to a different device" — see Handler.TrustedIPs.
func (h *Handler) requestDeviceFingerprint(ctx *gin.Context) string {

	if slices.Contains(h.TrustedIPs, ctx.ClientIP()) {
		return trustedIPMockFingerprint
	}

	return util.ComputeDeviceFingerprint(
		ctx.GetHeader("User-Agent"),
		ctx.GetHeader("Accept-Language"),
		ctx.GetHeader("Accept-Encoding"),
	)
}
