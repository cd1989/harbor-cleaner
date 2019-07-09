package clean

import (
	"fmt"
	"path"
	"sync"

	"github.com/goharbor/harbor/src/common/utils"
	"github.com/sirupsen/logrus"

	"github.com/cd1989/harbor-cleaner/pkg/config"
	"github.com/cd1989/harbor-cleaner/pkg/harbor"
)

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
		client: client,
		policy: policy,
		lock:   &sync.Mutex{},
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
		logrus.Infof("Start to clean %d images for repo '%s'...", len(repo.Tags), repo.Repo)
		for _, tag := range repo.Tags {
			if err := c.client.DeleteTag(repo.Project, repo.Repo, tag.Name); err != nil {
				logrus.Warningf("Clean image '%s' error: %v", fmt.Sprintf("%s/%s:%s", repo.Project, repo.Repo, tag), err)
			} else {
				count++
			}
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
	Name   string `json:"name"`
	Digest string `json:"digest"`
}
type RepoImages struct {
	Project string `json:"project"`
	Repo    string `json:"repo"`
	Tags    []Tag  `json:"tags"`
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

		remainsDigests := make(map[string]struct{})
		for _, t := range remains {
			remainsDigests[t.Digest] = struct{}{}
		}

		var safeCandidates []Tag
		for _, t := range candidates {
			if _, ok := remainsDigests[t.Digest]; !ok {
				safeCandidates = append(safeCandidates, t)
			}
		}

		if len(safeCandidates) > 0 {
			imagesToClean = append(imagesToClean, &RepoImages{
				Project: r.Project,
				Repo:    r.Repo,
				Tags:    safeCandidates,
			})
		}
	}

	return imagesToClean, nil
}

func (c *policyClean) imagesToCleanByTouchTime() ([]*RepoImages, error) {
	return nil, fmt.Errorf("clean by touch time not implemented yet")
}

func (c *policyClean) listImages() ([]*RepoImages, error) {
	public := ""
	if c.policy.IncludePublic {
		public = "true"
	}
	_, projects, err := c.client.ListProjects(1, 9999, "", public)
	if err != nil {
		logrus.Errorf("List projects error: %v", err)
		return nil, err
	}

	if len(c.projects) != 0 {
		projectsMap := make(map[string]*harbor.HarborProject)
		for _, p := range projects {
			projectsMap[p.Name] = p
		}

		var configuredProjects []*harbor.HarborProject
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
					Name:   tag.Name,
					Digest: tag.Digest,
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
