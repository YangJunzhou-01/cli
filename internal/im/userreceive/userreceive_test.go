// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package userreceive

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestNormalizeParams_CanonicalizesAndSorts(t *testing.T) {
	params := map[string]string{
		FieldResourceType: "chat",
		FieldResourceIDs:  "oc_2, oc_1",
	}

	if err := NormalizeParams(params); err != nil {
		t.Fatalf("NormalizeParams() error = %v", err)
	}

	want := map[string]string{
		FieldResourceType: "chat",
		FieldResourceIDs:  "oc_1,oc_2",
	}
	if !reflect.DeepEqual(params, want) {
		t.Fatalf("params = %#v, want %#v", params, want)
	}
}

func TestBuildSubscriptionBody(t *testing.T) {
	params := map[string]string{
		FieldResourceType: "sender_user",
		FieldResourceIDs:  "ou_1,ou_2",
	}

	body, err := BuildSubscriptionBody(params)
	if err != nil {
		t.Fatalf("BuildSubscriptionBody() error = %v", err)
	}

	want := map[string]interface{}{
		FieldResourceType: ResourceSenderUser,
		FieldResourceIDs:  []string{"ou_1", "ou_2"},
	}
	if !reflect.DeepEqual(body, want) {
		t.Fatalf("body = %#v, want %#v", body, want)
	}
}

func TestValidateResourceIDs(t *testing.T) {
	tests := []struct {
		name         string
		resourceType int
		ids          []string
		wantField    string
	}{
		{
			name:         "sender user requires ids",
			resourceType: ResourceSenderUser,
			wantField:    FieldResourceIDs,
		},
		{
			name:         "chat rejects user id",
			resourceType: ResourceChat,
			ids:          []string{"ou_1"},
			wantField:    FieldResourceIDs,
		},
		{
			name:         "mention me allows empty ids",
			resourceType: ResourceMentionMe,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateResourceIDs(tt.resourceType, tt.ids)
			if tt.wantField == "" {
				if err != nil {
					t.Fatalf("ValidateResourceIDs() unexpected error = %v", err)
				}
				return
			}
			paramErr, ok := err.(*ParamError)
			if !ok {
				t.Fatalf("error = %T, want *ParamError", err)
			}
			if paramErr.Field != tt.wantField {
				t.Fatalf("Field = %q, want %q", paramErr.Field, tt.wantField)
			}
		})
	}
}

func TestParseSubscriptionIDs_NumberWireShape(t *testing.T) {
	raw := json.RawMessage(`{"code":0,"data":{"subscriptions":[{"subscription_id":7626963373590186951},{"subscription_id":7626963373590186952}]}}`)

	got := ParseSubscriptionIDs(raw)
	want := []json.Number{"7626963373590186951", "7626963373590186952"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseSubscriptionIDs() = %#v, want %#v", got, want)
	}
}
