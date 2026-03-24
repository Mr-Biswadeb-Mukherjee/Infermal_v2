// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package engine

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

type ctxKey string

const traceIDKey ctxKey = "trace-id"

func restoreEngineHooks(t *testing.T) {
	oldRunner := runApp
	oldPrinter := printLine
	t.Cleanup(func() {
		runApp = oldRunner
		printLine = oldPrinter
	})
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var b bytes.Buffer
		_, _ = io.Copy(&b, r)
		done <- b.String()
	}()

	fn()
	_ = w.Close()
	os.Stdout = oldStdout
	out := <-done
	_ = r.Close()
	return out
}

func TestRunInvokesAppRunner(t *testing.T) {
	restoreEngineHooks(t)

	var gotCtx context.Context
	var lines []string

	runApp = func(ctx context.Context, deps Dependencies) error {
		gotCtx = ctx
		_ = deps
		return nil
	}
	printLine = func(args ...any) {
		lines = append(lines, strings.TrimSpace(fmt.Sprintln(args...)))
	}

	Run(Dependencies{})

	if gotCtx == nil {
		t.Fatal("expected Run to pass a context to app runner")
	}
	if gotCtx.Err() != nil {
		t.Fatalf("expected non-canceled context, got err=%v", gotCtx.Err())
	}
	if len(lines) != 1 {
		t.Fatalf("expected one printed line, got %d", len(lines))
	}
	if lines[0] != "Shutdown complete." {
		t.Fatalf("unexpected output: %q", lines[0])
	}
}

func TestRunPrintsErrorAndShutdown(t *testing.T) {
	restoreEngineHooks(t)

	var lines []string
	runApp = func(context.Context, Dependencies) error {
		return errors.New("boom")
	}
	printLine = func(args ...any) {
		lines = append(lines, strings.TrimSpace(fmt.Sprintln(args...)))
	}

	Run(Dependencies{})

	if len(lines) != 2 {
		t.Fatalf("expected two printed lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "Error: boom") {
		t.Fatalf("expected error output, got %q", lines[0])
	}
	if lines[1] != "Shutdown complete." {
		t.Fatalf("unexpected shutdown output: %q", lines[1])
	}
}

func TestRunContextHasNoForcedDeadline(t *testing.T) {
	restoreEngineHooks(t)

	runApp = func(ctx context.Context, deps Dependencies) error {
		_ = deps
		if _, ok := ctx.Deadline(); ok {
			t.Fatal("run context should not have an implicit deadline")
		}
		return nil
	}
	printLine = func(args ...any) {}

	Run(Dependencies{})
}

func TestRunWithDefaultRunnerStillShutsDown(t *testing.T) {
	oldRunner := runApp
	restoreEngineHooks(t)

	var lines []string
	runApp = oldRunner
	printLine = func(args ...any) {
		lines = append(lines, strings.TrimSpace(fmt.Sprintln(args...)))
	}

	Run(Dependencies{})

	if len(lines) != 1 {
		t.Fatalf("expected one printed line, got %d", len(lines))
	}
	if lines[0] != "Shutdown complete." {
		t.Fatalf("unexpected output: %q", lines[0])
	}
}

func TestDefaultPrintLineWritesToStdout(t *testing.T) {
	restoreEngineHooks(t)
	defaultPrinter := printLine

	out := captureStdout(t, func() {
		defaultPrinter("engine-output-check")
	})

	if !strings.Contains(out, "engine-output-check") {
		t.Fatalf("expected stdout output, got %q", out)
	}
}

func TestNormalizeContextNilReturnsBackground(t *testing.T) {
	ctx := normalizeContext(context.TODO())
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	select {
	case <-ctx.Done():
		t.Fatalf("background context should not be canceled: %v", ctx.Err())
	case <-time.After(1 * time.Millisecond):
	}
}

func TestNormalizeContextReturnsSameContext(t *testing.T) {
	base := context.WithValue(context.Background(), traceIDKey, "test-123")
	got := normalizeContext(base)
	if got != base {
		t.Fatal("expected normalizeContext to preserve non-nil context")
	}
	if got.Value(traceIDKey) != "test-123" {
		t.Fatal("expected context values to be preserved")
	}
}
