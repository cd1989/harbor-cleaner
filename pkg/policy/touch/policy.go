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
	return nil, nil
}
