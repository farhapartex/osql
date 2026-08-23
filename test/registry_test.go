package test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/farhapartex/osql/internal/engine"
	"github.com/farhapartex/osql/internal/query"
)

func TestRegistryLookup(t *testing.T) {
	sel := &fakeExecutor{verb: "select"}
	r := engine.NewRegistry(sel)

	got, ok := r.Lookup("select")
	if !ok {
		t.Fatal("Lookup(\"select\") = _, false; want true")
	}
	if got != sel {
		t.Errorf("Lookup returned a different executor than registered")
	}
}

func TestRegistryLookupUnknownVerb(t *testing.T) {
	r := engine.NewRegistry(&fakeExecutor{verb: "select"})

	if _, ok := r.Lookup("delete"); ok {
		t.Error("Lookup(\"delete\") = _, true; want false for an unregistered verb")
	}
	if _, ok := r.Lookup(""); ok {
		t.Error("Lookup(\"\") = _, true; want false")
	}
	if _, ok := r.Lookup("SELECT"); ok {
		t.Error("Lookup is case-sensitive; the parser lowercases keywords before dispatch")
	}
}

func TestRegistryEmpty(t *testing.T) {
	r := engine.NewRegistry()

	if _, ok := r.Lookup("select"); ok {
		t.Error("empty registry returned an executor")
	}
	if verbs := r.Verbs(); len(verbs) != 0 {
		t.Errorf("Verbs() = %v, want empty", verbs)
	}
}

func TestRegistryZeroValueIsUsable(t *testing.T) {
	var r engine.Registry

	if _, ok := r.Lookup("select"); ok {
		t.Error("zero-value registry returned an executor")
	}
	r.Register(&fakeExecutor{verb: "select"})
	if _, ok := r.Lookup("select"); !ok {
		t.Error("Register on a zero-value Registry did not take effect")
	}
}

func TestRegistryVerbsAreSorted(t *testing.T) {
	r := engine.NewRegistry(
		&fakeExecutor{verb: "select"},
		&fakeExecutor{verb: "delete"},
		&fakeExecutor{verb: "create"},
	)

	got := r.Verbs()
	want := []string{"create", "delete", "select"}
	if !slices.Equal(got, want) {
		t.Errorf("Verbs() = %v, want %v (sorted, so error messages are deterministic)", got, want)
	}
}

func TestRegistryRegisterOverwritesSameVerb(t *testing.T) {
	first := &fakeExecutor{verb: "select"}
	second := &fakeExecutor{verb: "select"}

	r := engine.NewRegistry(first)
	r.Register(second)

	got, _ := r.Lookup("select")
	if got != second {
		t.Error("re-registering a verb did not replace the previous executor")
	}
	if verbs := r.Verbs(); len(verbs) != 1 {
		t.Errorf("Verbs() = %v, want one entry after overwrite", verbs)
	}
}

func TestExecutorPushesRowsIntoSink(t *testing.T) {
	rows := []engine.Row{{Name: "a.txt"}, {Name: "b.txt"}, {Name: "c.txt"}}
	exec := &fakeExecutor{verb: "select", rows: rows}
	sink := &sliceSink{}

	if err := exec.Execute(context.Background(), &query.Statement{}, sink); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(sink.rows) != 3 {
		t.Errorf("sink received %d rows, want 3", len(sink.rows))
	}
}

func TestSinkStopsWalkAtLimit(t *testing.T) {
	rows := []engine.Row{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	exec := &fakeExecutor{verb: "select", rows: rows}
	sink := &sliceSink{limit: 2}

	err := exec.Execute(context.Background(), &query.Statement{}, sink)
	if !errors.Is(err, engine.ErrStopWalk) {
		t.Fatalf("Execute() error = %v, want ErrStopWalk once the sink is full", err)
	}
	if len(sink.rows) != 2 {
		t.Errorf("sink collected %d rows, want 2", len(sink.rows))
	}
}
