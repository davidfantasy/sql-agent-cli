package cli

import "github.com/davidfantasy/sql-agent-cli/internal/output"

func structEnvelopeForTest() output.Envelope {
	return output.Envelope{OK: true, Data: map[string]any{"ok": true}}
}
