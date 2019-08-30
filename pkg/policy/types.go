package policy

import "time"

// Candidate defines images candidate to remove in a repo. 'Tags' are image tags to be
// removed, while 'Protected' holds all tags that will be affected when remove those tags.
// It is a map from digest ID to tags list. A tag needs to be protected when it has the
// same digest ID to those tags in 'Tags'.
type Candidate struct {
	Project   string
	Repo      string
	Tags      []Tag
	Protected map[string][]string
}

// RepoTags defines all image tags in a repo
type RepoTags struct {
	Project string
	Repo    string
	Tags    []Tag
}

// Tag describes an image tag
type Tag struct {
	Name    string
	Digest  string
	Created time.Time
}
