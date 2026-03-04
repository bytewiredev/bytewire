package plugin

import (
	"context"
	"errors"
	"testing"
)

func TestHookOrdering(t *testing.T) {
	r := &Registry{}
	var order []int

	r.OnConnect(func(HookContext) error { order = append(order, 1); return nil })
	r.OnConnect(func(HookContext) error { order = append(order, 2); return nil })
	r.OnConnect(func(HookContext) error { order = append(order, 3); return nil })

	ctx := HookContext{SessionID: 1, Path: "/", Ctx: context.Background()}
	if err := r.RunConnect(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("expected 3 hooks to run, got %d", len(order))
	}
	for i, v := range order {
		if v != i+1 {
			t.Errorf("hook %d ran at position %d", v, i)
		}
	}
}

func TestErrorPropagation(t *testing.T) {
	r := &Registry{}
	errBlocked := errors.New("blocked")
	secondRan := false

	r.OnConnect(func(HookContext) error { return errBlocked })
	r.OnConnect(func(HookContext) error { secondRan = true; return nil })

	ctx := HookContext{SessionID: 1, Ctx: context.Background()}
	err := r.RunConnect(ctx)
	if !errors.Is(err, errBlocked) {
		t.Fatalf("expected errBlocked, got %v", err)
	}
	if secondRan {
		t.Fatal("second hook should not have run after first returned error")
	}
}

func TestNilRegistrySafety(t *testing.T) {
	var r *Registry

	ctx := HookContext{SessionID: 1, Ctx: context.Background()}

	if err := r.RunConnect(ctx); err != nil {
		t.Fatalf("RunConnect on nil: %v", err)
	}
	r.RunDisconnect(ctx) // should not panic
	if err := r.RunMount(ctx); err != nil {
		t.Fatalf("RunMount on nil: %v", err)
	}
	if err := r.RunIntent(ctx, []byte("test")); err != nil {
		t.Fatalf("RunIntent on nil: %v", err)
	}
	if err := r.RunFlush(ctx); err != nil {
		t.Fatalf("RunFlush on nil: %v", err)
	}
}

func TestDisconnectHooksAllRun(t *testing.T) {
	r := &Registry{}
	var count int

	r.OnDisconnect(func(HookContext) { count++ })
	r.OnDisconnect(func(HookContext) { count++ })
	r.OnDisconnect(func(HookContext) { count++ })

	ctx := HookContext{SessionID: 1, Ctx: context.Background()}
	r.RunDisconnect(ctx)

	if count != 3 {
		t.Fatalf("expected 3 disconnect hooks to run, got %d", count)
	}
}
