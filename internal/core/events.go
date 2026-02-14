package core

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

const EventSchemaVersion = 2

type ErrorObject struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Detail     string `json:"detail,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

type Event struct {
	V     int          `json:"version"`
	TS    string       `json:"timestamp"`
	Cmd   string       `json:"command"`
	Type  string       `json:"type"`
	Level string       `json:"level,omitempty"`
	Code  string       `json:"code,omitempty"`
	Msg   string       `json:"message,omitempty"`
	Data  any          `json:"data,omitempty"`
	Err   *ErrorObject `json:"error,omitempty"`
}

func NowTS() string { return time.Now().UTC().Format(time.RFC3339Nano) }

type Emitter interface {
	Emit(ev Event)
}

type NDJSONEmitter struct {
	w       io.Writer
	version int
}

func NewNDJSONEmitter(w io.Writer, version int) *NDJSONEmitter {
	if version <= 0 {
		version = EventSchemaVersion
	}
	return &NDJSONEmitter{w: w, version: version}
}

func (e *NDJSONEmitter) Emit(ev Event) {
	if ev.V == 0 {
		ev.V = e.version
	}
	if ev.TS == "" {
		ev.TS = NowTS()
	}
	b, err := json.Marshal(ev)
	if err != nil {
		// Last resort: emit a minimal JSON line.
		fmt.Fprintf(e.w, "{\"version\":%d,\"timestamp\":\"%s\",\"command\":\"%s\",\"type\":\"error\",\"message\":\"failed to encode event: %v\"}\n", e.version, NowTS(), ev.Cmd, err)
		return
	}
	e.w.Write(b)
	e.w.Write([]byte("\n"))
}

type TextEmitter struct {
	w io.Writer
}

func NewTextEmitter(w io.Writer) *TextEmitter { return &TextEmitter{w: w} }

func (e *TextEmitter) Emit(ev Event) {
	// Simple human output. The TUI has its own rendering.
	if ev.Msg != "" {
		if ev.Type == "log_raw" {
			return
		}
		if ev.Level != "" {
			fmt.Fprintf(e.w, "[%s] %s\n", ev.Level, ev.Msg)
			return
		}
		fmt.Fprintln(e.w, ev.Msg)
		return
	}
	if ev.Err != nil {
		fmt.Fprintf(e.w, "error[%s]: %s\n", ev.Err.Code, ev.Err.Message)
		if ev.Err.Suggestion != "" {
			fmt.Fprintf(e.w, "  hint: %s\n", ev.Err.Suggestion)
		}
	}
}

func Status(cmd, msg string, data any) Event {
	return Event{V: EventSchemaVersion, TS: NowTS(), Cmd: cmd, Type: "status", Level: "info", Msg: msg, Data: data}
}

func Log(cmd, msg string) Event {
	return Event{V: EventSchemaVersion, TS: NowTS(), Cmd: cmd, Type: "log", Level: "info", Msg: msg}
}

func LogStream(cmd, msg, stream string) Event {
	data := map[string]any{}
	if stream != "" {
		data["stream"] = stream
	}
	return Event{V: EventSchemaVersion, TS: NowTS(), Cmd: cmd, Type: "log", Level: "info", Msg: msg, Data: data}
}

func LogPretty(cmd, msg string) Event {
	return Event{V: EventSchemaVersion, TS: NowTS(), Cmd: cmd, Type: "log", Level: "info", Msg: msg, Data: map[string]any{"pretty": true}}
}

func LogRaw(cmd, msg string) Event {
	return Event{V: EventSchemaVersion, TS: NowTS(), Cmd: cmd, Type: "log_raw", Level: "info", Msg: msg}
}

func Warn(cmd, msg string) Event {
	return Event{V: EventSchemaVersion, TS: NowTS(), Cmd: cmd, Type: "warning", Level: "warn", Msg: msg}
}

func Err(cmd string, eo ErrorObject) Event {
	return Event{V: EventSchemaVersion, TS: NowTS(), Cmd: cmd, Type: "error", Level: "error", Err: &eo, Msg: eo.Message}
}

func Result(cmd string, ok bool, data any) Event {
	status := "success"
	if !ok {
		status = "failure"
	}
	return Event{V: EventSchemaVersion, TS: NowTS(), Cmd: cmd, Type: "result", Level: "info", Data: map[string]any{"status": status, "data": data}}
}
