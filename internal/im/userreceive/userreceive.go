// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

// Package userreceive contains shared IM user message subscription rules.
package userreceive

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	EventType = "im.message.user_receive_v1"

	SubscribePath = "/open-apis/im/v1/user_message_subscriptions"
	DeletePath    = "/open-apis/im/v1/user_message_subscriptions/batch_delete"

	FieldResourceType = "resource_type"
	FieldResourceIDs  = "resource_ids"
)

const (
	ResourceSenderUser = 1
	ResourceChat       = 2
	ResourceMentionMe  = 3
	ResourceP2PChat    = 4
)

type ParamError struct {
	Field   string
	Message string
}

func (e *ParamError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func ParseResourceType(value string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "mention_me":
		return ResourceMentionMe, nil
	case "sender_user":
		return ResourceSenderUser, nil
	case "chat":
		return ResourceChat, nil
	case "p2p_chat":
		return ResourceP2PChat, nil
	default:
		return 0, paramError(FieldResourceType,
			"invalid resource_type %q, allowed: sender_user, chat, mention_me, p2p_chat", value)
	}
}

func ResourceTypeName(resourceType int) string {
	switch resourceType {
	case ResourceSenderUser:
		return "sender_user"
	case ResourceChat:
		return "chat"
	case ResourceMentionMe:
		return "mention_me"
	case ResourceP2PChat:
		return "p2p_chat"
	default:
		return fmt.Sprintf("%d", resourceType)
	}
}

func NormalizeParams(params map[string]string) error {
	resourceType, err := ParseResourceType(params[FieldResourceType])
	if err != nil {
		return err
	}
	params[FieldResourceType] = ResourceTypeName(resourceType)

	ids := SplitResourceIDs(params[FieldResourceIDs])
	if err := ValidateResourceIDs(resourceType, ids); err != nil {
		return err
	}
	sort.Strings(ids)
	if len(ids) > 0 {
		params[FieldResourceIDs] = strings.Join(ids, ",")
	} else {
		delete(params, FieldResourceIDs)
	}
	return nil
}

func BuildSubscriptionBody(params map[string]string) (map[string]interface{}, error) {
	resourceType, err := ParseResourceType(params[FieldResourceType])
	if err != nil {
		return nil, err
	}
	body := map[string]interface{}{FieldResourceType: resourceType}
	if ids := SplitResourceIDs(params[FieldResourceIDs]); len(ids) > 0 {
		body[FieldResourceIDs] = ids
	}
	return body, nil
}

func SplitResourceIDs(value string) []string {
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

func ValidateResourceIDs(resourceType int, ids []string) error {
	switch resourceType {
	case ResourceSenderUser:
		if len(ids) == 0 {
			return paramError(FieldResourceIDs, "resource_ids is required when resource_type=sender_user")
		}
		if err := validateResourceIDPrefixes(ids, "ou_", "invalid user ID format, should start with 'ou_' (e.g., ou_abc123)"); err != nil {
			return err
		}
	case ResourceChat:
		if len(ids) == 0 {
			return paramError(FieldResourceIDs, "resource_ids is required when resource_type=chat")
		}
		if err := validateResourceIDPrefixes(ids, "oc_", "invalid chat ID format, should start with 'oc_' (e.g., oc_abc123)"); err != nil {
			return err
		}
	case ResourceMentionMe, ResourceP2PChat:
		// resource_ids are optional for these subscription modes.
	}
	if len(ids) > 10 {
		return paramError(FieldResourceIDs, "resource_ids exceeds the maximum of 10 (got %d)", len(ids))
	}
	return nil
}

func ParseSubscriptionIDs(raw json.RawMessage) []json.Number {
	var resp struct {
		Data struct {
			Subscriptions []struct {
				SubscriptionID json.Number `json:"subscription_id"`
			} `json:"subscriptions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil
	}
	ids := make([]json.Number, 0, len(resp.Data.Subscriptions))
	for _, sub := range resp.Data.Subscriptions {
		if string(sub.SubscriptionID) != "" {
			ids = append(ids, sub.SubscriptionID)
		}
	}
	return ids
}

func validateResourceIDPrefixes(ids []string, prefix, message string) error {
	for _, id := range ids {
		if !strings.HasPrefix(id, prefix) {
			return paramError(FieldResourceIDs, "%s", message)
		}
	}
	return nil
}

func paramError(field, format string, args ...interface{}) error {
	return &ParamError{
		Field:   field,
		Message: fmt.Sprintf(format, args...),
	}
}
