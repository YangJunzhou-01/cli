// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package im

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/larksuite/cli/internal/output"
	"github.com/larksuite/cli/shortcuts/common"
)

const imMessageUserReceiveSubscribePath = "/open-apis/im/v1/user_message_subscriptions"

const (
	messageUserReceiveResourceSenderUser = 1
	messageUserReceiveResourceChat       = 2
	messageUserReceiveResourceMentionMe  = 3
	messageUserReceiveResourceP2PChat    = 4
)

var ImMessageUserReceiveSubscribe = common.Shortcut{
	Service:     "im",
	Command:     "+message-user-receive-subscribe",
	Description: "Create a message receive subscription; user-only; supports sender_user/chat/mention_me/p2p_chat resource types",
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
			POST(imMessageUserReceiveSubscribePath).
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

		resData, err := runtime.DoAPIJSON(http.MethodPost, imMessageUserReceiveSubscribePath, nil, body)
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
	resourceType, err := parseMessageUserReceiveResourceType(runtime.Str("resource-type"))
	if err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"resource_type": resourceType,
	}
	if ids := common.SplitCSV(runtime.Str("resource-ids")); len(ids) > 0 {
		body["resource_ids"] = ids
	}

	return body, nil
}

func validateMessageUserReceiveSubscribe(runtime *common.RuntimeContext) error {
	resourceType, err := parseMessageUserReceiveResourceType(runtime.Str("resource-type"))
	if err != nil {
		return err
	}

	resourceIDs := common.SplitCSV(runtime.Str("resource-ids"))
	switch resourceType {
	case messageUserReceiveResourceSenderUser, messageUserReceiveResourceChat:
		if len(resourceIDs) == 0 {
			return output.ErrValidation("--resource-ids is required for resource-type %s", resourceTypeName(resourceType))
		}
		if len(resourceIDs) > 10 {
			return output.ErrValidation("--resource-ids exceeds the maximum of 10 (got %d)", len(resourceIDs))
		}
		for _, resourceID := range resourceIDs {
			switch resourceType {
			case messageUserReceiveResourceSenderUser:
				if _, err := common.ValidateUserID(resourceID); err != nil {
					return err
				}
			case messageUserReceiveResourceChat:
				if _, err := common.ValidateChatID(resourceID); err != nil {
					return err
				}
			}
		}
	case messageUserReceiveResourceMentionMe, messageUserReceiveResourceP2PChat:
		if len(resourceIDs) > 10 {
			return output.ErrValidation("--resource-ids exceeds the maximum of 10 (got %d)", len(resourceIDs))
		}
	}
	return nil
}

func parseMessageUserReceiveResourceType(value string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "mention_me":
		return messageUserReceiveResourceMentionMe, nil
	case "sender_user":
		return messageUserReceiveResourceSenderUser, nil
	case "chat":
		return messageUserReceiveResourceChat, nil
	case "p2p_chat":
		return messageUserReceiveResourceP2PChat, nil
	default:
		return 0, output.ErrValidation("invalid --resource-type %q, allowed: sender_user, chat, mention_me, p2p_chat", value)
	}
}

func resourceTypeName(resourceType int) string {
	switch resourceType {
	case messageUserReceiveResourceSenderUser:
		return "sender_user"
	case messageUserReceiveResourceChat:
		return "chat"
	case messageUserReceiveResourceMentionMe:
		return "mention_me"
	case messageUserReceiveResourceP2PChat:
		return "p2p_chat"
	default:
		return fmt.Sprintf("%d", resourceType)
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
