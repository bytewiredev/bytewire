package plugin

import "context"

// HookContext provides session information to hook functions.
type HookContext struct {
	SessionID uint64
	Path      string
	Ctx       context.Context
}

// Hook function types
type ConnectHook func(HookContext) error
type DisconnectHook func(HookContext)
type MountHook func(HookContext) error
type IntentHook func(HookContext, []byte) error
type FlushHook func(HookContext) error

// Registry holds registered lifecycle hooks.
type Registry struct {
	connectHooks    []ConnectHook
	disconnectHooks []DisconnectHook
	mountHooks      []MountHook
	intentHooks     []IntentHook
	flushHooks      []FlushHook
}

// Registration methods
func (r *Registry) OnConnect(h ConnectHook)       { r.connectHooks = append(r.connectHooks, h) }
func (r *Registry) OnDisconnect(h DisconnectHook) { r.disconnectHooks = append(r.disconnectHooks, h) }
func (r *Registry) OnMount(h MountHook)           { r.mountHooks = append(r.mountHooks, h) }
func (r *Registry) OnIntent(h IntentHook)         { r.intentHooks = append(r.intentHooks, h) }
func (r *Registry) OnFlush(h FlushHook)           { r.flushHooks = append(r.flushHooks, h) }

// Execution methods — all nil-safe (no-op if registry is nil)
func (r *Registry) RunConnect(ctx HookContext) error {
	if r == nil {
		return nil
	}
	for _, h := range r.connectHooks {
		if err := h(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) RunDisconnect(ctx HookContext) {
	if r == nil {
		return
	}
	for _, h := range r.disconnectHooks {
		h(ctx)
	}
}

func (r *Registry) RunMount(ctx HookContext) error {
	if r == nil {
		return nil
	}
	for _, h := range r.mountHooks {
		if err := h(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) RunIntent(ctx HookContext, data []byte) error {
	if r == nil {
		return nil
	}
	for _, h := range r.intentHooks {
		if err := h(ctx, data); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) RunFlush(ctx HookContext) error {
	if r == nil {
		return nil
	}
	for _, h := range r.flushHooks {
		if err := h(ctx); err != nil {
			return err
		}
	}
	return nil
}
