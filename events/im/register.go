// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

// Package im registers IM-domain EventKeys.
package im

import (
	"reflect"

	"github.com/larksuite/cli/internal/event"
)

// Keys returns all IM-domain EventKey definitions.
func Keys() []event.KeyDefinition {
	out := []event.KeyDefinition{
		{
			Key:         "im.message.receive_v1",
			DisplayName: "Receive message",
			Description: "Receive IM messages",
			EventType:   "im.message.receive_v1",
			Schema: event.SchemaDef{
				Custom: &event.SchemaSpec{Type: reflect.TypeOf(ImMessageReceiveOutput{})},
			},
			Process: processImMessageReceive,
			// Narrowest grant; kept single-element since MissingScopes uses AND semantics.
			Scopes:                []string{"im:message.p2p_msg:readonly"},
			AuthTypes:             []string{"bot"},
			RequiredConsoleEvents: []string{"im.message.receive_v1"},
		},
		{
			Key:         eventTypeMessageUserReceive,
			DisplayName: "Receive user message",
			Description: "Receive user-scoped IM message events; output includes message IDs only",
			EventType:   eventTypeMessageUserReceive,
			Params: []event.ParamDef{
				{
					Name:            "resource_type",
					Type:            event.ParamEnum,
					Default:         "mention_me",
					Description:     "Subscription resource type",
					SubscriptionKey: true,
					Values: []event.ParamValue{
						{Value: "sender_user", Desc: "Messages sent by the specified user open_ids in resource_ids"},
						{Value: "chat", Desc: "Messages in the specified chat open_ids in resource_ids"},
						{Value: "mention_me", Desc: "Messages that mention the current subscriber"},
						{Value: "p2p_chat", Desc: "P2P messages associated with the current subscriber"},
					},
				},
				{
					Name:            "resource_ids",
					Type:            event.ParamString,
					Description:     "Comma-separated resource IDs; required for sender_user (ou_xxx) and chat (oc_xxx); max 10",
					SubscriptionKey: true,
				},
			},
			Schema: event.SchemaDef{
				Custom: &event.SchemaSpec{Type: reflect.TypeOf(ImMessageUserReceiveOutput{})},
			},
			NormalizeParams:       normalizeMessageUserReceiveParams,
			Process:               processImMessageUserReceive,
			PreConsume:            messageUserReceivePreConsume,
			Scopes:                []string{"im:message.user_event_message:read"},
			AuthTypes:             []string{"user"},
			RequiredConsoleEvents: []string{eventTypeMessageUserReceive},
		},
	}

	for _, rk := range nativeIMKeys {
		out = append(out, event.KeyDefinition{
			Key:         rk.key,
			DisplayName: rk.title,
			Description: rk.description,
			EventType:   rk.key,
			Schema: event.SchemaDef{
				Native:         &event.SchemaSpec{Type: rk.bodyType},
				FieldOverrides: rk.fieldOverrides,
			},
			Scopes:                rk.scopes,
			AuthTypes:             []string{"bot"},
			RequiredConsoleEvents: []string{rk.key},
		})
	}

	return out
}
