// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package im

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/larksuite/cli/shortcuts/common"
	"github.com/spf13/cobra"
)

func newMessageUserReceiveSubscribeRuntime(t *testing.T, stringFlags map[string]string, rt http.RoundTripper) *common.RuntimeContext {
	t.Helper()

	runtime := newUserShortcutRuntime(t, rt)
	cmd := &cobra.Command{Use: "test"}
	for _, name := range []string{"resource-type", "resource-ids", "user-id-type"} {
		cmd.Flags().String(name, "", "")
	}
	if err := cmd.ParseFlags(nil); err != nil {
		t.Fatalf("ParseFlags() error = %v", err)
	}
	for name, value := range stringFlags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("Flags().Set(%q) error = %v", name, err)
		}
	}
	runtime.Cmd = cmd
	runtime.Format = "json"
	return runtime
}

func TestBuildMessageUserReceiveSubscribeRequest(t *testing.T) {
	runtime := newTestRuntimeContext(t, map[string]string{
		"resource-type": "sender_user",
		"resource-ids":  "ou_1, ou_2",
		"user-id-type":  "union_id",
	}, nil)

	body, params, err := buildMessageUserReceiveSubscribeRequest(runtime)
	if err != nil {
		t.Fatalf("buildMessageUserReceiveSubscribeRequest() error = %v", err)
	}

	wantBody := map[string]interface{}{
		"resource_type": 1,
		"resource_ids":  []string{"ou_1", "ou_2"},
	}
	if !reflect.DeepEqual(body, wantBody) {
		t.Fatalf("body = %#v, want %#v", body, wantBody)
	}
	if got := params.Get("user_id_type"); got != "union_id" {
		t.Fatalf("user_id_type = %q, want union_id", got)
	}
}

func TestMessageUserReceiveSubscribeValidate(t *testing.T) {
	tests := []struct {
		name    string
		flags   map[string]string
		wantErr string
	}{
		{
			name: "mention_me without resource ids is allowed",
			flags: map[string]string{
				"resource-type": "mention_me",
				"user-id-type":  "open_id",
			},
		},
		{
			name: "sender_user without resource ids is allowed",
			flags: map[string]string{
				"resource-type": "sender_user",
				"user-id-type":  "open_id",
			},
		},
		{
			name: "chat rejects more than ten resource ids",
			flags: map[string]string{
				"resource-type": "chat",
				"resource-ids":  "oc_1,oc_2,oc_3,oc_4,oc_5,oc_6,oc_7,oc_8,oc_9,oc_10,oc_11",
				"user-id-type":  "open_id",
			},
			wantErr: "--resource-ids exceeds the maximum of 10",
		},
		{
			name: "p2p_chat with explicit resource ids is allowed",
			flags: map[string]string{
				"resource-type": "p2p_chat",
				"resource-ids":  "ou_1",
				"user-id-type":  "open_id",
			},
		},
		{
			name: "unknown resource type fails",
			flags: map[string]string{
				"resource-type": "unknown",
				"user-id-type":  "open_id",
			},
			wantErr: "invalid --resource-type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtime := newTestRuntimeContext(t, tt.flags, nil)
			err := ImMessageUserReceiveSubscribe.Validate(context.Background(), runtime)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() unexpected error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestMessageUserReceiveSubscribeDryRun(t *testing.T) {
	runtime := newTestRuntimeContext(t, map[string]string{
		"resource-type": "mention_me",
		"user-id-type":  "open_id",
	}, nil)

	got := mustMarshalDryRun(t, ImMessageUserReceiveSubscribe.DryRun(context.Background(), runtime))
	if !strings.Contains(got, `"/open-apis/v1/user_message_subscriptions"`) ||
		!strings.Contains(got, `"user_id_type":"open_id"`) ||
		!strings.Contains(got, `"resource_type":3`) {
		t.Fatalf("DryRun() = %s", got)
	}
}

func TestMessageUserReceiveSubscribeDryRun_WithResourceIDs(t *testing.T) {
	runtime := newTestRuntimeContext(t, map[string]string{
		"resource-type": "p2p_chat",
		"resource-ids":  "ou_1",
		"user-id-type":  "open_id",
	}, nil)

	got := mustMarshalDryRun(t, ImMessageUserReceiveSubscribe.DryRun(context.Background(), runtime))
	if !strings.Contains(got, `"resource_type":4`) || !strings.Contains(got, `"resource_ids":["ou_1"]`) {
		t.Fatalf("DryRun() = %s", got)
	}
}

func TestMessageUserReceiveSubscribeExecute(t *testing.T) {
	var capturedPath, capturedQuery string
	var capturedBody map[string]interface{}

	runtime := newMessageUserReceiveSubscribeRuntime(t, map[string]string{
		"resource-type": "chat",
		"resource-ids":  "oc_1",
		"user-id-type":  "open_id",
	}, shortcutRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.Path, "/open-apis/v1/user_message_subscriptions") {
			return nil, fmt.Errorf("unexpected path: %s", req.URL.Path)
		}
		capturedPath = req.URL.Path
		capturedQuery = req.URL.RawQuery
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &capturedBody); err != nil {
			return nil, err
		}
		return shortcutJSONResponse(200, map[string]interface{}{
			"code": 0,
			"msg":  "ok",
			"data": map[string]interface{}{
				"subscriptions": []map[string]interface{}{
					{
						"subscription_id": "sub_1",
						"resource_type":   2,
						"resource_id":     "oc_1",
						"status":          1,
					},
				},
			},
		}), nil
	}))

	if err := ImMessageUserReceiveSubscribe.Execute(context.Background(), runtime); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if capturedPath != "/open-apis/v1/user_message_subscriptions" {
		t.Fatalf("path = %q", capturedPath)
	}
	if !strings.Contains(capturedQuery, "user_id_type=open_id") {
		t.Fatalf("query = %q, want user_id_type=open_id", capturedQuery)
	}
	if got := capturedBody["resource_type"]; got != float64(2) {
		t.Fatalf("resource_type = %#v, want 2", got)
	}
	if got := capturedBody["resource_ids"]; !reflect.DeepEqual(got, []interface{}{"oc_1"}) {
		t.Fatalf("resource_ids = %#v, want [oc_1]", got)
	}
}
