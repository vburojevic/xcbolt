package core

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestEventJSONUsesV2FieldNames(t *testing.T) {
	ev := Status("build", "Build started", map[string]any{"id": "123"})
	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}

	for _, key := range []string{"version", "timestamp", "command", "type", "message"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("missing key %q in %v", key, got)
		}
	}
	for _, key := range []string{"v", "ts", "cmd"} {
		if _, ok := got[key]; ok {
			t.Fatalf("unexpected legacy key %q in %v", key, got)
		}
	}
}

func TestNDJSONEmitterUsesConfiguredVersion(t *testing.T) {
	var buf bytes.Buffer
	e := NewNDJSONEmitter(&buf, EventSchemaVersion)
	e.Emit(Event{Cmd: "build", Type: "status", Msg: "ok"})

	var got map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &got); err != nil {
		t.Fatalf("unmarshal line: %v", err)
	}
	if got["version"] != float64(EventSchemaVersion) {
		t.Fatalf("version = %v, want %d", got["version"], EventSchemaVersion)
	}
}
