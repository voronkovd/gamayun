package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/voronkovd/gamayun/internal/checks"
	"github.com/voronkovd/gamayun/internal/config"
	"github.com/voronkovd/gamayun/internal/notify"
	"github.com/voronkovd/gamayun/internal/service"
	"github.com/voronkovd/gamayun/internal/update"
	"github.com/voronkovd/gamayun/internal/version"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("gamayun: ")

	once := flag.Bool("once", false, "run checks once and exit")
	test := flag.Bool("test", false, "send a Telegram test message")
	doDigest := flag.Bool("digest", false, "send daily digest now")
	showVersion := flag.Bool("version", false, "print version and exit")
	doUpdate := flag.Bool("update", false, "download the latest GitHub release and replace this binary")
	repo := flag.String("repo", "", "GitHub owner/name for --update")
	conf := flag.String("config", config.DefaultPath, "path to YAML config")
	flag.Parse()

	if *showVersion {
		fmt.Printf("gamayun %s", version.Version)
		if version.Repo != "" {
			fmt.Printf(" (%s)", version.Repo)
		}
		fmt.Println()
		return
	}

	modes := 0
	if *once {
		modes++
	}
	if *test {
		modes++
	}
	if *doDigest {
		modes++
	}
	if *doUpdate {
		modes++
	}
	if modes > 1 {
		log.Fatal("use only one of --once, --test, --digest, --update")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if *doUpdate {
		if err := runUpdate(ctx, *repo, *conf); err != nil {
			log.Fatal(err)
		}
		return
	}

	cfg, err := config.Load(*conf)
	if err != nil {
		log.Fatal(err)
	}

	var sender notify.Sender
	if cfg.TelegramOK() {
		sender = notify.LogSender{Inner: notify.NewTelegram(cfg.TGBotToken, cfg.TGChatID), Log: log.Printf}
	}

	switch {
	case *test:
		if sender == nil {
			log.Fatalf("telegram not configured: set telegram.bot_token and telegram.chat_id in %s", cfg.ConfigPath)
		}
		text := fmt.Sprintf("[TEST] from %s: Gamayun is wired up correctly (%s)", cfg.ServerName, time.Now().Format("2006-01-02 15:04:05 MST"))
		if err := sender.Send(ctx, text); err != nil {
			log.Fatal(err)
		}
	case *once:
		os.Exit(service.PrintOnce(checks.Default(cfg, checks.DefaultExec()).Run(ctx)))
	case *doDigest:
		d, err := service.New(cfg, checks.Default(cfg, checks.DefaultExec()), sender)
		if err != nil {
			log.Fatal(err)
		}
		if err := d.ForceDigest(ctx); err != nil {
			log.Fatal(err)
		}
	default:
		d, err := service.New(cfg, checks.Default(cfg, checks.DefaultExec()), sender)
		if err != nil {
			log.Fatal(err)
		}
		if err := d.Run(ctx); err != nil {
			log.Fatal(err)
		}
	}
}

func runUpdate(ctx context.Context, repoFlag, confPath string) error {
	repo := repoFlag
	if repo == "" {
		repo = version.Repo
	}
	if repo == "" {
		if cfg, err := config.Load(confPath); err == nil {
			repo = cfg.GitHubRepo
		}
	}
	c := update.DefaultClient(repo, "")
	return c.Run(ctx)
}
