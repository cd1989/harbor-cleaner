package clean

import (
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/goharbor/harbor/src/common/utils"
	"github.com/sirupsen/logrus"

	"github.com/cd1989/harbor-cleaner/pkg/config"
	"github.com/cd1989/harbor-cleaner/pkg/harbor"
)

var acceptMediaTypes = []string{schema1.MediaTypeManifest, schema2.MediaTypeManifest}

type ImageCleaner interface {
	// Clean images
	Clean() error
	// DryRun collects images to be cleaned, but not clean them
	DryRun() ([]*RepoImages, error)
}

type policyClean struct {
	client   *harbor.Client
	policy   *config.Policy
	projects []string
	lock     *sync.Mutex
}

func NewPolicyCleaner(client *harbor.Client, projects []string, policy *config.Policy) ImageCleaner {
	return &policyClean{
		client:   client,
		policy:   policy,
		projects: projects,
		lock:     &sync.Mutex{},
	}
}

func (c *policyClean) Clean() error {
	// Collect images to clean by DryRun
	repoImages, err := c.DryRun()
	if err != nil {
		return fmt.Errorf("collect images to clean for '%v' error: %v", c.projects, err)
	}

	// Clean the collected images
	logrus.Infof("Start to clean images for %d repo...", len(repoImages))
	count := 0
	for _, repo := range repoImages {
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

func (c *policyClean) DryRun() ([]*RepoImages, error) {
	if c.policy == nil || c.client == nil {
		return nil, fmt.Errorf("clean policy or harbor client is empty")
	}

	if c.policy.NumPolicy != nil {
		return c.imagesToCleanByNum()
	}

	return nil, fmt.Errorf("clean policy not given")
}

type Tag struct {
	Name    string
	Digest  string
	Created time.Time
}
type RepoImages struct {
	Project   string
	Repo      string
	Tags      []Tag
	Protected map[string][]string
}

func (c *policyClean) imagesToCleanByNum() ([]*RepoImages, error) {
	images, err := c.listImages()
	if err != nil {
		return nil, err
	}

	var imagesToClean []*RepoImages
	for _, r := range images {
		if len(r.Tags) <= c.policy.NumPolicy.Num {
			continue
		}

		var candidates []Tag
		var remains = r.Tags[0:c.policy.NumPolicy.Num]
		for _, t := range r.Tags[c.policy.NumPolicy.Num:] {
			if !retain(c.policy.RetainTags, t.Name) {
				candidates = append(candidates, t)
				continue
			}

			remains = append(remains, t)
		}

		remainsDigests := make(map[string][]string)
		for _, t := range remains {
			remainsDigests[t.Digest] = append(remainsDigests[t.Digest], t.Name)
		}

		dangerTags := make(map[string][]string)
		for _, t := range candidates {
			if tags, ok := remainsDigests[t.Digest]; ok {
				dangerTags[t.Digest] = tags
			}
		}

		if len(candidates) > 0 {
			imagesToClean = append(imagesToClean, &RepoImages{
				Project:   r.Project,
				Repo:      r.Repo,
				Tags:      candidates,
				Protected: dangerTags,
			})
		}
	}

	return imagesToClean, nil
}

func (c *policyClean) imagesToCleanByTouchTime() ([]*RepoImages, error) {
	return nil, fmt.Errorf("clean by touch time not implemented yet")
}

func (c *policyClean) listImages() ([]*RepoImages, error) {
	projects, err := c.client.AllProjects("", "")
	if err != nil {
		logrus.Errorf("List projects error: %v", err)
		return nil, err
	}

	if len(c.projects) != 0 {
		projectsMap := make(map[string]*harbor.Project)
		for _, p := range projects {
			projectsMap[p.Name] = p
		}

		var configuredProjects []*harbor.Project
		for _, p := range c.projects {
			if pinfo, ok := projectsMap[p]; ok {
				configuredProjects = append(configuredProjects, pinfo)
			} else {
				return nil, fmt.Errorf("project %s not found", p)
			}
		}
		projects = configuredProjects
	}

	var results []*RepoImages
	for _, pinfo := range projects {
		logrus.Infof("Start to collect images for project '%s'", pinfo.Name)
		repos, err := c.client.ListAllRepositories(pinfo.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("list repos for project '%s' error: %v", pinfo.Name, err)
		}

		for _, repo := range repos {
			p, r := utils.ParseRepository(repo.Name)
			tags, err := c.client.ListTags(p, r)
			if err != nil {
				logrus.Errorf("List tags for '%s/%s' error: %v", p, r, err)
				continue
			}

			var tagsInfo []Tag
			for _, tag := range tags {
				tagsInfo = append(tagsInfo, Tag{
					Name:    tag.Name,
					Digest:  tag.Digest,
					Created: tag.Created,
				})
			}

			results = append(results, &RepoImages{Project: pinfo.Name, Repo: r, Tags: tagsInfo})
		}
	}

	return results, nil
}

// Check whether to retain the tag against the patterns.
func retain(patterns []string, tag string) bool {
	for _, pattern := range patterns {
		m, e := path.Match(pattern, tag)
		if e == nil && m {
			return true
		}
	}

	return false
}
