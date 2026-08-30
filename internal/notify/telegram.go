package notify

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Sender interface {
	Send(ctx context.Context, text string) error
}

type Telegram struct {
	Token  string
	ChatID string
	Client *http.Client
}

func NewTelegram(token, chatID string) *Telegram {
	return &Telegram{
		Token:  token,
		ChatID: chatID,
		Client: &http.Client{Timeout: 20 * time.Second},
	}
}

func (t *Telegram) Send(ctx context.Context, text string) error {
	if t.Token == "" || t.ChatID == "" {
		return fmt.Errorf("telegram not configured: set telegram.bot_token and telegram.chat_id")
	}
	endpoint := "https://api.telegram.org/bot" + t.Token + "/sendMessage"
	form := url.Values{}
	form.Set("chat_id", t.ChatID)
	form.Set("text", text)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := t.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram: FAILED (HTTP %d)", resp.StatusCode)
	}
	return nil
}

type LogSender struct {
	Inner Sender
	Log   func(string, ...any)
}

func (l LogSender) Send(ctx context.Context, text string) error {
	err := l.Inner.Send(ctx, text)
	if l.Log == nil {
		return err
	}
	if err != nil {
		l.Log("telegram: %v", err)
		return err
	}
	l.Log("telegram: sent")
	return nil
}
