# cron-sidecar

> ⚠️ **實驗性專案** — 僅在 [agent-broker](https://github.com/thepagent/agent-broker) 環境下測試過，請自行評估風險。

為 [agent-broker](https://github.com/thepagent/agent-broker) 新增排程功能的 sidecar。以第二個 container 的形式運行在同一個 Kubernetes Pod 中，共用 PVC，可使用相同的 kiro-cli sessions 和 MCP 工具。

## 功能

### Discord Slash Commands

在 Discord 聊天輸入框輸入 `/` 即可使用以下指令：

- `/cron` — 透過表單建立排程任務，可設定：
  - 任務名稱
  - 執行頻率（cron 表達式，例如 `*/30 * * * *`）
  - 要 kiro 執行的 prompt
  - 使用的 AI model（選填）
  - 工作目錄（選填）
- `/cron-list` — 列出目前頻道的所有排程任務，並提供按鈕操作：
  - ⏸️ 暫停 / ▶️ 恢復
  - ✏️ 編輯
  - 🗑️ 刪除
- `/reminder` — 設定一次性提醒（例如 `+30m`、`14:00`、`明天 09:00`）

### Kiro Skill 整合

透過注入 steering 檔案，kiro 可在 Discord 對話中直接管理排程，無需使用 `/` 指令：

- 直接告訴 kiro「幫我建立一個每天早上 9 點的排程」
- kiro 會讀取並編輯排程設定檔（`cron.json`），cron-sidecar 會自動偵測變更並執行

排程執行時，會啟動一個臨時的 `kiro-cli` agent 執行 prompt，並將結果回傳到指定的 Discord 頻道。

## 前置需求

- 已透過 Helm 部署 [agent-broker](https://github.com/thepagent/agent-broker)
- agent-broker 與 cron-sidecar 共用 PVC
- Discord bot token（與 agent-broker 使用同一個 bot）

## 與 agent-broker 共用的資源

agent-broker 已建立的資源可直接複用：

| 資源 | 名稱 | 用途 |
|------|------|------|
| Secret | `agent-broker` | `discord-bot-token` 欄位 |
| PVC | `agent-broker` | 共用 `/home/agent` volume |

## 部署方式

### 步驟一：編譯 Docker image

```bash
git clone https://github.com/Sadivo/cron-sidecar.git
cd cron-sidecar
docker build -t cron-sidecar:latest .
```

### 步驟二：將 cron-sidecar 加入 agent-broker deployment

```bash
kubectl patch deployment agent-broker --type=json -p='[
  {"op":"add","path":"/spec/template/spec/containers/-","value":{
    "name": "cron-sidecar",
    "image": "cron-sidecar:latest",
    "imagePullPolicy": "Never",
    "env": [
      {"name":"DISCORD_BOT_TOKEN","valueFrom":{"secretKeyRef":{"name":"agent-broker","key":"discord-bot-token"}}},
      {"name":"HOME","value":"/home/agent"},
      {"name":"DATA_DIR","value":"/home/agent/.kiro/cron-data"},
      {"name":"KIRO_CLI_PATH","value":"/usr/local/bin/kiro-cli"},
      {"name":"DISCORD_GUILD_ID","value":"<your-guild-id>"},
      {"name":"CRON_TIMEZONE","value":"Asia/Taipei"},
      {"name":"BOT_LOCALE","value":"zh-TW"}
    ],
    "volumeMounts":[
      {"name":"data","mountPath":"/home/agent"}
    ]
  }}
]'
```

將 `<your-guild-id>` 替換為你的 Discord 伺服器 ID。

### 步驟三：注入 cron skill 給 kiro

將 steering 檔案複製到 PVC，讓 kiro 能透過對話管理排程任務：

```bash
kubectl cp steering/cron.md <pod-name>:/home/agent/.kiro/steering/cron.md -c agent-broker
```

完成後，kiro 可直接在 Discord 對話中建立、列出、啟停、刪除排程任務，不需要使用 `/cron` 指令。

### 步驟四：確認部署

```bash
kubectl get pods
# agent-broker pod 應顯示 2/2 Running

kubectl logs <pod-name> -c cron-sidecar
# 應顯示：cron-sidecar running
```

## 環境變數

| 變數 | 必填 | 預設值 | 說明 |
|------|------|--------|------|
| `DISCORD_BOT_TOKEN` | ✅ | — | 從 agent-broker secret 共用 |
| `DISCORD_GUILD_ID` | — | — | 限定特定伺服器。**建議設定** — 不設定時 slash commands 最多需要 1 小時才會出現（Global commands）；設定後立即生效（Guild commands）。 |
| `DATA_DIR` | — | `/home/agent/.kiro/cron-data` | 排程資料儲存目錄 |
| `KIRO_CLI_PATH` | — | `kiro-cli` | kiro-cli binary 路徑 |
| `CRON_TIMEZONE` | — | `Asia/Taipei` | Cron 表達式的時區 |
| `BOT_LOCALE` | — | `zh-TW` | Bot 語言（`zh-TW` 或 `en`） |

## MCP 整合

cron-sidecar 與 agent-broker 共用 PVC，因此它啟動的 kiro-cli agent 會自動載入相同的 MCP server（例如 [mcp-discord](https://github.com/Sadivo/mcp-discord)），排程 prompt 可直接使用 Discord 工具。

範例 prompt：
```
抓取頻道 1306423103659835443 的最新訊息，找出有日文 embed 的訊息，翻譯成繁體中文後回覆到該訊息下。
```

## 相關專案

- [agent-broker](https://github.com/thepagent/agent-broker) — 本 sidecar 所擴充的主要 Discord bot
- [mcp-discord](https://github.com/Sadivo/mcp-discord) — kiro-cli agent 使用的 Discord MCP server

---

## AI 助理部署提示詞

> 將以下內容貼給 AI 助理即可協助部署。

```
Please deploy https://github.com/Sadivo/cron-sidecar as a sidecar to the existing agent-broker Kubernetes deployment, reusing its secret and PVC. Refer to the README for details.
```
