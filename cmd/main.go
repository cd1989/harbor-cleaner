package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/cd1989/harbor-cleaner/pkg/cleaner"
	"github.com/cd1989/harbor-cleaner/pkg/config"
	"github.com/cd1989/harbor-cleaner/pkg/harbor"
	_ "github.com/cd1989/harbor-cleaner/pkg/policy/number"
	_ "github.com/cd1989/harbor-cleaner/pkg/policy/regex"
	_ "github.com/cd1989/harbor-cleaner/pkg/policy/touch"
	"github.com/cd1989/harbor-cleaner/pkg/trigger"
)

var configFile *string
var dryRun *bool

func main() {
	configFile = flag.String("config", "/workspace/config.yaml", "Config file")
	dryRun = flag.Bool("dryrun", false, "Whether only dry run the clean")
	if !flag.Parsed() {
		flag.Parse()
	}

	err := config.Load(*configFile)
	if err != nil {
		logrus.Fatalf("Load config failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	gracefulShutdown(cancel)
	client, err := harbor.NewClient(&config.Config, ctx.Done())
	if err != nil {
		logrus.Fatalf("Init Harbor client error: %v", err)
	}
	harbor.APIClient = client

	if config.HasCronSchedule() {
		scheduler := trigger.NewCronScheduler(config.Config.Trigger.Cron)
		scheduler.Submit(func() {
			RunTask(client)
		})
		scheduler.Start()
		<-ctx.Done()
	} else {
		RunTask(client)
	}
}

// RunTask starts cleanup task
func RunTask(client *harbor.Client) {
	runner := cleaner.NewRunner(client, config.Config)
	if *dryRun {
		if err := runner.DryRun(); err != nil {
			logrus.Errorf("Dryrun error: %v", err)
		}
	} else {
		if err := runner.Clean(); err != nil {
			logrus.Errorf("Clean error: %v", err)
		}
	}
}

// gracefulShutdown catches signals of Interrupt, SIGINT, SIGTERM, SIGQUIT and cancel a context.
// If any signals caught, it will call the CancelFunc to cancel a context. If a second signal
// caught, exit directly with code 1.
func gracefulShutdown(cancel context.CancelFunc) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		s := <-c
		logrus.WithField("signal", s).Debug("System signal caught, cancel context.")
		cancel()
		s = <-c
		logrus.WithField("signal", s).Debug("Another system signal caught, exit directly.")
		os.Exit(1)
	}()
}
