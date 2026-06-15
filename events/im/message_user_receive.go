// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package im

import (
	"context"
	"encoding/json"
	"time"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/event"
	"github.com/larksuite/cli/internal/im/userreceive"
)

const (
	eventTypeMessageUserReceive = userreceive.EventType
)

// ImMessageUserReceiveOutput is the flattened shape for im.message.user_receive_v1.
// The user-scoped event intentionally exposes only message identifiers; callers
// should fetch message details explicitly when they need content.
type ImMessageUserReceiveOutput struct {
	Type      string `json:"type"                 desc:"Event type; always im.message.user_receive_v1"`
	EventID   string `json:"event_id,omitempty"   desc:"Globally unique event ID; safe for deduplication"`
	Timestamp string `json:"timestamp,omitempty"  desc:"Event delivery time (ms timestamp string); taken from header.create_time when present" kind:"timestamp_ms"`
	ID        string `json:"id,omitempty"         desc:"Message ID (legacy alias of message_id, kept for compatibility). Use im +messages-mget with this ID when message body details are needed." kind:"message_id"`
	MessageID string `json:"message_id,omitempty" desc:"Message ID; prefixed with om_. Use im +messages-mget with this ID when message body details are needed."                            kind:"message_id"`
}

func processImMessageUserReceive(_ context.Context, _ event.APIClient, raw *event.RawEvent, _ map[string]string) (json.RawMessage, error) {
	var envelope struct {
		Header struct {
			EventID    string `json:"event_id"`
			EventType  string `json:"event_type"`
			CreateTime string `json:"create_time"`
		} `json:"header"`
		Event struct {
			MessageID string `json:"message_id"`
			Message   struct {
				MessageID string `json:"message_id"`
			} `json:"message"`
		} `json:"event"`
	}
	if err := json.Unmarshal(raw.Payload, &envelope); err != nil {
		return raw.Payload, nil //nolint:nilerr // passthrough on malformed payload so consumers still see the event
	}

	messageID := envelope.Event.MessageID
	if messageID == "" {
		messageID = envelope.Event.Message.MessageID
	}
	out := &ImMessageUserReceiveOutput{
		Type:      envelope.Header.EventType,
		EventID:   envelope.Header.EventID,
		Timestamp: envelope.Header.CreateTime,
		ID:        messageID,
		MessageID: messageID,
	}
	if out.Type == "" {
		out.Type = raw.EventType
	}
	if out.EventID == "" {
		out.EventID = raw.EventID
	}
	if out.Timestamp == "" {
		out.Timestamp = raw.SourceTime
	}
	return json.Marshal(out)
}

func normalizeMessageUserReceiveParams(_ context.Context, _ event.APIClient, params map[string]string) error {
	return userReceiveParamError(userreceive.NormalizeParams(params))
}

func messageUserReceivePreConsume(ctx context.Context, rt event.APIClient, params map[string]string) (func() error, error) {
	if rt == nil {
		return nil, errs.NewInternalError(errs.SubtypeUnknown,
			"runtime API client is required for pre-consume subscription")
	}
	body, err := userreceive.BuildSubscriptionBody(params)
	if err != nil {
		return nil, userReceiveParamError(err)
	}
	raw, err := rt.CallAPI(ctx, "POST", userreceive.SubscribePath, body)
	if err != nil {
		return nil, err
	}
	subscriptionIDs := userreceive.ParseSubscriptionIDs(raw)
	if len(subscriptionIDs) == 0 {
		return func() error { return nil }, nil
	}

	return func() error {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := rt.CallAPI(cleanupCtx, "POST", userreceive.DeletePath, map[string]interface{}{
			"subscription_ids": subscriptionIDs,
		})
		return err
	}, nil
}

func userReceiveParamError(err error) error {
	if err == nil {
		return nil
	}
	param := ""
	if paramErr, ok := err.(*userreceive.ParamError); ok {
		param = paramErr.Field
	}
	validationErr := errs.NewValidationError(errs.SubtypeInvalidArgument, "%s", err).
		WithHint("run `lark-cli event schema %s` for resource_type and resource_ids usage", eventTypeMessageUserReceive)
	if param != "" {
		validationErr.WithParam(param)
	}
	return validationErr
}
