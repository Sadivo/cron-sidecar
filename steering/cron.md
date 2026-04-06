---
name: cron
description: Use when the user wants to create, list, enable, disable, or delete scheduled tasks (cron jobs).
---

# Cron Scheduling

Scheduled tasks are managed by the `cron-sidecar` container and stored at:
`/home/agent/.kiro/cron-data/cron/cron.json`

## How to Create a Cron Job

⚠️ **IMPORTANT**: Always read the existing file first, then merge your new entry into it. Never overwrite the entire file.

```
1. fs_read /home/agent/.kiro/cron-data/cron/cron.json
2. Parse the JSON object
3. Add your new entry to the object
4. fs_write the entire merged object back
```

### JSON Schema

```json
{
  "<8-char-hex-id>": {
    "id": "<8-char-hex-id>",
    "name": "任務名稱",
    "channel_id": "<discord-channel-id>",
    "guild_id": "<discord-guild-id>",
    "schedule": "*/30 * * * *",
    "schedule_human": "*/30 * * * *",
    "prompt": "要 agent 執行的任務描述",
    "cwd": "",
    "model": "",
    "history_limit": 10,
    "enabled": true,
    "one_shot": false,
    "use_agent": true,
    "created_by": "kiro",
    "created_at": "<RFC3339 timestamp>",
    "last_run": "",
    "next_run": ""
  }
}
```

### Field Notes

- `id`: 8-char random hex, e.g. `a1b2c3d4`
- `schedule`: standard 5-field cron expression (`分 時 日 月 星期`)
- `use_agent`: always `true` to run via kiro-cli agent
- `one_shot`: set `true` for one-time reminders (job auto-deletes after running)
- `model`: leave empty to use default, or specify e.g. `claude-haiku-4.5`
- `channel_id`: the Discord channel where results will be posted

### Common Schedules

| Expression | Meaning |
|------------|---------|
| `*/30 * * * *` | Every 30 minutes |
| `0 9 * * *` | Daily at 09:00 |
| `0 9 * * 1-5` | Weekdays at 09:00 |
| `0 9,18 * * *` | Daily at 09:00 and 18:00 |

## How to List / Edit / Delete

⚠️ **IMPORTANT**: Always read the existing file first, then merge changes. Never overwrite the entire file with only your changes.

Read and modify `/home/agent/.kiro/cron-data/cron/cron.json` directly using `fs_read` and `fs_write`.

To disable a job: set `"enabled": false`
To delete a job: remove the entry from the JSON object
