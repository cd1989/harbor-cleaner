package number

import (
	"fmt"

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
			BaseProcessor: policy.BaseProcessor{
				Client: harbor.APIClient,
				Cfg:    cfg,
			},
		}
	}
}

type numberPolicyProcessor struct {
	policy.BaseProcessor
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

	if p.Cfg.Policy.NumPolicy == nil {
		return nil, fmt.Errorf("policy.NumPolicy not configured")
	}

	var imagesToClean []*policy.Candidate
	for _, r := range images {
		if len(r.Tags) <= p.Cfg.Policy.NumPolicy.Num {
			continue
		}

		var candidates []policy.Tag
		var remains = r.Tags[0:p.Cfg.Policy.NumPolicy.Num]
		for _, t := range r.Tags[p.Cfg.Policy.NumPolicy.Num:] {
			if !policy.Retain(p.Cfg.Policy.RetainTags, t.Name) {
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
