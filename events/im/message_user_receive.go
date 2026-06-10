// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package im

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/event"
)

const (
	eventTypeMessageUserReceive = "im.message.user_receive_v1"

	pathMessageUserReceiveSubscribe = "/open-apis/im/v1/user_message_subscriptions"
	pathMessageUserReceiveDelete    = "/open-apis/im/v1/user_message_subscriptions/batch_delete"

	userReceiveResourceSenderUser = 1
	userReceiveResourceChat       = 2
	userReceiveResourceMentionMe  = 3
	userReceiveResourceP2PChat    = 4
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
	resourceType, err := parseUserReceiveResourceType(params["resource_type"])
	if err != nil {
		return err
	}
	params["resource_type"] = resourceTypeName(resourceType)

	ids := splitResourceIDs(params["resource_ids"])
	switch resourceType {
	case userReceiveResourceSenderUser:
		if len(ids) == 0 {
			return userReceiveParamError("resource_ids is required when resource_type=sender_user")
		}
		if err := validateResourceIDs(ids, "ou_"); err != nil {
			return err
		}
	case userReceiveResourceChat:
		if len(ids) == 0 {
			return userReceiveParamError("resource_ids is required when resource_type=chat")
		}
		if err := validateResourceIDs(ids, "oc_"); err != nil {
			return err
		}
	case userReceiveResourceMentionMe, userReceiveResourceP2PChat:
		// resource_ids are optional for these subscription modes.
	}
	if len(ids) > 10 {
		return userReceiveParamError("resource_ids exceeds the maximum of 10 (got %d)", len(ids))
	}
	sort.Strings(ids)
	if len(ids) > 0 {
		params["resource_ids"] = strings.Join(ids, ",")
	} else {
		delete(params, "resource_ids")
	}
	return nil
}

func messageUserReceivePreConsume(ctx context.Context, rt event.APIClient, params map[string]string) (func() error, error) {
	if rt == nil {
		return nil, errs.NewInternalError(errs.SubtypeUnknown,
			"runtime API client is required for pre-consume subscription")
	}
	body, err := buildMessageUserReceiveSubscriptionBody(params)
	if err != nil {
		return nil, err
	}
	raw, err := rt.CallAPI(ctx, "POST", pathMessageUserReceiveSubscribe, body)
	if err != nil {
		return nil, err
	}
	subscriptionIDs := parseMessageUserReceiveSubscriptionIDs(raw)
	if len(subscriptionIDs) == 0 {
		return func() error { return nil }, nil
	}

	return func() error {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := rt.CallAPI(cleanupCtx, "POST", pathMessageUserReceiveDelete, map[string]interface{}{
			"subscription_ids": subscriptionIDs,
		})
		return err
	}, nil
}

func buildMessageUserReceiveSubscriptionBody(params map[string]string) (map[string]interface{}, error) {
	resourceType, err := parseUserReceiveResourceType(params["resource_type"])
	if err != nil {
		return nil, err
	}
	body := map[string]interface{}{"resource_type": resourceType}
	if ids := splitResourceIDs(params["resource_ids"]); len(ids) > 0 {
		body["resource_ids"] = ids
	}
	return body, nil
}

func parseMessageUserReceiveSubscriptionIDs(raw json.RawMessage) []string {
	var resp struct {
		Data struct {
			Subscriptions []struct {
				SubscriptionID string `json:"subscription_id"`
			} `json:"subscriptions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil
	}
	ids := make([]string, 0, len(resp.Data.Subscriptions))
	for _, sub := range resp.Data.Subscriptions {
		if sub.SubscriptionID != "" {
			ids = append(ids, sub.SubscriptionID)
		}
	}
	return ids
}

func parseUserReceiveResourceType(value string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "mention_me":
		return userReceiveResourceMentionMe, nil
	case "sender_user":
		return userReceiveResourceSenderUser, nil
	case "chat":
		return userReceiveResourceChat, nil
	case "p2p_chat":
		return userReceiveResourceP2PChat, nil
	default:
		return 0, userReceiveParamError("invalid resource_type %q, allowed: sender_user, chat, mention_me, p2p_chat", value)
	}
}

func resourceTypeName(resourceType int) string {
	switch resourceType {
	case userReceiveResourceSenderUser:
		return "sender_user"
	case userReceiveResourceChat:
		return "chat"
	case userReceiveResourceMentionMe:
		return "mention_me"
	case userReceiveResourceP2PChat:
		return "p2p_chat"
	default:
		return ""
	}
}

func splitResourceIDs(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	ids := make([]string, 0, len(parts))
	for _, part := range parts {
		if id := strings.TrimSpace(part); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func validateResourceIDs(ids []string, prefix string) error {
	for _, id := range ids {
		if !strings.HasPrefix(id, prefix) {
			return userReceiveParamError("resource_id %q must be prefixed with %s", id, prefix)
		}
	}
	return nil
}

func userReceiveParamError(format string, args ...interface{}) error {
	return errs.NewValidationError(errs.SubtypeInvalidArgument, format, args...).
		WithParam("--param").
		WithHint("run `lark-cli event schema %s` for resource_type and resource_ids usage", eventTypeMessageUserReceive)
}
