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

// ParseLogMode maps a settings string to a log-masking Mode. Logs are free-form
// and default to MaskSensitive (best-effort secret redaction); only an explicit
// "off" disables it. There is no all-values mode, masking a whole log line
// wholesale would destroy it. The MCP/agent path floors this to MaskSensitive
// regardless, so an automated client can never set logs to "off".
func ParseLogMode(s string) Mode {
	if Mode(s) == MaskOff {
		return MaskOff
	}
	return MaskSensitive
}

// masked is a fixed placeholder, no characters and no length leaked.
const masked = "••••••••"

// sensitiveKey substrings (matched case-insensitively against the var name).
var sensitiveKey = []string{
	"KEY", "SECRET", "TOKEN", "PASSWORD", "PASSWD", "PASS", "PWD",
	"CREDENTIAL", "CRED", "AUTH", "PRIVATE", "SIGNING", "SALT",
	"APIKEY", "ACCESS", "SESSION", "DSN", "CONNECTIONSTRING",
	"BEARER", "JWT", "WEBHOOK", "CERT", "PEM", "GPG", "PGP", "OTP", "LICENSE",
}

// secretPrefix value shapes that are almost always a credential.
var secretPrefix = []string{
	"sk-", "rk-", "pk_live", "sk_live", "ghp_", "gho_", "ghs_", "ghr_",
	"github_pat_", "glpat-", "AKIA", "ASIA", "xoxb-", "xoxp-", "xoxa-",
	"xoxr-", "xoxs-", "-----BEGIN", "eyJ", "AIza", "ya29.", "npm_", "dckr_pat_",
}

// IsSensitive reports whether an env entry should be treated as a secret, by the
// variable name or by the shape of its value.
func IsSensitive(key, value string) bool {
	return nameIsSensitive(key) || looksSecret(value)
}

// nameIsSensitive reports whether a variable/flag/field name alone marks the
// entry as holding a secret, by a known sensitive substring (case-insensitive).
// Split out from IsSensitive so log masking can match a `password:`-style label
// whose value sits in the next token.
func nameIsSensitive(key string) bool {
	up := strings.ToUpper(key)
	for _, s := range sensitiveKey {
		if strings.Contains(up, s) {
			return true
		}
	}
	return false
}

func looksSecret(v string) bool {
	for _, p := range secretPrefix {
		if strings.HasPrefix(v, p) {
			return true
		}
	}
	// A URL/DSN is a secret only when it carries credentials in the userinfo
	// (scheme://user:pass@host, e.g. DATABASE_URL / REDIS_URL); a plain URL is
	// not. Checked before the token heuristic since URLs aren't bare tokens.
	if i := strings.Index(v, "://"); i >= 0 {
		rest := v[i+3:]
		if at := strings.IndexByte(rest, '@'); at > 0 {
			if cred := rest[:at]; strings.IndexByte(cred, ':') >= 0 && !strings.ContainsAny(cred, "/?#") {
				return true
			}
		}
		return false
	}
	// Best-effort: a long, whitespace-free, token-shaped run is probably a
	// credential (hex, base64, base64url). Exclude obvious filesystem paths.
	// Sensitive mode is heuristic; "all" is the safe default.
	if len(v) < 24 || strings.ContainsAny(v, " \t\n") {
		return false
	}
	if v[0] == '/' || v[0] == '~' {
		return false
	}
	for _, r := range v {
		switch {
		case r >= '0' && r <= '9',
			r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r == '_' || r == '-' || r == '+' || r == '=' || r == '.' || r == '/':
			// token-safe character
		default:
			return false // a non-token char → likely not a bare credential
		}
	}
	return true
}

// MaskValue returns the placeholder for a non-empty value (empty stays empty,
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

// MaskLabels masks only label values that look sensitive (by name or value
// shape). Unlike env, label sets are mostly metadata (compose project, image
// version), so "all" is not applied wholesale, that would gut the inspect view.
func MaskLabels(labels map[string]string, mode Mode) map[string]string {
	if mode == MaskOff || len(labels) == 0 {
		return labels
	}
	out := make(map[string]string, len(labels))
	for k, v := range labels {
		if v != "" && IsSensitive(k, v) {
			out[k] = MaskValue(v)
		} else {
			out[k] = v
		}
	}
	return out
}

// MaskCommand masks credential-looking tokens inside a container's command line,
// leaving the rest readable: a `--flag=value` / `KEY=value` whose name or value
// is sensitive, or a bare token that looks like a credential (sk-…, JWT, long
// token). A command is mostly non-secret, so only detected tokens are masked;
// "off" disables it. Heuristic, combined forms like `-psecret` aren't caught.
func MaskCommand(cmd string, mode Mode) string {
	if mode == MaskOff || cmd == "" {
		return cmd
	}
	fields := strings.Fields(cmd)
	for i, f := range fields {
		if k, v, ok := strings.Cut(f, "="); ok && v != "" {
			if IsSensitive(strings.TrimLeft(k, "-"), v) {
				fields[i] = k + "=" + MaskValue(v)
			}
			continue
		}
		if looksSecret(f) {
			fields[i] = MaskValue(f)
		}
	}
	return strings.Join(fields, " ")
}

// MaskLine redacts secret-shaped tokens from a single free-form log line,
// leaving the rest readable. A log line has no fixed structure, so unlike env
// (KEY=VALUE) this is best-effort: it masks `KEY=secret` / `--flag=secret`
// pairs, `password: secret` / `"token":"secret"` labelled values,
// `Authorization: Bearer <token>` headers, credentialed connection strings, and
// bare tokens that match a known secret shape (sk-…, JWT, long high-entropy
// runs). Benign content (level=info, environment=production) is left intact so
// the line stays useful to a viewer or an AI client.
//
// MaskOff is a pass-through. MaskSensitive and MaskAll behave identically here:
// a log line can't be masked "wholesale" without destroying it, so there is no
// all-values mode. The masking is applied to the value tokens by literal
// replacement, preserving the line's original spacing.
func MaskLine(line string, mode Mode) string {
	if mode == MaskOff || line == "" {
		return line
	}
	fields := strings.Fields(line)
	var redact []string // value tokens to replace with the placeholder
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		// KEY=value / --flag=value (the common structured-log and argv shape).
		if k, v, ok := strings.Cut(f, "="); ok && v != "" {
			if IsSensitive(strings.TrimLeft(k, "-"), v) {
				redact = append(redact, v)
			}
			continue
		}
		// Colon forms, but never a URL scheme (handled by looksSecret below).
		if strings.Contains(f, ":") && !strings.Contains(f, "://") {
			k, v, _ := strings.Cut(f, ":")
			name := strings.Trim(k, `"' `)
			switch {
			case v != "": // key:value / "key":"value" in one token (JSON / logfmt)
				if nameIsSensitive(name) {
					// Stop at the next separator so a packed `"k":"v","k2":"v2"`
					// token redacts only this value, not the rest of the line.
					if c := strings.IndexByte(v, ','); c >= 0 {
						v = v[:c]
					}
					if v = strings.Trim(v, `"'{} `); v != "" {
						redact = append(redact, v)
					}
					continue
				}
			case nameIsSensitive(name): // "Authorization:" / "password:" then a separate value token
				if j := i + 1; j < len(fields) {
					switch strings.ToLower(fields[j]) { // skip an auth scheme word
					case "bearer", "basic", "token", "digest":
						if j+1 < len(fields) {
							redact = append(redact, fields[j+1])
							i = j + 1
							continue
						}
					}
					redact = append(redact, fields[j])
					i = j
					continue
				}
			}
		}
		// Bare token that matches a known secret shape.
		if looksSecret(f) {
			redact = append(redact, f)
		}
	}
	if len(redact) == 0 {
		return line
	}
	for _, s := range redact {
		if s != "" && s != masked {
			line = strings.ReplaceAll(line, s, masked)
		}
	}
	return line
}
