# cron-sidecar

> ⚠️ **Experimental** — This project has been tested as an add-on for [agent-broker](https://github.com/thepagent/agent-broker) only. Use at your own risk.

A Discord bot sidecar that adds cron scheduling capabilities to [agent-broker](https://github.com/thepagent/agent-broker). Runs as a second container in the same Kubernetes Pod, sharing the same PVC so it can use the same kiro-cli sessions and MCP tools.

## Features

- `/cron` — Create a scheduled task via Discord modal (name, cron expression, prompt, model, working directory)
- `/cron-list` — List, enable/disable, edit, or delete scheduled tasks
- `/reminder` — Set a one-shot reminder

When a scheduled task runs, it spawns a temporary `kiro-cli` agent, executes the prompt, and posts the result back to the specified Discord channel.

## Prerequisites

- [agent-broker](https://github.com/thepagent/agent-broker) deployed on Kubernetes via Helm
- Shared PVC between agent-broker and cron-sidecar
- Discord bot token (same bot as agent-broker)

## Shared Resources with agent-broker

The following resources are already created by agent-broker and can be reused:

| Resource | Name | Used for |
|----------|------|----------|
| Secret | `agent-broker` | `discord-bot-token` key |
| PVC | `agent-broker` | Shared `/home/agent` volume |

## Deployment

### Step 1: Build the Docker image

```bash
git clone https://github.com/Sadivo/cron-sidecar.git
cd cron-sidecar
docker build -t cron-sidecar:latest .
```

### Step 2: Add cron-sidecar to the agent-broker deployment

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

Replace `<your-guild-id>` with your Discord server ID.

### Step 3: Inject the cron skill for kiro

Copy the steering file to the agent's PVC so kiro knows how to manage cron jobs via conversation:

```bash
kubectl cp steering/cron.md <pod-name>:/home/agent/.kiro/steering/cron.md -c agent-broker
```

After this, kiro can create, list, enable/disable, and delete scheduled tasks directly from Discord conversation — without needing the `/cron` slash command.

### Step 4: Verify

```bash
kubectl get pods
# Should show 2/2 Running for agent-broker pod

kubectl logs <pod-name> -c cron-sidecar
# Should show: cron-sidecar running
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DISCORD_BOT_TOKEN` | ✅ | — | Shared from agent-broker secret |
| `DISCORD_GUILD_ID` | — | — | Restrict to specific guild. **Recommended** — without it, slash commands take up to 1 hour to appear in Discord (Global commands). With it, commands appear instantly (Guild commands). |
| `DATA_DIR` | — | `/home/agent/.kiro/cron-data` | Directory for cron job storage |
| `KIRO_CLI_PATH` | — | `kiro-cli` | Path to kiro-cli binary |
| `CRON_TIMEZONE` | — | `Asia/Taipei` | Timezone for cron expressions |
| `BOT_LOCALE` | — | `zh-TW` | Bot language (`zh-TW` or `en`) |

## MCP Integration

Since cron-sidecar shares the PVC with agent-broker, the kiro-cli agents it spawns automatically load the same MCP servers (e.g. [mcp-discord](https://github.com/Sadivo/mcp-discord)). This means cron prompts can use Discord tools directly.

Example cron prompt:
```
抓取頻道 1306423103659835443 的最新訊息，找出有日文 embed 的訊息，翻譯成繁體中文後回覆到該訊息下。
```

## Related Projects

- [agent-broker](https://github.com/thepagent/agent-broker) — The main Discord bot this sidecar extends
- [mcp-discord](https://github.com/Sadivo/mcp-discord) — Discord MCP server used by kiro-cli agents

---

## AI Assistant Deployment Guide

> Paste the following to an AI assistant to deploy this project.

```
Please deploy https://github.com/Sadivo/cron-sidecar as a sidecar to the existing agent-broker Kubernetes deployment, reusing its secret and PVC. Refer to the README for details.
```
