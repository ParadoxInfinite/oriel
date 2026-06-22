// Package secrets masks sensitive environment-variable values so they don't leak
// from the inspect panel (screenshots, screen-shares) or to an AI model over MCP.
// Masking is policy applied above the docker layer, which always returns raw env;
// callers decide what the viewer is allowed to see.
package secrets

import "strings"

// Mode is how container env values are masked.
type Mode string

const (
	MaskAll       Mode = "all"       // mask every value (default)
	MaskSensitive Mode = "sensitive" // mask only values that look like secrets
	MaskOff       Mode = "off"       // no masking
)

// ParseMode maps a settings string to a Mode, defaulting to MaskAll.
func ParseMode(s string) Mode {
	switch Mode(s) {
	case MaskSensitive:
		return MaskSensitive
	case MaskOff:
		return MaskOff
	default:
		return MaskAll
	}
}

// masked is a fixed placeholder — no characters and no length leaked.
const masked = "••••••••"

// sensitiveKey substrings (matched case-insensitively against the var name).
var sensitiveKey = []string{
	"KEY", "SECRET", "TOKEN", "PASSWORD", "PASSWD", "PASS",
	"CREDENTIAL", "CRED", "AUTH", "PRIVATE", "SIGNING", "SALT",
	"APIKEY", "ACCESS", "SESSION", "DSN", "CONNECTIONSTRING",
}

// secretPrefix value shapes that are almost always a credential.
var secretPrefix = []string{
	"sk-", "rk-", "pk_live", "sk_live", "ghp_", "gho_", "ghs_", "ghr_",
	"github_pat_", "glpat-", "AKIA", "ASIA", "xoxb-", "xoxp-", "xoxa-",
	"xoxr-", "xoxs-", "-----BEGIN", "eyJ",
}

// IsSensitive reports whether an env entry should be treated as a secret, by the
// variable name or by the shape of its value.
func IsSensitive(key, value string) bool {
	up := strings.ToUpper(key)
	for _, s := range sensitiveKey {
		if strings.Contains(up, s) {
			return true
		}
	}
	return looksSecret(value)
}

func looksSecret(v string) bool {
	for _, p := range secretPrefix {
		if strings.HasPrefix(v, p) {
			return true
		}
	}
	// High-entropy token: long, no spaces, no path separators (so we don't mask
	// PATH-like values), a mix of letters and digits.
	if len(v) < 32 || strings.ContainsAny(v, " \t/") {
		return false
	}
	var hasLetter, hasDigit bool
	for _, r := range v {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			hasLetter = true
		case r == '_' || r == '-' || r == '+' || r == '=' || r == '.':
			// token-safe punctuation
		default:
			return false // unexpected char → probably not a token
		}
	}
	return hasLetter && hasDigit
}

// MaskValue returns the placeholder for a non-empty value (empty stays empty —
// there's nothing to hide).
func MaskValue(v string) string {
	if v == "" {
		return ""
	}
	return masked
}

// MaskEnv returns a copy of env ("KEY=VALUE" entries) with values masked per mode.
// Entries without an '=' are passed through unchanged.
func MaskEnv(env []string, mode Mode) []string {
	if mode == MaskOff || len(env) == 0 {
		return env
	}
	out := make([]string, len(env))
	for i, kv := range env {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || v == "" {
			out[i] = kv
			continue
		}
		if mode == MaskAll || IsSensitive(k, v) {
			out[i] = k + "=" + MaskValue(v)
		} else {
			out[i] = kv
		}
	}
	return out
}
