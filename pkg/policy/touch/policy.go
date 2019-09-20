package touch

import (
	"fmt"
	"time"

	"github.com/cd1989/harbor-cleaner/pkg/config"
	"github.com/cd1989/harbor-cleaner/pkg/harbor"
	"github.com/cd1989/harbor-cleaner/pkg/policy"
)

func init() {
	policy.RegisterProcessorFactory(policy.RecentlyNotTouchedPolicy, newFactory())
}

func newFactory() func(cfg config.C) policy.Processor {
	return func(cfg config.C) policy.Processor {
		return &touchPolicyProcessor{
			BaseProcessor: policy.BaseProcessor{
				Client: harbor.APIClient,
				Cfg:    cfg,
			},
		}
	}
}

type touchPolicyProcessor struct {
	policy.BaseProcessor
}

// Ensure (*numberPolicyProcessor) implements interface Processor
var _ policy.Processor = (*touchPolicyProcessor)(nil)

// GetPolicyType gets policy type.
func (p *touchPolicyProcessor) GetPolicyType() policy.Type {
	return policy.RecentlyNotTouchedPolicy
}

// ListCandidates list all candidates to be remove based on the policy
func (p *touchPolicyProcessor) ListCandidates() ([]*policy.Candidate, error) {
	images, err := p.ListTags()
	if err != nil {
		return nil, err
	}

	if p.Cfg.Policy.NotTouchedPolicy == nil {
		return nil, fmt.Errorf("policy.notTouchedPolicy not configured, it's necessary when policy.type == 'recentlyNotTouched'")
	}

	endTime := time.Now().Unix()
	startTime := endTime - p.Cfg.Policy.NotTouchedPolicy.Time
	accessLogs, err := p.Client.ListAllAccessLogs(startTime, endTime)
	if err != nil {
		return nil, err
	}

	touchedMap := make(map[string]struct{})
	for _, log := range accessLogs {
		touchedMap[fmt.Sprintf("%s:%s", log.RepoName, log.Tag)] = struct{}{}
	}

	var imagesToClean []*policy.Candidate
	for _, r := range images {
		var candidates []policy.Tag
		var remains []policy.Tag
		for _, t := range r.Tags {
			if _, ok := touchedMap[fmt.Sprintf("%s/%s:%s", r.Project, r.Repo, t.Name)]; ok || policy.Retain(p.Cfg.Policy.RetainTags, t.Name) {
				remains = append(remains, t)
				continue
			}

			candidates = append(candidates, t)
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
