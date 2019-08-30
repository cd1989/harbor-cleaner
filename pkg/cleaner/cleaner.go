package cleaner

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/cd1989/harbor-cleaner/pkg/config"
	"github.com/cd1989/harbor-cleaner/pkg/harbor"
	"github.com/cd1989/harbor-cleaner/pkg/policy"
)

type Runner interface {
	DryRun() error
	Clean() error
}

type runner struct {
	client *harbor.Client
	cfg    config.C
}

func NewRunner(client *harbor.Client, cfg config.C) Runner {
	return &runner{
		client: client,
		cfg:    cfg,
	}
}

func (c *runner) DryRun() error {
	factory := policy.GetProcessorFactory((policy.Type)(c.cfg.Policy.Type))
	if factory == nil {
		return fmt.Errorf("no processor factory found for policy type: %s", c.cfg.Policy.Type)
	}

	candidates, err := factory(c.cfg).ListCandidates()
	if err != nil {
		return fmt.Errorf("list candidates error: %v", err)
	}

	imageCount := 0
	for _, repo := range candidates {
		for _, tag := range repo.Tags {
			imageCount++
			fmt.Printf("[%s] %s/%s:%s\n", tag.Created.Format("2006-01-02 15:04:05"), repo.Project, repo.Repo, tag.Name)
		}
		for _, tags := range repo.Protected {
			fmt.Printf("Repo: %s/%s, tags: %v to protect\n", repo.Project, repo.Repo, tags)
		}
	}
	fmt.Printf("Total %d repos with %d images are ready for clean\n", len(candidates), imageCount)

	return nil
}

func (c *runner) Clean() error {
	factory := policy.GetProcessorFactory((policy.Type)(c.cfg.Policy.Type))
	if factory == nil {
		return fmt.Errorf("no processor factory found for policy type: %s", c.cfg.Policy.Type)
	}

	candidates, err := factory(c.cfg).ListCandidates()
	if err != nil {
		return fmt.Errorf("list candidates error: %v", err)
	}

	// Clean the collected images
	logrus.Infof("Start to clean images for %d repo...", len(candidates))
	count := 0
	for _, repo := range candidates {
		repoCleaner := NewRepoCleaner(repo, c.client)

		// Protect tags not to be deleted as side effect of other tags' deletion
		logrus.Infof("Start to protect tags")
		if err := repoCleaner.Protect(); err != nil {
			logrus.Error("Failed to protect tags, skip this repo")
			continue
		}

		// Delete tags
		logrus.Infof("Start to clean %d images for repo '%s'...", len(repo.Tags), repo.Repo)
		n, err := repoCleaner.Clean()
		if err != nil {
			logrus.Warningf("Clean tags error: %v", err)
			continue
		}
		count += n

		// Push back tags that are removed as side effect of previous tag deletion
		if err := repoCleaner.Restore(); err != nil {
			logrus.Errorf("Restore tags error: %v", err)
			continue
		}
	}
	logrus.Infof("Totally %d images cleaned", count)

	return nil
}
