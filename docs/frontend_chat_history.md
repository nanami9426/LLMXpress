# 前端接入文档：会话历史与续聊

本文说明网关新增的会话能力，包括：

- 聊天自动落库（会话 + 消息）
- 会话列表查询
- 会话消息查询
- 指定历史会话继续对话

> 注意：本次仅涉及大模型网关聊天（`/v1/chat/completions`），不涉及 `chat_ws` 私聊。

## 1. 新增了什么

### 1.1 聊天请求新增可选字段

在 `POST /v1/chat/completions` 的 JSON body 中，新增两个网关自定义字段（可选）：

- `conversation_id`：继续某个历史会话
- `new_chat`：`true` 时强制新建会话

网关会自动剥离这两个字段再转发给上游模型，所以不会影响上游 OpenAI 兼容接口。

### 1.2 聊天响应新增 Header

`POST /v1/chat/completions` 响应头会返回：

- `X-Conversation-ID: <会话ID>`

前端可以从这个 Header 取到会话 ID 并缓存，后续继续对话时回传。

### 1.3 新增查询接口

- `GET /v1/conversations`
- `GET /v1/conversations/:conversation_id/messages`

## 2. 鉴权要求

所有 `/v1/*` 接口都需要带 JWT：

```http
Authorization: Bearer <token>
```

## 3. /v1/chat/completions 用法

### 3.1 新建会话（推荐）

不传 `conversation_id` 即可创建新会话。

```json
{
  "model": "Qwen/Qwen2.5-1.5B-Instruct",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "你好，帮我规划一份学习计划"}
  ],
  "stream": false
}
```

### 3.2 强制新建会话

即使本地已有会话，也可传 `new_chat: true` 强制开新会话。

```json
{
  "model": "Qwen/Qwen2.5-1.5B-Instruct",
  "new_chat": true,
  "messages": [
    {"role": "user", "content": "这是一个全新话题"}
  ],
  "stream": false
}
```

### 3.3 继续历史会话（严格校验）

传 `conversation_id` 时，`messages` 只允许：

1. 仅 1 条 `user`
2. 或 2 条，且顺序必须是 `system` + `user`

否则网关会返回参数错误。

示例：

```json
{
  "model": "Qwen/Qwen2.5-1.5B-Instruct",
  "conversation_id": 19182736491827364,
  "messages": [
    {"role": "user", "content": "继续上次的话题，给我更具体的步骤"}
  ],
  "stream": true
}
```

### 3.4 会话上下文如何拼接

续聊时，前端只传“本轮输入”，网关会自动把该会话最近历史消息拼接到 `messages` 并转发。

前端不需要自己拼全量历史。

## 4. 新增查询接口

## 4.1 获取会话列表

`GET /v1/conversations?page=1&page_size=20`

分页规则：

- `page` 默认 `1`
- `page_size` 默认 `20`
- `page_size` 最大 `100`

响应示例：

```json
{
  "success": true,
  "data": {
    "list": [
      {
        "conversation_id": 19182736491827364,
        "title": "你好，帮我规划一份学习计划",
        "model": "Qwen/Qwen2.5-1.5B-Instruct",
        "message_count": 6,
        "last_message_preview": "可以按 4 周拆分目标执行……",
        "last_message_at": "2026-02-11T01:23:45.123456789Z"
      }
    ],
    "page": 1,
    "page_size": 20,
    "total": 12
  }
}
```

## 4.2 获取某个会话的消息

`GET /v1/conversations/:conversation_id/messages?page=1&page_size=50`

分页规则：

- `page` 默认 `1`
- `page_size` 默认 `50`
- `page_size` 最大 `200`

响应示例：

```json
{
  "success": true,
  "data": {
    "conversation": {
      "conversation_id": 19182736491827364,
      "title": "你好，帮我规划一份学习计划",
      "model": "Qwen/Qwen2.5-1.5B-Instruct",
      "message_count": 6,
      "last_message_preview": "可以按 4 周拆分目标执行……",
      "last_message_at": "2026-02-11T01:23:45.123456789Z"
    },
    "messages": [
      {
        "message_id": 19182736491820001,
        "role": "user",
        "content": "你好，帮我规划一份学习计划",
        "model": "Qwen/Qwen2.5-1.5B-Instruct",
        "created_at": "2026-02-11T01:20:00.000000000Z",
        "updated_at": "2026-02-11T01:20:00.000000000Z"
      },
      {
        "message_id": 19182736491820002,
        "role": "assistant",
        "content": "当然可以，我们先明确目标……",
        "model": "Qwen/Qwen2.5-1.5B-Instruct",
        "created_at": "2026-02-11T01:20:02.000000000Z",
        "updated_at": "2026-02-11T01:20:02.000000000Z"
      }
    ],
    "page": 1,
    "page_size": 50,
    "total": 6
  }
}
```

## 5. 前端建议接入流程

1. 发送 `POST /v1/chat/completions`（不传 `conversation_id`）。
2. 从响应头读取 `X-Conversation-ID`，写入当前会话状态。
3. 用户在会话列表中切换会话时，调用 `GET /v1/conversations/:id/messages` 拉历史并渲染。
4. 继续会话时，调用 `POST /v1/chat/completions`，传 `conversation_id` + 本轮消息。
5. 需要开启新会话时，清空当前 `conversation_id` 或传 `new_chat: true`。

## 6. fetch 示例

```ts
const API = "http://127.0.0.1:5000";

export async function chat({
  token,
  model,
  messages,
  conversationId,
  newChat = false,
  stream = false
}: {
  token: string;
  model: string;
  messages: Array<{role: "system" | "user"; content: string}>;
  conversationId?: number;
  newChat?: boolean;
  stream?: boolean;
}) {
  const body: Record<string, unknown> = { model, messages, stream };
  if (conversationId) body.conversation_id = conversationId;
  if (newChat) body.new_chat = true;

  const resp = await fetch(`${API}/v1/chat/completions`, {
    method: "POST",
    headers: {
      "Authorization": `Bearer ${token}`,
      "Content-Type": "application/json"
    },
    body: JSON.stringify(body)
  });

  const newConversationId = resp.headers.get("X-Conversation-ID");
  return { resp, conversationId: newConversationId ? Number(newConversationId) : undefined };
}
```

```ts
export async function listConversations(token: string, page = 1, pageSize = 20) {
  const resp = await fetch(
    `${API}/v1/conversations?page=${page}&page_size=${pageSize}`,
    { headers: { "Authorization": `Bearer ${token}` } }
  );
  return resp.json();
}

export async function listConversationMessages(token: string, conversationId: number, page = 1, pageSize = 50) {
  const resp = await fetch(
    `${API}/v1/conversations/${conversationId}/messages?page=${page}&page_size=${pageSize}`,
    { headers: { "Authorization": `Bearer ${token}` } }
  );
  return resp.json();
}
```

## 7. 错误处理注意事项

- 会话不存在/不属于当前用户：返回 `success=false`，错误码 `1004`（`not found`）。
- 参数错误（如续聊时 messages 格式不合规）：返回 `success=false`，错误码 `1001`。
- 建议前端统一优先判断：
  1. HTTP 状态码
  2. `success` 字段（若返回是网关统一格式）

