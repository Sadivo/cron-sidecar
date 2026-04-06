package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/nczz/cron-sidecar/acp"
	"github.com/nczz/cron-sidecar/heartbeat"
	L "github.com/nczz/cron-sidecar/locale"
)

type Config struct {
	Token    string
	KiroCLI  string
	DataDir  string
	GuildID  string
	Timezone string
}

type Bot struct {
	ds        *discordgo.Session
	cfg       Config
	cronStore *heartbeat.CronStore
	hb        *heartbeat.Heartbeat
	hbCancel  context.CancelFunc
	agents    sync.Map // name → *acp.Agent
}

func New(cfg Config) (*Bot, error) {
	ds, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, err
	}
	ds.Identify.Intents = discordgo.IntentsGuilds

	store, err := heartbeat.NewCronStore(cfg.DataDir)
	if err != nil {
		return nil, err
	}

	b := &Bot{ds: ds, cfg: cfg, cronStore: store}

	hb := heartbeat.New(60)
	hb.Register(heartbeat.NewCronTask(store, b, cfg.DataDir, cfg.Timezone, cfg.GuildID))
	b.hb = hb

	return b, nil
}

func (b *Bot) Start(ctx context.Context) error {
	b.ds.AddHandler(b.onInteraction)

	if err := b.ds.Open(); err != nil {
		return err
	}

	if err := b.registerCommands(); err != nil {
		return err
	}

	hbCtx, cancel := context.WithCancel(ctx)
	b.hbCancel = cancel
	go b.hb.Start(hbCtx)

	return nil
}

func (b *Bot) Stop() {
	if b.hbCancel != nil {
		b.hbCancel()
	}
	b.agents.Range(func(k, v any) bool {
		v.(*acp.Agent).Stop()
		return true
	})
	b.ds.Close()
}

// --- CronDeps interface ---

func (b *Bot) StartTempAgent(name, cwd, model string) (*acp.Agent, error) {
	if cwd == "" {
		cwd = "/home/agent"
	}
	agent, err := acp.StartAgent(name, b.cfg.KiroCLI, cwd, model)
	if err != nil {
		return nil, err
	}
	b.agents.Store(name, agent)
	return agent, nil
}

func (b *Bot) StopTempAgent(agent *acp.Agent) {
	b.agents.Delete(agent.Name)
	agent.Stop()
}

func (b *Bot) AskAgentStream(ctx context.Context, agent *acp.Agent, prompt string) (string, string, error) {
	var fullLog strings.Builder
	resp, err := agent.Ask(ctx, prompt, func(chunk string) {
		fullLog.WriteString(chunk)
	})
	return resp, fullLog.String(), err
}

func (b *Bot) Notify(channelID, msg string) {
	_, _ = b.ds.ChannelMessageSend(channelID, msg)
}

// --- Discord slash commands ---

func (b *Bot) registerCommands() error {
	cmds := []*discordgo.ApplicationCommand{
		{
			Name:        "cron",
			Description: "建立定時排程任務",
		},
		{
			Name:        "cron-list",
			Description: "列出並管理排程任務",
		},
		{
			Name:        "reminder",
			Description: "設定一次性提醒",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "time", Description: "時間 (e.g. +30m, 14:00, 明天 09:00)", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "message", Description: "提醒內容", Required: true},
			},
		},
	}
	for _, cmd := range cmds {
		if _, err := b.ds.ApplicationCommandCreate(b.ds.State.User.ID, b.cfg.GuildID, cmd); err != nil {
			return fmt.Errorf("register %s: %w", cmd.Name, err)
		}
	}
	log.Println("[bot] slash commands registered")
	return nil
}

func (b *Bot) onInteraction(ds *discordgo.Session, i *discordgo.InteractionCreate) {
	if b.cfg.GuildID != "" && i.GuildID != b.cfg.GuildID {
		return
	}
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		switch i.ApplicationCommandData().Name {
		case "cron":
			b.handleCronModal(ds, i)
		case "cron-list":
			b.handleCronList(ds, i)
		case "reminder":
			b.handleReminder(ds, i)
		}
	case discordgo.InteractionModalSubmit:
		id := i.ModalSubmitData().CustomID
		switch {
		case id == "cron_add_modal":
			b.handleCronModalSubmit(ds, i)
		case strings.HasPrefix(id, "cron_edit_modal:"):
			b.handleCronEditSubmit(ds, i, strings.TrimPrefix(id, "cron_edit_modal:"))
		}
	case discordgo.InteractionMessageComponent:
		cid := i.MessageComponentData().CustomID
		switch {
		case strings.HasPrefix(cid, "cron_toggle:"):
			b.handleCronToggle(ds, i, strings.TrimPrefix(cid, "cron_toggle:"))
		case strings.HasPrefix(cid, "cron_delete:"):
			b.handleCronDelete(ds, i, strings.TrimPrefix(cid, "cron_delete:"))
		case strings.HasPrefix(cid, "cron_edit:"):
			b.handleCronEditModal(ds, i, strings.TrimPrefix(cid, "cron_edit:"))
		}
	}
}

func (b *Bot) handleCronModal(ds *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = ds.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "cron_add_modal",
			Title:    L.Get("cron.modal.title"),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "cron_name", Label: L.Get("cron.modal.name"), Style: discordgo.TextInputShort, Placeholder: L.Get("cron.modal.name_ph"), Required: true, MaxLength: 100},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "cron_schedule", Label: L.Get("cron.modal.schedule"), Style: discordgo.TextInputShort, Placeholder: L.Get("cron.modal.schedule_ph"), Required: true, MaxLength: 100},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "cron_prompt", Label: L.Get("cron.modal.prompt"), Style: discordgo.TextInputParagraph, Placeholder: L.Get("cron.modal.prompt_ph"), Required: true, MaxLength: 2000},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "cron_cwd", Label: L.Get("cron.modal.cwd"), Style: discordgo.TextInputShort, Placeholder: L.Get("cron.modal.cwd_ph"), Required: false, MaxLength: 200},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "cron_model", Label: L.Get("cron.modal.model"), Style: discordgo.TextInputShort, Placeholder: L.Get("cron.modal.model_ph"), Required: false, MaxLength: 100},
				}},
			},
		},
	})
}

func (b *Bot) handleCronModalSubmit(ds *discordgo.Session, i *discordgo.InteractionCreate) {
	fields := modalFields(i)
	cronExpr, err := heartbeat.ParseSchedule(fields["cron_schedule"])
	if err != nil {
		respond(ds, i, L.Getf("error.parse_schedule", err.Error()))
		return
	}
	job := &heartbeat.CronJob{
		Name: fields["cron_name"], ChannelID: i.ChannelID, GuildID: i.GuildID,
		Schedule: cronExpr, ScheduleHuman: fields["cron_schedule"],
		Prompt: fields["cron_prompt"], CWD: fields["cron_cwd"], Model: fields["cron_model"],
		HistoryLimit: 10, Enabled: true, UseAgent: true,
		CreatedBy: memberName(i),
	}
	if err := b.cronStore.Add(job); err != nil {
		respond(ds, i, L.Getf("error.save_failed", err.Error()))
		return
	}
	respond(ds, i, L.Getf("cron.created", job.Name, cronExpr, heartbeat.DescribeSchedule(cronExpr), job.Prompt))
}

func (b *Bot) handleCronEditModal(ds *discordgo.Session, i *discordgo.InteractionCreate, jobID string) {
	job, ok := b.cronStore.Get(jobID)
	if !ok {
		respond(ds, i, L.Get("cron.not_found"))
		return
	}
	_ = ds.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "cron_edit_modal:" + jobID,
			Title:    L.Get("cron.modal.title_edit"),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "cron_name", Label: L.Get("cron.modal.name"), Style: discordgo.TextInputShort, Value: job.Name, Required: true, MaxLength: 100},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "cron_schedule", Label: L.Get("cron.modal.schedule"), Style: discordgo.TextInputShort, Value: job.ScheduleHuman, Required: true, MaxLength: 100},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "cron_prompt", Label: L.Get("cron.modal.prompt"), Style: discordgo.TextInputParagraph, Value: job.Prompt, Required: true, MaxLength: 2000},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "cron_cwd", Label: L.Get("cron.modal.cwd"), Style: discordgo.TextInputShort, Value: job.CWD, Required: false, MaxLength: 200},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "cron_model", Label: L.Get("cron.modal.model"), Style: discordgo.TextInputShort, Value: job.Model, Required: false, MaxLength: 100},
				}},
			},
		},
	})
}

func (b *Bot) handleCronEditSubmit(ds *discordgo.Session, i *discordgo.InteractionCreate, jobID string) {
	job, ok := b.cronStore.Get(jobID)
	if !ok {
		respond(ds, i, L.Get("cron.not_found"))
		return
	}
	fields := modalFields(i)
	cronExpr, err := heartbeat.ParseSchedule(fields["cron_schedule"])
	if err != nil {
		respond(ds, i, L.Getf("error.parse_schedule", err.Error()))
		return
	}
	job.Name = fields["cron_name"]
	job.Schedule = cronExpr
	job.ScheduleHuman = fields["cron_schedule"]
	job.Prompt = fields["cron_prompt"]
	job.CWD = fields["cron_cwd"]
	job.Model = fields["cron_model"]
	if err := b.cronStore.Update(job); err != nil {
		respond(ds, i, L.Getf("error.save_failed", err.Error()))
		return
	}
	respond(ds, i, L.Getf("cron.updated", job.Name, cronExpr, heartbeat.DescribeSchedule(cronExpr), job.Prompt))
}

func (b *Bot) handleCronList(ds *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = ds.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	jobs := b.cronStore.ListByChannel(i.ChannelID)
	if len(jobs) == 0 {
		followup(ds, i, L.Get("cron.list.empty"))
		return
	}
	for _, job := range jobs {
		status := "✅"
		if !job.Enabled {
			status = "⏸️"
		}
		toggleLabel := L.Get("cron.btn.pause")
		if !job.Enabled {
			toggleLabel = L.Get("cron.btn.resume")
		}
		nextRun := job.NextRun
		if nextRun == "" {
			nextRun = L.Get("cron.list.next_run_pending")
		} else if t, err := time.Parse(time.RFC3339, nextRun); err == nil {
			nextRun = t.Format("01/02 15:04")
		}
		content := fmt.Sprintf("%s **%s**\n`%s` (%s)\n下次執行：%s\n> %s",
			status, job.Name, job.Schedule, heartbeat.DescribeSchedule(job.Schedule), nextRun, job.Prompt)
		_, _ = ds.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: content,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{Label: toggleLabel, Style: discordgo.SecondaryButton, CustomID: "cron_toggle:" + job.ID},
					discordgo.Button{Label: L.Get("cron.btn.edit"), Style: discordgo.PrimaryButton, CustomID: "cron_edit:" + job.ID},
					discordgo.Button{Label: L.Get("cron.btn.delete"), Style: discordgo.DangerButton, CustomID: "cron_delete:" + job.ID},
				}},
			},
		})
	}
}

func (b *Bot) handleCronToggle(ds *discordgo.Session, i *discordgo.InteractionCreate, jobID string) {
	job, ok := b.cronStore.Get(jobID)
	if !ok {
		respond(ds, i, L.Get("cron.not_found"))
		return
	}
	job.Enabled = !job.Enabled
	_ = b.cronStore.Update(job)
	statusMsg := L.Get("cron.resumed")
	if !job.Enabled {
		statusMsg = L.Get("cron.paused")
	}
	respond(ds, i, fmt.Sprintf(statusMsg, job.Name))
}

func (b *Bot) handleCronDelete(ds *discordgo.Session, i *discordgo.InteractionCreate, jobID string) {
	job, ok := b.cronStore.Get(jobID)
	if !ok {
		respond(ds, i, L.Get("cron.not_found"))
		return
	}
	_ = b.cronStore.Remove(jobID)
	respond(ds, i, L.Getf("cron.deleted", job.Name))
}

func (b *Bot) handleReminder(ds *discordgo.Session, i *discordgo.InteractionCreate) {
	opts := i.ApplicationCommandData().Options
	timeStr, msg := opts[0].StringValue(), opts[1].StringValue()

	loc, _ := time.LoadLocation(b.cfg.Timezone)
	t, err := heartbeat.ParseTime(timeStr, loc)
	if err != nil {
		respond(ds, i, err.Error())
		return
	}

	// Convert to one-shot cron: run at specific minute/hour
	cronExpr := fmt.Sprintf("%d %d %d %d *", t.Minute(), t.Hour(), t.Day(), int(t.Month()))
	mentionID := ""
	if i.Member != nil && i.Member.User != nil {
		mentionID = i.Member.User.ID
	}
	job := &heartbeat.CronJob{
		Name: "reminder", ChannelID: i.ChannelID, GuildID: i.GuildID,
		Schedule: cronExpr, Prompt: msg, OneShot: true, UseAgent: false,
		MentionID: mentionID, Enabled: true, HistoryLimit: 1,
		CreatedBy: memberName(i),
	}
	if err := b.cronStore.Add(job); err != nil {
		respond(ds, i, L.Getf("error.save_failed", err.Error()))
		return
	}
	respond(ds, i, fmt.Sprintf("⏰ 提醒已設定：%s\n> %s", t.Format("01/02 15:04"), msg))
}

// --- helpers ---

func respond(ds *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	_ = ds.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: msg},
	})
}

func followup(ds *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	_, _ = ds.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{Content: msg})
}

func modalFields(i *discordgo.InteractionCreate) map[string]string {
	fields := map[string]string{}
	for _, row := range i.ModalSubmitData().Components {
		if ar, ok := row.(*discordgo.ActionsRow); ok {
			for _, comp := range ar.Components {
				if ti, ok := comp.(*discordgo.TextInput); ok {
					fields[ti.CustomID] = ti.Value
				}
			}
		}
	}
	return fields
}

func memberName(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.Username
	}
	return ""
}
