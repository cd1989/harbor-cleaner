package number

import (
	"fmt"
	"path"

	"github.com/goharbor/harbor/src/common/utils"
	"github.com/sirupsen/logrus"

	"github.com/cd1989/harbor-cleaner/pkg/config"
	"github.com/cd1989/harbor-cleaner/pkg/harbor"
	"github.com/cd1989/harbor-cleaner/pkg/policy"
)

func init() {
	policy.RegisterProcessorFactory(policy.NumberLimitPolicy, newFactory())
}

func newFactory() func(cfg config.C) policy.Processor {
	return func(cfg config.C) policy.Processor {
		return &numberPolicyProcessor{
			client: harbor.APIClient,
			cfg:    cfg,
		}
	}
}

type numberPolicyProcessor struct {
	cfg    config.C
	client *harbor.Client
}

// Ensure (*numberPolicyProcessor) implements interface Processor
var _ policy.Processor = (*numberPolicyProcessor)(nil)

// GetPolicyType gets policy type.
func (p *numberPolicyProcessor) GetPolicyType() policy.Type {
	return policy.NumberLimitPolicy
}

// ListCandidates list all candidates to be remove based on the policy
func (p *numberPolicyProcessor) ListCandidates() ([]*policy.Candidate, error) {
	images, err := p.ListTags()
	if err != nil {
		return nil, err
	}

	var imagesToClean []*policy.Candidate
	for _, r := range images {
		if len(r.Tags) <= p.cfg.Policy.NumPolicy.Num {
			continue
		}

		var candidates []policy.Tag
		var remains = r.Tags[0:p.cfg.Policy.NumPolicy.Num]
		for _, t := range r.Tags[p.cfg.Policy.NumPolicy.Num:] {
			if !retain(p.cfg.Policy.RetainTags, t.Name) {
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
			imagesToClean = append(imagesToClean, &policy.Candidate{
				Project:   r.Project,
				Repo:      r.Repo,
				Tags:      candidates,
				Protected: dangerTags,
			})
		}
	}

	return imagesToClean, nil
}

// ListTags lists all tags
func (p *numberPolicyProcessor) ListTags() ([]*policy.RepoTags, error) {
	projects, err := p.client.AllProjects("", "")
	if err != nil {
		logrus.Errorf("List projects error: %v", err)
		return nil, err
	}

	if len(p.cfg.Projects) != 0 {
		projectsMap := make(map[string]*harbor.Project)
		for _, p := range projects {
			projectsMap[p.Name] = p
		}

		var configuredProjects []*harbor.Project
		for _, p := range p.cfg.Projects {
			if pinfo, ok := projectsMap[p]; ok {
				configuredProjects = append(configuredProjects, pinfo)
			} else {
				return nil, fmt.Errorf("project %s not found", p)
			}
		}
		projects = configuredProjects
	}

	var results []*policy.RepoTags
	for _, pinfo := range projects {
		logrus.Infof("Start to collect images for project '%s'", pinfo.Name)
		repos, err := p.client.ListAllRepositories(pinfo.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("list repos for project '%s' error: %v", pinfo.Name, err)
		}

		for _, repo := range repos {
			proj, r := utils.ParseRepository(repo.Name)
			tags, err := p.client.ListTags(proj, r)
			if err != nil {
				logrus.Errorf("List tags for '%s/%s' error: %v", proj, r, err)
				continue
			}

			var tagsInfo []policy.Tag
			for _, tag := range tags {
				tagsInfo = append(tagsInfo, policy.Tag{
					Name:    tag.Name,
					Digest:  tag.Digest,
					Created: tag.Created,
				})
			}

			results = append(results, &policy.RepoTags{Project: pinfo.Name, Repo: r, Tags: tagsInfo})
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
