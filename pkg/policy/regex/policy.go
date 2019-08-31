package regex

import (
	"fmt"
	"regexp"

	"strings"

	"github.com/cd1989/harbor-cleaner/pkg/config"
	"github.com/cd1989/harbor-cleaner/pkg/harbor"
	"github.com/cd1989/harbor-cleaner/pkg/policy"
)

func init() {
	policy.RegisterProcessorFactory(policy.RegexPolicy, newFactory())
}

func newFactory() func(cfg config.C) policy.Processor {
	return func(cfg config.C) policy.Processor {
		return &regexPolicyProcessor{
			BaseProcessor: policy.BaseProcessor{
				Client: harbor.APIClient,
				Cfg:    cfg,
			},
		}
	}
}

type regexPolicyProcessor struct {
	policy.BaseProcessor
	repoPatterns []*regexp.Regexp
	tagPatterns  []*regexp.Regexp
}

// Ensure (*numberPolicyProcessor) implements interface Processor
var _ policy.Processor = (*regexPolicyProcessor)(nil)

// GetPolicyType gets policy type.
func (p *regexPolicyProcessor) GetPolicyType() policy.Type {
	return policy.RegexPolicy
}

// ListCandidates list all candidates to be remove based on the policy
func (p *regexPolicyProcessor) ListCandidates() ([]*policy.Candidate, error) {
	images, err := p.ListTags()
	if err != nil {
		return nil, err
	}

	if p.Cfg.Policy.RegexPolicy == nil {
		return nil, fmt.Errorf("policy.regexPolicy not configured")
	}

	if len(p.Cfg.Policy.RegexPolicy.Repos) == 0 {
		return nil, fmt.Errorf("policy.regexPolicy.repos is empty, nothing will be cleaned, you may want '.*'")
	}

	if len(p.Cfg.Policy.RegexPolicy.Tags) == 0 {
		return nil, fmt.Errorf("policy.regexPolicy.tags is empty, nothing will be cleaned, you may want '.*'")
	}

	if err := p.compileRegex(); err != nil {
		return nil, err
	}

	var imagesToClean []*policy.Candidate
	for _, r := range images {
		if !p.matchRepo(r.Repo) {
			continue
		}

		var candidates []policy.Tag
		var remains []policy.Tag
		for _, t := range r.Tags {
			if p.matchTag(t.Name) && !policy.Retain(p.Cfg.Policy.RetainTags, t.Name) {
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

func normalizeRegex(regex string) string {
	if !strings.HasPrefix(regex, "^") {
		regex = "^" + regex
	}

	if !strings.HasSuffix(regex, "$") {
		regex = regex + "$"
	}

	return regex
}

func (p *regexPolicyProcessor) compileRegex() error {
	for _, repoRegex := range p.Cfg.Policy.RegexPolicy.Repos {
		r, e := regexp.Compile(normalizeRegex(repoRegex))
		if e != nil {
			return fmt.Errorf("compile regex %s error: %v", repoRegex, e)
		}
		p.repoPatterns = append(p.repoPatterns, r)
	}

	for _, tagRegex := range p.Cfg.Policy.RegexPolicy.Tags {
		r, e := regexp.Compile(normalizeRegex(tagRegex))
		if e != nil {
			return fmt.Errorf("compile regex %s error: %v", tagRegex, e)
		}
		p.tagPatterns = append(p.tagPatterns, r)
	}

	return nil
}

func (p *regexPolicyProcessor) matchRepo(repo string) bool {
	for _, repoPattern := range p.repoPatterns {
		if repoPattern.MatchString(repo) {
			return true
		}
	}

	return false
}

func (p *regexPolicyProcessor) matchTag(tag string) bool {
	for _, tagPattern := range p.tagPatterns {
		if tagPattern.MatchString(tag) {
			return true
		}
	}

	return false
}
