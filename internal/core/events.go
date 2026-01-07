package core

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type ErrorObject struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Detail     string `json:"detail,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

type Event struct {
	V     int          `json:"v"`
	TS    string       `json:"ts"`
	Cmd   string       `json:"cmd"`
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
	w io.Writer
}

func NewNDJSONEmitter(w io.Writer) *NDJSONEmitter { return &NDJSONEmitter{w: w} }

func (e *NDJSONEmitter) Emit(ev Event) {
	if ev.V == 0 {
		ev.V = 1
	}
	if ev.TS == "" {
		ev.TS = NowTS()
	}
	b, err := json.Marshal(ev)
	if err != nil {
		// Last resort: emit a minimal JSON line.
		fmt.Fprintf(e.w, "{\"v\":1,\"ts\":\"%s\",\"cmd\":\"%s\",\"type\":\"error\",\"message\":\"failed to encode event: %v\"}\n", NowTS(), ev.Cmd, err)
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
	return Event{V: 1, TS: NowTS(), Cmd: cmd, Type: "status", Level: "info", Msg: msg, Data: data}
}

func Log(cmd, msg string) Event {
	return Event{V: 1, TS: NowTS(), Cmd: cmd, Type: "log", Level: "info", Msg: msg}
}

func Warn(cmd, msg string) Event {
	return Event{V: 1, TS: NowTS(), Cmd: cmd, Type: "warning", Level: "warn", Msg: msg}
}

func Err(cmd string, eo ErrorObject) Event {
	return Event{V: 1, TS: NowTS(), Cmd: cmd, Type: "error", Level: "error", Err: &eo, Msg: eo.Message}
}

func Result(cmd string, ok bool, data any) Event {
	status := "success"
	if !ok {
		status = "failure"
	}
	return Event{V: 1, TS: NowTS(), Cmd: cmd, Type: "result", Level: "info", Data: map[string]any{"status": status, "data": data}}
}
