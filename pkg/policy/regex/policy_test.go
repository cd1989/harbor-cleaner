package regex

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cd1989/harbor-cleaner/pkg/config"
	"github.com/cd1989/harbor-cleaner/pkg/policy"
)

func TestMatch(t *testing.T) {
	processor := &regexPolicyProcessor{
		BaseProcessor: policy.BaseProcessor{
			Client: nil,
			Cfg: config.C{
				Policy: config.Policy{
					RegexPolicy: &config.RegexPolicy{
						Repos: []string{".*"},
						Tags:  []string{"v1.*"},
					},
				},
			},
		},
	}
	processor.compileRegex()

	assert.True(t, processor.matchRepo("busybox"))
	assert.True(t, processor.matchTag("v1.0.0"))
	assert.False(t, processor.matchTag("vv1.0.0"))
	assert.False(t, processor.matchTag("v2.0.0"))
}
