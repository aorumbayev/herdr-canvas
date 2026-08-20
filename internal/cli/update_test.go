package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"herdr-canvas/internal/update"
	"herdr-canvas/internal/version"
)

func TestUpdateDevDoesNotCheck(t *testing.T) {
	prevC, prevA := updateCheck, updateApply
	t.Cleanup(func() {
		updateCheck, updateApply = prevC, prevA
		version.Version = "dev"
	})
	updateCheck = func(context.Context) (update.Result, error) {
		t.Fatal("Check must not run")
		return update.Result{}, nil
	}
	updateApply = func(context.Context, string) error {
		t.Fatal("Apply must not run")
		return nil
	}
	root := newRootCmd()
	root.SetArgs([]string{"update"})
	err := root.Execute()
	if err == nil {
		t.Fatal("want exit error")
	}
	if !strings.Contains(err.Error(), "development build") {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdateAlreadyUpToDate(t *testing.T) {
	prevC, prevA, prevV := updateCheck, updateApply, version.Version
	t.Cleanup(func() {
		updateCheck, updateApply = prevC, prevA
		version.Version = prevV
	})
	version.Version = "0.2.0"
	updateCheck = func(context.Context) (update.Result, error) {
		return update.Result{Current: "0.2.0", Latest: "0.2.0", Newer: false}, nil
	}
	updateApply = func(context.Context, string) error {
		t.Fatal("Apply must not run when up to date")
		return nil
	}
	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"update"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "already up to date (0.2.0)\n" {
		t.Fatalf("out = %q", got)
	}
}

func TestUpdateAppliesNewer(t *testing.T) {
	prevC, prevA, prevV := updateCheck, updateApply, version.Version
	t.Cleanup(func() {
		updateCheck, updateApply = prevC, prevA
		version.Version = prevV
	})
	version.Version = "0.1.0"
	var applied string
	updateCheck = func(context.Context) (update.Result, error) {
		return update.Result{Current: "0.1.0", Latest: "0.2.0", Newer: true}, nil
	}
	updateApply = func(_ context.Context, latest string) error {
		applied = latest
		return nil
	}
	root := newRootCmd()
	root.SetOut(io.Discard)
	root.SetArgs([]string{"update"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if applied != "0.2.0" {
		t.Fatalf("applied %q", applied)
	}
}

func TestUpdateCheckFailure(t *testing.T) {
	prevC, prevV := updateCheck, version.Version
	t.Cleanup(func() {
		updateCheck = prevC
		version.Version = prevV
	})
	version.Version = "0.1.0"
	updateCheck = func(context.Context) (update.Result, error) {
		return update.Result{}, errors.New("latest release: HTTP 404")
	}
	root := newRootCmd()
	root.SetArgs([]string{"update"})
	if err := root.Execute(); err == nil {
		t.Fatal("want check error")
	}
}
