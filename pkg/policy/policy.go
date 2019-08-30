package policy

import "github.com/cd1989/harbor-cleaner/pkg/config"

type Type string

const (
	NumberLimitPolicy        Type = "number"
	RecentlyNotTouchedPolicy Type = "recentlyNotTouched"
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
