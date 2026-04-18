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
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
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
	Description: "Create a user message receive subscription",
	Risk:        "write",
	Scopes:      []string{"im:message.user_event_message:read"},
	AuthTypes:   []string{"user"},
	HasFormat:   true,
	Flags: []common.Flag{
		{
			Name:    "resource-type",
			Default: "mention_me",
			Desc:    "subscription resource type",
			Enum:    []string{"sender_user", "chat", "mention_me", "p2p_chat"},
		},
		{
			Name: "resource-ids",
			Desc: "comma-separated resource open IDs; required for sender_user/chat and omitted for mention_me/p2p_chat",
		},
		{
			Name:    "user-id-type",
			Default: "open_id",
			Desc:    "user ID type used by the API",
			Enum:    []string{"open_id", "user_id", "union_id"},
		},
	},
	DryRun: func(ctx context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
		body, params, err := buildMessageUserReceiveSubscribeRequest(runtime)
		if err != nil {
			return common.NewDryRunAPI().Set("error", err.Error())
		}
		return common.NewDryRunAPI().
			POST(imMessageUserReceiveSubscribePath).
			Params(map[string]interface{}{"user_id_type": params.Get("user_id_type")}).
			Body(body)
	},
	Validate: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return validateMessageUserReceiveSubscribe(runtime)
	},
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		body, params, err := buildMessageUserReceiveSubscribeRequest(runtime)
		if err != nil {
			return err
		}

		resData, err := runtime.DoAPIJSON(http.MethodPost, imMessageUserReceiveSubscribePath, params, body)
		if err != nil {
			return err
		}

		outData := map[string]interface{}{
			"subscriptions": resData["subscriptions"],
		}
		runtime.OutFormat(outData, nil, func(w io.Writer) {
			fmt.Fprintln(w, "Message user receive subscription created successfully")
			if subscriptions, ok := outData["subscriptions"].([]interface{}); ok {
				output.PrintTable(w, []map[string]interface{}{
					{"subscriptions": len(subscriptions)},
				})
			}
		})
		return nil
	},
}

func buildMessageUserReceiveSubscribeRequest(runtime *common.RuntimeContext) (map[string]interface{}, larkcore.QueryParams, error) {
	resourceType, err := parseMessageUserReceiveResourceType(runtime.Str("resource-type"))
	if err != nil {
		return nil, nil, err
	}
	userIDType := runtime.Str("user-id-type")
	if strings.TrimSpace(userIDType) == "" {
		userIDType = "open_id"
	}

	body := map[string]interface{}{
		"resource_type": resourceType,
	}
	if ids := common.SplitCSV(runtime.Str("resource-ids")); len(ids) > 0 {
		body["resource_ids"] = ids
	}

	params := larkcore.QueryParams{"user_id_type": []string{userIDType}}
	return body, params, nil
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
	case messageUserReceiveResourceMentionMe, messageUserReceiveResourceP2PChat:
		if len(resourceIDs) > 0 {
			return output.ErrValidation("--resource-ids is not supported for resource-type %s", resourceTypeName(resourceType))
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
