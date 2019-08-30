package touch

import (
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
			client: harbor.APIClient,
			cfg:    cfg,
		}
	}
}

type touchPolicyProcessor struct {
	cfg    config.C
	client *harbor.Client
}

// Ensure (*numberPolicyProcessor) implements interface Processor
var _ policy.Processor = (*touchPolicyProcessor)(nil)

// GetPolicyType gets policy type.
func (p *touchPolicyProcessor) GetPolicyType() policy.Type {
	return policy.NumberLimitPolicy
}

// ListCandidates list all candidates to be remove based on the policy
func (p *touchPolicyProcessor) ListCandidates() ([]*policy.Candidate, error) {
	return nil, nil
}

// ListTags lists all tags
func (p *touchPolicyProcessor) ListTags() ([]*policy.RepoTags, error) {
	return nil, nil
}
