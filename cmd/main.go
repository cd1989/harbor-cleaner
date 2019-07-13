package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/cd1989/harbor-cleaner/pkg/clean"
	"github.com/cd1989/harbor-cleaner/pkg/config"
	"github.com/cd1989/harbor-cleaner/pkg/harbor"
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

	cleaner := clean.NewPolicyCleaner(client, config.Config.Projects, &config.Config.Policy)
	if *dryRun {
		images, err := cleaner.DryRun()
		if err != nil {
			logrus.Errorf("Dryrun error: %v", err)
			return
		}
		imageCount := 0
		for _, repo := range images {
			for _, tag := range repo.Tags {
				imageCount++
				fmt.Printf("[%s] %s/%s:%s\n", tag.Created.Format("2006-01-02 15:04:05"), repo.Project, repo.Repo, tag.Name)
			}
			for digest, tags := range repo.Protected {
				fmt.Printf("Digest: %s, tags: %v\n", digest, tags)
			}
		}
		fmt.Printf("Total %d repos with %d images are ready for clean\n", len(images), imageCount)
	} else {
		err := cleaner.Clean()
		if err != nil {
			logrus.Errorf("Clean images error: %v", err)
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
