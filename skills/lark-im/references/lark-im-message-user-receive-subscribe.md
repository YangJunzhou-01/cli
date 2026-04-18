# im +message-user-receive-subscribe

> **Prerequisite:** Read [`../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) first to understand authentication, global parameters, and safety rules.

Create a message receive subscription for user identity. This shortcut is **user-only** (`--as user`) and creates a user message subscription through `POST /open-apis/im/v1/user_message_subscriptions`.

## Commands

```bash
# Subscribe to messages that mention the current subscriber (default)
lark-cli im +message-user-receive-subscribe

# Subscribe to messages sent by specified users
lark-cli im +message-user-receive-subscribe \
  --resource-type sender_user \
  --resource-ids "ou_xxx,ou_yyy"

# Subscribe to messages in specified chats
lark-cli im +message-user-receive-subscribe \
  --resource-type chat \
  --resource-ids "oc_xxx,oc_yyy"

# Subscribe to mention_me explicitly
lark-cli im +message-user-receive-subscribe --resource-type mention_me

# Subscribe to p2p_chat
lark-cli im +message-user-receive-subscribe --resource-type p2p_chat

# JSON output
lark-cli im +message-user-receive-subscribe \
  --resource-type chat \
  --resource-ids "oc_xxx" \
  --format json

# Preview the request without creating anything
lark-cli im +message-user-receive-subscribe \
  --resource-type sender_user \
  --resource-ids "ou_xxx" \
  --dry-run
```

## Parameters

| Parameter | Required | Limits | Description |
|------|------|------|------|
| `--resource-type <type>` | No | `sender_user` / `chat` / `mention_me` / `p2p_chat` | Subscription resource type. Default is `mention_me` |
| `--resource-ids <ids>` | Required for `sender_user` / `chat`; optional for `mention_me` / `p2p_chat` | Up to 10 IDs | Comma-separated resource IDs. For `sender_user`, use user open_ids (`ou_xxx`). For `chat`, use chat open_ids (`oc_xxx`) |
| `--format <fmt>` | No | `json` (default) / `pretty` / `table` / `ndjson` / `csv` | Output format |
| `--as <identity>` | No | `user` only | Identity type |
| `--dry-run` | No | - | Preview the request without executing it |

## Resource Type Semantics

| Resource Type | Meaning | Typical Input |
|------|------|------|
| `sender_user` | Subscribe to messages sent by specified users | `--resource-ids "ou_xxx,ou_yyy"` |
| `chat` | Subscribe to messages in specified chats | `--resource-ids "oc_xxx,oc_yyy"` |
| `mention_me` | Subscribe to messages that mention the current subscriber | Usually no `--resource-ids` needed |
| `p2p_chat` | Subscribe to p2p messages associated with the current subscriber | Usually no `--resource-ids` needed |

## Output Fields

The response contains a `subscriptions` array. Each item typically includes:

| Field | Description |
|------|------|
| `subscription_id` | Subscription ID |
| `subscriber_id` | Subscriber ID |
| `resource_type` | Resource type enum value |
| `resource_id` | Resource ID |
| `status` | Subscription status |
| `create_time` | Creation timestamp |
| `update_time` | Last update timestamp |
| `version` | Record version |

In `pretty` mode, the shortcut prints the returned subscription rows as a table.

## AI Usage Guidance

1. Use `mention_me` when the user wants "messages that @ me".
2. Use `sender_user` when the user wants messages from a specific person or set of people.
3. Use `chat` when the user wants messages from specific chats or groups.
4. Prefer `--format json` if the result needs to be piped into another step.
5. If the user only gives a chat name, resolve the `chat_id` first with [`+chat-search`](lark-im-chat-search.md).

## Common Errors and Troubleshooting

| Symptom | Root Cause | Solution |
|---------|---------|---------|
| `--resource-ids is required for resource-type sender_user` | `sender_user` requires target user IDs | Provide one or more `ou_xxx` IDs |
| `--resource-ids is required for resource-type chat` | `chat` requires target chat IDs | Provide one or more `oc_xxx` IDs |
| `invalid user ID format` | A `sender_user` resource ID is not in `ou_xxx` format | Use user open_ids |
| `invalid chat ID format` | A `chat` resource ID is not in `oc_xxx` format | Use chat open_ids |
| `--resource-ids exceeds the maximum of 10` | Too many IDs were provided | Split into multiple requests |
| Permission denied | Missing `im:message.user_event_message:read` permission or missing user authorization | Enable the scope for the app and complete user auth |

## References

- [lark-im](../SKILL.md) - all IM commands
- [lark-im-chat-search](lark-im-chat-search.md) - resolve chat IDs before subscribing by chat
- [lark-shared](../../lark-shared/SKILL.md) - authentication and global parameters
