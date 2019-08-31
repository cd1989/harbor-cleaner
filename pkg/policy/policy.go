package policy

import (
	"fmt"

	"github.com/goharbor/harbor/src/common/utils"
	"github.com/sirupsen/logrus"

	"github.com/cd1989/harbor-cleaner/pkg/config"
	"github.com/cd1989/harbor-cleaner/pkg/harbor"
)

type Type string

const (
	NumberLimitPolicy        Type = "number"
	RecentlyNotTouchedPolicy Type = "recentlyNotTouched"
	RegexPolicy              Type = "regex"
)

// Processor defines process interface of a clean policy.
type Processor interface {
	// ListCandidates lists all image candidates to be removed.
	ListCandidates() ([]*Candidate, error)
	// ListTags lists all image tags
	ListTags() ([]*RepoTags, error)
	// Get policy type
	GetPolicyType() Type
}

// processorFactoryRegistry stores factories for all supported policy processors
var processorFactoryRegistry = make(map[Type]func(cfg config.C) Processor)

// RegisterProcessorFactory register a processor factory with given policy type.
func RegisterProcessorFactory(policyType Type, factory func(cfg config.C) Processor) {
	processorFactoryRegistry[policyType] = factory
}

// GetProcessorFactory gets processor factory with the given policy type
func GetProcessorFactory(policyType Type) func(cfg config.C) Processor {
	factory, ok := processorFactoryRegistry[policyType]
	if !ok {
		return nil
	}

	return factory
}

// BaseProcessor defines base logic for policy processor
type BaseProcessor struct {
	Cfg    config.C
	Client *harbor.Client
}

// Ensure (*numberPolicyProcessor) implements interface Processor
var _ Processor = (*BaseProcessor)(nil)

// GetPolicyType gets policy type.
func (p *BaseProcessor) GetPolicyType() Type {
	return ""
}

// ListCandidates list all candidates to be remove based on the policy
func (p *BaseProcessor) ListCandidates() ([]*Candidate, error) {
	return nil, fmt.Errorf("ListCandidates not implemented")
}

// ListTags lists all tags
func (p *BaseProcessor) ListTags() ([]*RepoTags, error) {
	projects, err := p.Client.AllProjects("", "")
	if err != nil {
		logrus.Errorf("List projects error: %v", err)
		return nil, err
	}

	if len(p.Cfg.Projects) != 0 {
		projectsMap := make(map[string]*harbor.Project)
		for _, p := range projects {
			projectsMap[p.Name] = p
		}

		var configuredProjects []*harbor.Project
		for _, p := range p.Cfg.Projects {
			if pinfo, ok := projectsMap[p]; ok {
				configuredProjects = append(configuredProjects, pinfo)
			} else {
				return nil, fmt.Errorf("project %s not found", p)
			}
		}
		projects = configuredProjects
	}

	var results []*RepoTags
	for _, pinfo := range projects {
		logrus.Infof("Start to collect images for project '%s'", pinfo.Name)
		repos, err := p.Client.ListAllRepositories(pinfo.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("list repos for project '%s' error: %v", pinfo.Name, err)
		}

		for _, repo := range repos {
			proj, r := utils.ParseRepository(repo.Name)
			tags, err := p.Client.ListTags(proj, r)
			if err != nil {
				logrus.Errorf("List tags for '%s/%s' error: %v", proj, r, err)
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

			results = append(results, &RepoTags{Project: pinfo.Name, Repo: r, Tags: tagsInfo})
		}
	}

	return results, nil
}
