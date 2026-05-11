package cli

import "github.com/david/sql-agent-cli/internal/output"

func structEnvelopeForTest() output.Envelope {
	return output.Envelope{OK: true, Data: map[string]any{"ok": true}}
}
