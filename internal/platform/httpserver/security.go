package httpserver

import "github.com/gin-gonic/gin"

// APIContentSecurityPolicy fits JSON responses: nothing may load, and nothing may frame them.
const APIContentSecurityPolicy = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"

type SecurityOptions struct {
	// HSTS tells browsers to use HTTPS only. Leave it off when the API is served over plain HTTP.
	HSTS bool
}

// SecurityHeaders sets defaults on every response, including errors. Routes that serve HTML,
// such as the API docs, replace Content-Security-Policy with their own.
func SecurityHeaders(opts SecurityOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Content-Security-Policy", APIContentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
		// Responses can carry patient data, so neither browsers nor proxies may store them.
		h.Set("Cache-Control", "no-store")
		if opts.HSTS {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}
