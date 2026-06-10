// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package im

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/larksuite/cli/internal/event"
)

func TestMain(m *testing.M) {
	for _, k := range Keys() {
		event.RegisterKey(k)
	}
	os.Exit(m.Run())
}

func TestIMKeys_ProcessedReceiveRegistered(t *testing.T) {
	def, ok := event.Lookup("im.message.receive_v1")
	if !ok {
		t.Fatal("im.message.receive_v1 should be registered via Keys()")
	}
	if def.Schema.Custom == nil {
		t.Error("Processed key must set Schema.Custom")
	}
	if def.Schema.Native != nil {
		t.Error("Processed key must not set Schema.Native")
	}
	if def.Process == nil {
		t.Error("Process must not be nil for Processed key")
	}
	if len(def.Scopes) == 0 {
		t.Error("Scopes must not be empty — preflightScopes would bypass validation")
	}
}

func TestIMKeys_UserReceiveRegistered(t *testing.T) {
	def, ok := event.Lookup("im.message.user_receive_v1")
	if !ok {
		t.Fatal("im.message.user_receive_v1 should be registered via Keys()")
	}
	if def.Schema.Custom == nil {
		t.Error("Processed key must set Schema.Custom")
	}
	if def.Schema.Native != nil {
		t.Error("Processed key must not set Schema.Native")
	}
	if def.Process == nil {
		t.Error("Process must not be nil for Processed key")
	}
	if def.PreConsume == nil {
		t.Error("PreConsume must not be nil for user subscription setup")
	}
	if len(def.AuthTypes) != 1 || def.AuthTypes[0] != "user" {
		t.Errorf("AuthTypes = %v, want [user]", def.AuthTypes)
	}
	if len(def.RequiredConsoleEvents) != 1 || def.RequiredConsoleEvents[0] != "im.message.user_receive_v1" {
		t.Errorf("RequiredConsoleEvents = %v", def.RequiredConsoleEvents)
	}
}

func TestIMKeys_NativeEventsRegistered(t *testing.T) {
	want := []string{
		"im.message.message_read_v1",
		"im.message.reaction.created_v1",
		"im.message.reaction.deleted_v1",
		"im.chat.member.bot.added_v1",
		"im.chat.member.bot.deleted_v1",
		"im.chat.member.user.added_v1",
		"im.chat.member.user.withdrawn_v1",
		"im.chat.member.user.deleted_v1",
		"im.chat.updated_v1",
		"im.chat.disbanded_v1",
	}
	for _, k := range want {
		def, ok := event.Lookup(k)
		if !ok {
			t.Errorf("%s should be registered via Keys()", k)
			continue
		}
		if def.Schema.Native == nil {
			t.Errorf("%s: Schema.Native must be set for native key", k)
		}
		if def.Schema.Custom != nil {
			t.Errorf("%s: Native key must not set Schema.Custom", k)
		}
		if def.Process != nil {
			t.Errorf("%s: Native key must not set Process", k)
		}
		if def.Schema.Native != nil && def.Schema.Native.Type == nil {
			t.Errorf("%s: Schema.Native.Type must reference an SDK type", k)
		}
	}
}

func TestProcessImMessageUserReceive_MessageIDOnly(t *testing.T) {
	payload := `{
		"schema": "2.0",
		"header": {
			"event_id": "ev_user_text",
			"event_type": "im.message.user_receive_v1",
			"create_time": "1776409469275",
			"app_id": "cli_test"
		},
		"event": {
			"message_id": "om_user_001",
			"message": {
				"message_id": "om_legacy_body",
				"content": "{\"text\":\"should not be exposed\"}"
			}
		}
	}`
	out := runUserReceive(t, payload)

	if out.Type != "im.message.user_receive_v1" {
		t.Errorf("Type = %q", out.Type)
	}
	if out.EventID != "ev_user_text" {
		t.Errorf("EventID = %q", out.EventID)
	}
	if out.MessageID != "om_user_001" || out.ID != "om_user_001" {
		t.Errorf("MessageID/ID = %q/%q, want om_user_001", out.MessageID, out.ID)
	}
	if out.Timestamp != "1776409469275" {
		t.Errorf("Timestamp = %q", out.Timestamp)
	}
}

func TestProcessImMessageUserReceive_LegacyNestedMessageID(t *testing.T) {
	payload := `{
		"schema": "2.0",
		"header": {
			"event_id": "ev_user_legacy",
			"event_type": "im.message.user_receive_v1"
		},
		"event": {
			"message": {
				"message_id": "om_nested_only",
				"content": "{\"text\":\"should not be exposed\"}"
			}
		}
	}`
	out := runUserReceive(t, payload)

	if out.MessageID != "om_nested_only" {
		t.Errorf("MessageID = %q, want om_nested_only", out.MessageID)
	}
}

func TestProcessImMessageUserReceive_MalformedPayload(t *testing.T) {
	raw := &event.RawEvent{
		EventID:   "ev_bad_user",
		EventType: "im.message.user_receive_v1",
		Payload:   json.RawMessage(`not json`),
		Timestamp: time.Now(),
	}
	got, err := processImMessageUserReceive(context.Background(), nil, raw, nil)
	if err != nil {
		t.Fatalf("Process should swallow parse errors, got %v", err)
	}
	if string(got) != "not json" {
		t.Errorf("malformed fallback output = %q, want original bytes", string(got))
	}
}

func TestMessageUserReceivePreConsume_SubscribeAndCleanup(t *testing.T) {
	rt := &recordingAPIClient{
		responses: []json.RawMessage{
			json.RawMessage(`{"code":0,"data":{"subscriptions":[{"subscription_id":"sub_1"},{"subscription_id":"sub_2"}]}}`),
			json.RawMessage(`{"code":0,"data":{}}`),
		},
	}
	params := map[string]string{
		"resource_type": "chat",
		"resource_ids":  "oc_2, oc_1",
	}
	if err := normalizeMessageUserReceiveParams(context.Background(), rt, params); err != nil {
		t.Fatalf("normalize params: %v", err)
	}

	cleanup, err := messageUserReceivePreConsume(context.Background(), rt, params)
	if err != nil {
		t.Fatalf("preconsume: %v", err)
	}
	if cleanup == nil {
		t.Fatal("cleanup should not be nil")
	}
	if len(rt.calls) != 1 {
		t.Fatalf("calls after subscribe = %d, want 1", len(rt.calls))
	}
	subscribe := rt.calls[0]
	if subscribe.method != "POST" || subscribe.path != pathMessageUserReceiveSubscribe {
		t.Fatalf("subscribe call = %s %s", subscribe.method, subscribe.path)
	}
	wantSubBody := map[string]interface{}{
		"resource_type": userReceiveResourceChat,
		"resource_ids":  []string{"oc_1", "oc_2"},
	}
	if !reflect.DeepEqual(subscribe.body, wantSubBody) {
		t.Fatalf("subscribe body = %#v, want %#v", subscribe.body, wantSubBody)
	}

	if err := cleanup(); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if len(rt.calls) != 2 {
		t.Fatalf("calls after cleanup = %d, want 2", len(rt.calls))
	}
	deleted := rt.calls[1]
	if deleted.method != "POST" || deleted.path != pathMessageUserReceiveDelete {
		t.Fatalf("delete call = %s %s", deleted.method, deleted.path)
	}
	wantDeleteBody := map[string]interface{}{
		"subscription_ids": []string{"sub_1", "sub_2"},
	}
	if !reflect.DeepEqual(deleted.body, wantDeleteBody) {
		t.Fatalf("delete body = %#v, want %#v", deleted.body, wantDeleteBody)
	}
}

func TestNormalizeMessageUserReceiveParams_Validation(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]string
		wantErr bool
	}{
		{
			name:   "mention_me default",
			params: map[string]string{},
		},
		{
			name: "sender_user requires ids",
			params: map[string]string{
				"resource_type": "sender_user",
			},
			wantErr: true,
		},
		{
			name: "sender_user rejects non user ids",
			params: map[string]string{
				"resource_type": "sender_user",
				"resource_ids":  "oc_chat",
			},
			wantErr: true,
		},
		{
			name: "chat rejects non chat ids",
			params: map[string]string{
				"resource_type": "chat",
				"resource_ids":  "ou_user",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := normalizeMessageUserReceiveParams(context.Background(), nil, tt.params)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestProcessImMessageReceive_Text(t *testing.T) {
	payload := `{
		"schema": "2.0",
		"header": {
			"event_id": "ev_test_text",
			"event_type": "im.message.receive_v1",
			"create_time": "1776409469273",
			"app_id": "cli_test"
		},
		"event": {
			"sender": {
				"sender_id": {"open_id": "ou_sender"}
			},
			"message": {
				"message_id":   "om_text_001",
				"chat_id":      "oc_chat",
				"chat_type":    "p2p",
				"message_type": "text",
				"create_time":  "1776409468987",
				"content":      "{\"text\":\"hello there\"}"
			}
		}
	}`
	out := runReceive(t, payload)

	if out.Type != "im.message.receive_v1" {
		t.Errorf("Type = %q", out.Type)
	}
	if out.MessageID != "om_text_001" || out.ID != "om_text_001" {
		t.Errorf("MessageID/ID = %q/%q", out.MessageID, out.ID)
	}
	if out.ChatType != "p2p" || out.ChatID != "oc_chat" {
		t.Errorf("chat_id/chat_type = %q/%q", out.ChatID, out.ChatType)
	}
	if out.SenderID != "ou_sender" {
		t.Errorf("SenderID = %q", out.SenderID)
	}
	if out.Content != "hello there" {
		t.Errorf("Content = %q, want \"hello there\"", out.Content)
	}
	if out.Timestamp != "1776409469273" {
		t.Errorf("Timestamp = %q", out.Timestamp)
	}
}

func TestProcessImMessageReceive_Interactive(t *testing.T) {
	payload := `{
		"schema": "2.0",
		"header": {
			"event_id": "ev_test_card",
			"event_type": "im.message.receive_v1",
			"create_time": "1776409469274",
			"app_id": "cli_test"
		},
		"event": {
			"sender": {
				"sender_id": {"open_id": "ou_sender"}
			},
			"message": {
				"message_id":   "om_card_001",
				"chat_id":      "oc_chat",
				"chat_type":    "group",
				"message_type": "interactive",
				"create_time":  "1776409468987",
				"content":      "{\"header\":{\"title\":{\"tag\":\"plain_text\",\"content\":\"A card\"}}}"
			}
		}
	}`
	out := runReceive(t, payload)

	if out.Type != "im.message.receive_v1" {
		t.Errorf("Type = %q", out.Type)
	}
	if out.MessageType != "interactive" {
		t.Errorf("MessageType = %q", out.MessageType)
	}
	if out.ChatType != "group" {
		t.Errorf("ChatType = %q", out.ChatType)
	}
}

func TestProcessImMessageReceive_MalformedPayload(t *testing.T) {
	raw := &event.RawEvent{
		EventID:   "ev_bad",
		EventType: "im.message.receive_v1",
		Payload:   json.RawMessage(`not json`),
		Timestamp: time.Now(),
	}
	got, err := processImMessageReceive(context.Background(), nil, raw, nil)
	if err != nil {
		t.Fatalf("Process should swallow parse errors, got %v", err)
	}
	if string(got) != "not json" {
		t.Errorf("malformed fallback output = %q, want original bytes", string(got))
	}
}

func runReceive(t *testing.T, payload string) ImMessageReceiveOutput {
	t.Helper()
	raw := &event.RawEvent{
		EventID:   "ev_test",
		EventType: "im.message.receive_v1",
		Payload:   json.RawMessage(payload),
		Timestamp: time.Now(),
	}
	got, err := processImMessageReceive(context.Background(), nil, raw, nil)
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}
	var out ImMessageReceiveOutput
	if err := json.Unmarshal(got, &out); err != nil {
		t.Fatalf("Process output is not valid ImMessageReceiveOutput JSON: %v\nraw=%s", err, string(got))
	}
	return out
}

type recordedCall struct {
	method string
	path   string
	body   interface{}
}

type recordingAPIClient struct {
	calls     []recordedCall
	responses []json.RawMessage
}

func (r *recordingAPIClient) CallAPI(_ context.Context, method, path string, body interface{}) (json.RawMessage, error) {
	r.calls = append(r.calls, recordedCall{method: method, path: path, body: body})
	if len(r.responses) == 0 {
		return json.RawMessage(`{}`), nil
	}
	resp := r.responses[0]
	r.responses = r.responses[1:]
	return resp, nil
}

var _ event.APIClient = (*recordingAPIClient)(nil)

func runUserReceive(t *testing.T, payload string) ImMessageUserReceiveOutput {
	t.Helper()
	raw := &event.RawEvent{
		EventID:   "ev_test",
		EventType: "im.message.user_receive_v1",
		Payload:   json.RawMessage(payload),
		Timestamp: time.Now(),
	}
	got, err := processImMessageUserReceive(context.Background(), nil, raw, nil)
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}
	var out ImMessageUserReceiveOutput
	if err := json.Unmarshal(got, &out); err != nil {
		t.Fatalf("Process output is not valid ImMessageUserReceiveOutput JSON: %v\nraw=%s", err, string(got))
	}
	return out
}
