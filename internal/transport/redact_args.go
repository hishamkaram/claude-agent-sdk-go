package transport

// isSensitiveArgFlag reports whether the value following flag must never appear
// in a log dump. The inline --mcp-config JSON envelope can carry MCP server
// configuration and the delegate __delegate-shim --sock path is a
// Unix-domain-socket address; both are sensitive under the same-UID threat
// model. Flag presence is still logged so operators can confirm the flag was
// set; only the value is elided.
func isSensitiveArgFlag(flag string) bool {
	switch flag {
	case "--mcp-config", "--sock":
		return true
	default:
		return false
	}
}

// redactArgsForLog returns a copy of args suitable for a Debug log dump, with
// the value following any sensitive flag (see isSensitiveArgFlag) replaced by
// "[redacted]". This is logging-only: it never mutates the input slice and is
// never used to build the actual spawned argv. A trailing sensitive flag with
// no following value is left unchanged (there is nothing to redact).
func redactArgsForLog(args []string) []string {
	if args == nil {
		return nil
	}
	out := make([]string, len(args))
	copy(out, args)
	for i := 0; i < len(out); i++ {
		if isSensitiveArgFlag(out[i]) && i+1 < len(out) {
			out[i+1] = "[redacted]"
			i++ // skip the value we just redacted
		}
	}
	return out
}
