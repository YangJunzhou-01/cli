// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package im

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/larksuite/cli/internal/im/userreceive"
	"github.com/larksuite/cli/internal/output"
	"github.com/larksuite/cli/shortcuts/common"
)

var ImMessageUserReceiveSubscribe = common.Shortcut{
	Service:     "im",
	Command:     "+message-user-receive-subscribe",
	Description: "Create a persistent server-side user message subscription; user-only; supports sender_user/chat/mention_me/p2p_chat resource types",
	Risk:        "write",
	Scopes:      []string{"im:message.user_event_message:read"},
	AuthTypes:   []string{"user"},
	HasFormat:   true,
	Flags: []common.Flag{
		{
			Name:    "resource-type",
			Default: "mention_me",
			Desc:    "subscription resource type: sender_user means messages sent by specified users; chat means messages in specified chats; mention_me means messages that mention the current subscriber; p2p_chat means p2p messages associated with the current subscriber",
			Enum:    []string{"sender_user", "chat", "mention_me", "p2p_chat"},
		},
		{
			Name: "resource-ids",
			Desc: "comma-separated resource IDs (max 10); for sender_user, use user open_ids (ou_xxx); for chat, use chat open_ids (oc_xxx); required for sender_user/chat and optional for mention_me/p2p_chat",
		},
	},
	DryRun: func(ctx context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
		body, err := buildMessageUserReceiveSubscribeRequest(runtime)
		if err != nil {
			return common.NewDryRunAPI().Set("error", err.Error())
		}
		return common.NewDryRunAPI().
			POST(userreceive.SubscribePath).
			Body(body)
	},
	Validate: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return validateMessageUserReceiveSubscribe(runtime)
	},
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		body, err := buildMessageUserReceiveSubscribeRequest(runtime)
		if err != nil {
			return err
		}

		resData, err := runtime.DoAPIJSONTyped(http.MethodPost, userreceive.SubscribePath, nil, body)
		if err != nil {
			return err
		}

		outData := map[string]interface{}{
			"subscriptions": resData["subscriptions"],
		}
		runtime.OutFormat(outData, nil, func(w io.Writer) {
			fmt.Fprintln(w, "Message user receive subscription created successfully")
			if rows := subscriptionRows(outData["subscriptions"]); len(rows) > 0 {
				output.PrintTable(w, rows)
			}
		})
		return nil
	},
}

func buildMessageUserReceiveSubscribeRequest(runtime *common.RuntimeContext) (map[string]interface{}, error) {
	params := messageUserReceiveSubscribeParams(runtime)
	if err := userreceive.NormalizeParams(params); err != nil {
		return nil, messageUserReceiveSubscribeParamError(err)
	}
	body, err := userreceive.BuildSubscriptionBody(params)
	if err != nil {
		return nil, messageUserReceiveSubscribeParamError(err)
	}
	return body, nil
}

func validateMessageUserReceiveSubscribe(runtime *common.RuntimeContext) error {
	params := messageUserReceiveSubscribeParams(runtime)
	if err := userreceive.NormalizeParams(params); err != nil {
		return messageUserReceiveSubscribeParamError(err)
	}
	return nil
}

func messageUserReceiveSubscribeParams(runtime *common.RuntimeContext) map[string]string {
	return map[string]string{
		userreceive.FieldResourceType: runtime.Str("resource-type"),
		userreceive.FieldResourceIDs:  runtime.Str("resource-ids"),
	}
}

func messageUserReceiveSubscribeParamError(err error) error {
	if err == nil {
		return nil
	}
	param := ""
	var paramErr *userreceive.ParamError
	if errors.As(err, &paramErr) {
		param = shortcutMessageUserReceiveParamName(paramErr.Field)
	}
	message := strings.NewReplacer(
		"resource_type", "--resource-type",
		"resource_ids", "--resource-ids",
	).Replace(err.Error())
	validationErr := common.ValidationErrorf("%s", message)
	if param != "" {
		validationErr.WithParam(param)
	}
	return validationErr
}

func shortcutMessageUserReceiveParamName(field string) string {
	switch field {
	case userreceive.FieldResourceType:
		return "--resource-type"
	case userreceive.FieldResourceIDs:
		return "--resource-ids"
	default:
		return ""
	}
}

func subscriptionRows(raw interface{}) []map[string]interface{} {
	subscriptions, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	rows := make([]map[string]interface{}, 0, len(subscriptions))
	for _, item := range subscriptions {
		row, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		rows = append(rows, row)
	}
	return rows
}
