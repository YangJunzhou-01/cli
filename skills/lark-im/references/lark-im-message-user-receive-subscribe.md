# im +message-user-receive-subscribe

> **Prerequisite:** Read [`../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) first to understand authentication, global parameters, and safety rules.

Create a **persistent server-side** message receive subscription for user identity. This shortcut is **user-only** (`--as user`) and creates a user message subscription through `POST /open-apis/im/v1/user_message_subscriptions`.

Use this shortcut when the user explicitly needs a subscription relationship that remains on the server after the CLI command exits. It does **not** consume events, and it does **not** automatically delete the subscription.

User message delivery has two layers:

| Layer | CLI surface | Purpose |
|------|------|------|
| Server-side subscription relationship | `im +message-user-receive-subscribe` and `im user_message_subscription ...` | Create/list/delete which user-message resources should emit events |
| Event consumption process | `event consume im.message.user_receive_v1` | Keep a local process connected and stream received event notifications |

Even for persistent subscriptions created by this shortcut, a consumer still needs to run, for example:

```bash
lark-cli event consume im.message.user_receive_v1 --as user \
  -p resource_type=chat \
  -p resource_ids=oc_xxx
```

`event consume` emits only `message_id` / `id` for each notification. Fetch message details explicitly with `im +messages-mget` when content is needed:

```bash
lark-cli event consume im.message.user_receive_v1 --as user --max-events 1 --jq '.message_id'
lark-cli im +messages-mget --as user --message-ids om_xxx
```

## Pick the Right Flow

| Need | Command | Notes |
|------|------|------|
| Quick bounded listen/test | `lark-cli event consume im.message.user_receive_v1 --as user ... --max-events 1 --timeout 10m` | The event command can manage a short-lived subscribe/consume/cleanup flow |
| Long-lived server-side subscription | `lark-cli im +message-user-receive-subscribe ...` | Creates the persistent relationship only; still run `event consume` or another consumer to receive events |
| Long-running CLI listener | `lark-cli event consume im.message.user_receive_v1 --as user ...` | Keep the process alive; fetch message details with `im +messages-mget` |
| Inspect existing persistent subscriptions | `lark-cli im user_message_subscription batch_query --as user ...` | Read-only |
| Delete leaked or no-longer-needed subscriptions | `lark-cli im user_message_subscription batch_delete --as user ...` | Deletes by `subscription_id` |

Do not treat `+message-user-receive-subscribe` as a listener. It only creates the server-side subscription record.

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

## Persistent Subscription Lifecycle

### 1. Create

```bash
lark-cli im +message-user-receive-subscribe \
  --as user \
  --resource-type chat \
  --resource-ids "oc_xxx" \
  --format json
```

Save the returned `subscription_id`; it is required for deletion.

### 2. List

Use the raw metadata API command to find active subscriptions:

```bash
# list all current user message subscriptions
lark-cli im user_message_subscription batch_query \
  --as user \
  --data '{"page_size":"100"}' \
  --format json

# filter by chat subscription
lark-cli im user_message_subscription batch_query \
  --as user \
  --data '{"resource_type":2,"resource_id":"oc_xxx","status":1,"page_size":"100"}' \
  --format json
```

`resource_type` values:

| Value | Meaning |
|------:|------|
| `1` | `sender_user` |
| `2` | `chat` |
| `3` | `mention_me` |
| `4` | `p2p_chat` |

### 3. Delete

Delete persistent subscriptions with the raw metadata API command:

```bash
lark-cli im user_message_subscription batch_delete \
  --as user \
  --data '{"subscription_ids":[7626963373590186951]}' \
  --format json
```

`subscription_ids` are numeric IDs returned by create/list. Pass 1 to 20 IDs per request.

## Event Consumption

Use `event consume` to listen for user message subscription events. For short-lived automation, `event consume` can manage a bounded subscribe/consume/cleanup flow:

```bash
# listen for one message in a chat, then auto-clean
lark-cli event consume im.message.user_receive_v1 \
  --as user \
  -p resource_type=chat \
  -p resource_ids=oc_xxx \
  --max-events 1 \
  --timeout 10m
```

Normal completion paths such as `--max-events`, `--timeout`, Ctrl+C/SIGTERM, and stdin EOF trigger cleanup for the subscription managed by that `event consume` run. Crashes, `kill -9`, or host termination can skip cleanup and leave an orphan subscription. If that happens, list with `batch_query` and delete with `batch_delete`.

For persistent server-side subscriptions created with `+message-user-receive-subscribe`, keep an event consumer running for actual delivery:

```bash
lark-cli event consume im.message.user_receive_v1 \
  --as user \
  -p resource_type=chat \
  -p resource_ids=oc_xxx
```

Do not rely on `event consume` cleanup to remove subscriptions created earlier by this shortcut. Delete persistent records explicitly with `im user_message_subscription batch_delete`.

## AI Usage Guidance

1. Use `event consume im.message.user_receive_v1 --as user` as the CLI listener/consumer.
2. Use `mention_me` when the user wants "messages that @ me".
3. Use `sender_user` when the user wants messages from a specific person or set of people.
4. Use `chat` when the user wants messages from specific chats or groups.
5. Prefer `--format json` if the result needs to be piped into another step.
6. If the user only gives a chat name, resolve the `chat_id` first with [`+chat-search`](lark-im-chat-search.md).

## Common Errors and Troubleshooting

| Symptom | Root Cause | Solution |
|---------|---------|---------|
| `--resource-ids is required for resource-type sender_user` | `sender_user` requires target user IDs | Provide one or more `ou_xxx` IDs |
| `--resource-ids is required for resource-type chat` | `chat` requires target chat IDs | Provide one or more `oc_xxx` IDs |
| `invalid user ID format` | A `sender_user` resource ID is not in `ou_xxx` format | Use user open_ids |
| `invalid chat ID format` | A `chat` resource ID is not in `oc_xxx` format | Use chat open_ids |
| `--resource-ids exceeds the maximum of 10` | Too many IDs were provided | Split into multiple requests |
| Permission denied | Missing `im:message.user_event_message:read` permission or missing user authorization | Enable the scope for the app and complete user auth |
| Duplicate delivery or unexpected events after a local listener exited | A persistent or leaked server-side subscription still exists | Run `im user_message_subscription batch_query`, then delete stale `subscription_id` values with `batch_delete` |
| Need to stop a subscription created by this shortcut | Shortcut has no paired unsubscribe command | Use `im user_message_subscription batch_delete --as user --data '{"subscription_ids":[...]}'` |

## References

- [lark-im](../SKILL.md) - all IM commands
- [lark-im-chat-search](lark-im-chat-search.md) - resolve chat IDs before subscribing by chat
- [lark-event IM events](../../lark-event/references/lark-event-im.md) - `event consume im.message.user_receive_v1` listener flow
- [lark-shared](../../lark-shared/SKILL.md) - authentication and global parameters
