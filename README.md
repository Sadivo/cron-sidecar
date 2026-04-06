# cron-sidecar

> ⚠️ **Experimental** — This project has been tested as an add-on for [agent-broker](https://github.com/thepagent/agent-broker) only. Use at your own risk.

A Discord bot sidecar that adds cron scheduling capabilities to [agent-broker](https://github.com/thepagent/agent-broker). Runs as a second container in the same Kubernetes Pod, sharing the same PVC so it can use the same kiro-cli sessions and MCP tools.

## Features

- `/cron` — Create a scheduled task via Discord modal (name, cron expression, prompt, model, working directory)
- `/cron-list` — List, enable/disable, edit, or delete scheduled tasks
- `/reminder` — Set a one-shot reminder

When a scheduled task runs, it spawns a temporary `kiro-cli` agent, executes the prompt, and posts the result back to the specified Discord channel.

## Prerequisites

- [agent-broker](https://github.com/thepagent/agent-broker) deployed on Kubernetes
- Shared PVC between agent-broker and cron-sidecar
- Discord bot token (same bot as agent-broker)
- Discord Guild ID

## Deployment

### Step 1: Build the Docker image

```bash
git clone https://github.com/Sadivo/cron-sidecar.git
cd cron-sidecar
docker build -t cron-sidecar:latest .
```

### Step 2: Add cron-sidecar to the agent-broker deployment

Patch the existing `agent-broker` deployment to add a second container:

```bash
kubectl patch deployment agent-broker --type=json -p='[
  {"op":"add","path":"/spec/template/spec/containers/-","value":{
    "name": "cron-sidecar",
    "image": "cron-sidecar:latest",
    "imagePullPolicy": "Never",
    "env": [
      {"name":"DISCORD_BOT_TOKEN","valueFrom":{"secretKeyRef":{"name":"<your-secret>","key":"discord-bot-token"}}},
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

Replace `<your-secret>` and `<your-guild-id>` with your actual values.

### Step 3: Verify

```bash
kubectl get pods
# Should show 2/2 Running for agent-broker pod

kubectl logs <pod-name> -c cron-sidecar
# Should show: cron-sidecar running
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DISCORD_BOT_TOKEN` | ✅ | — | Discord bot token |
| `DISCORD_GUILD_ID` | — | — | Restrict to specific guild |
| `DATA_DIR` | — | `/home/agent/.kiro/cron-data` | Directory for cron job storage |
| `KIRO_CLI_PATH` | — | `kiro-cli` | Path to kiro-cli binary |
| `CRON_TIMEZONE` | — | `Asia/Taipei` | Timezone for cron expressions |
| `BOT_LOCALE` | — | `zh-TW` | Bot language (`zh-TW` or `en`) |

## MCP Integration

Since cron-sidecar shares the PVC with agent-broker, the kiro-cli agents it spawns automatically load the same MCP servers (e.g. `mcp-discord`). This means cron prompts can use Discord tools directly.

Example cron prompt:
```
抓取頻道 1306423103659835443 的最新訊息，找出有日文 embed 的訊息，翻譯成繁體中文後回覆到該訊息下。
```

## Related Projects

- [agent-broker](https://github.com/thepagent/agent-broker) — The main Discord bot this sidecar extends
- [mcp-discord](https://github.com/Sadivo/mcp-discord) — Discord MCP server used by kiro-cli agents
