package clean

import (
	"fmt"
	"strings"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/sirupsen/logrus"

	"github.com/cd1989/harbor-cleaner/pkg/harbor"
)

type RepoCleaner struct {
	repo       *RepoImages
	client     *harbor.Client
	repoClient *harbor.RepoClient
	protected  []protectedTagsMenifest
}

type protectedTagsMenifest struct {
	tags      []string
	mediaType string
	payload   []byte
}

func NewRepoCleaner(repoImages *RepoImages, client *harbor.Client) *RepoCleaner {
	return &RepoCleaner{
		repo:   repoImages,
		client: client,
	}
}

func (c *RepoCleaner) Protect() error {
	if len(c.repo.Protected) > 0 {
		repoClient, err := harbor.NewRepoClient(c.client, fmt.Sprintf("%s/%s", c.repo.Project, c.repo.Repo))
		if err != nil {
			logrus.Errorf("Create repo client for repo %s/%s error: %v, skip this repo", c.repo.Project, c.repo.Repo, err)
			return err
		}
		c.repoClient = repoClient

		for digestID, tags := range c.repo.Protected {
			_, mediaType, payload, err := repoClient.PullManifest(digestID, acceptMediaTypes)
			if err != nil {
				logrus.Errorf("Pulling manifest %s/%s:%s error: %v", c.repo.Project, c.repo.Repo, digestID, err)
				return err
			}
			if strings.Contains(mediaType, "application/json") {
				mediaType = schema1.MediaTypeManifest
			}

			c.protected = append(c.protected, protectedTagsMenifest{
				tags:      tags,
				mediaType: mediaType,
				payload:   payload,
			})
		}
	}

	return nil
}

func (c *RepoCleaner) Clean() (int, error) {
	count := 0
	for _, tag := range c.repo.Tags {
		if err := c.client.DeleteTag(c.repo.Project, c.repo.Repo, tag.Name); err != nil {
			logrus.Warningf("Clean image '%s' error: %v", fmt.Sprintf("%s/%s:%s", c.repo.Project, c.repo.Repo, tag), err)
		} else {
			count++
		}
	}

	return count, nil
}

func (c *RepoCleaner) Restore() error {
	for _, r := range c.protected {
		logrus.Infof("Start to push back tags %v to %s/%s", r.tags, c.repo.Project, c.repo.Repo)
		for _, t := range r.tags {
			_, err := c.repoClient.PushManifest(t, r.mediaType, r.payload)
			if err != nil {
				logrus.Errorf("Push manifest %s/%s:%s error: %v", c.repo.Project, c.repo.Repo, t)
				return err
			}
		}
	}

	return nil
}
