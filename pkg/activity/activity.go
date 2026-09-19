// Package activity provides the single emission point for audit events.
// Call sites record best-effort: failures are logged, never propagated —
// auditing must not break business operations.
package activity

import (
	"context"

	"github.com/cloudreve/Cloudreve/v4/inventory"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
	"github.com/gofrs/uuid"
)

// Opt annotates a recorded event.
type Opt func(*inventory.RecordActivityParams)

// File attaches the subject file ID.
func File(id int) Opt {
	return func(p *inventory.RecordActivityParams) { p.FileID = id }
}

// Actor overrides the actor ID when ctx carries no user (e.g. token
// refresh, activation) or the actor differs from the ctx user.
func Actor(id int) Opt {
	return func(p *inventory.RecordActivityParams) { p.ActorID = id }
}

// Share attaches the subject share ID.
func Share(id int) Opt {
	return func(p *inventory.RecordActivityParams) { p.ShareID = id }
}

// Extra merges metadata into the event payload.
func Extra(extra map[string]any) Opt {
	return func(p *inventory.RecordActivityParams) {
		if p.Extra == nil {
			p.Extra = map[string]any{}
		}
		for k, v := range extra {
			p.Extra[k] = v
		}
	}
}

// Record writes one event of the given type, if the admin has the type
// enabled. Actor, IP and correlation ID are derived from ctx. The event
// inherits any ambient transaction so it rolls back together with the
// operation it describes — no phantom events for failed ops.
func Record(ctx context.Context, settings setting.Provider, client inventory.ActivityClient, eventType int, opts ...Opt) {
	if !settings.AuditLogEnabled(ctx, eventType) {
		return
	}

	params := &inventory.RecordActivityParams{
		Type:  eventType,
		IP:    clientIPFromContext(ctx),
		Extra: map[string]any{},
	}
	if cid := logging.CorrelationID(ctx); cid != uuid.Nil {
		params.CID = cid.String()
	}
	if actor := inventory.UserFromContext(ctx); actor != nil {
		params.ActorID = actor.ID
	}
	for _, opt := range opts {
		opt(params)
	}

	txClient, _ := inventory.InheritTx(ctx, client)
	if _, err := txClient.Record(ctx, params); err != nil {
		logging.FromContext(ctx).Warning("failed to record activity event %d: %s", eventType, err)
	}
}

// clientIPFromContext extracts the client IP when ctx is an HTTP request
// context (gin.Context implements ClientIP()). Worker contexts return "".
func clientIPFromContext(ctx context.Context) string {
	type clientIPer interface {
		ClientIP() string
	}
	if c, ok := ctx.(clientIPer); ok && c != nil {
		return c.ClientIP()
	}
	return ""
}
