package harbor

import (
	"time"
)

// HarborProject holds details of a project.
type Project struct {
	ProjectID    int64             `json:"project_id"`
	OwnerID      int               `json:"owner_id"`
	Name         string            `json:"name"`
	CreationTime time.Time         `json:"creation_time"`
	UpdateTime   time.Time         `json:"update_time"`
	OwnerName    string            `json:"owner_name"`
	Togglable    bool              `json:"togglable"`
	Role         int               `json:"current_user_role_id"`
	RepoCount    int               `json:"repo_count"`
	Metadata     map[string]string `json:"metadata"`
}

type Repo struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	ProjectID    int64     `json:"project_id"`
	Description  string    `json:"description"`
	PullCount    int64     `json:"pull_count"`
	StarCount    int64     `json:"star_count"`
	TagsCount    int64     `json:"tags_count"`
	CreationTime time.Time `json:"creation_time"`
	UpdateTime   time.Time `json:"update_time"`
}

type Tag struct {
	tagDetail
}

type tagDetail struct {
	Digest        string    `json:"digest"`
	Name          string    `json:"name"`
	Size          int64     `json:"size"`
	Architecture  string    `json:"architecture"`
	OS            string    `json:"os"`
	DockerVersion string    `json:"docker_version"`
	Author        string    `json:"author"`
	Created       time.Time `json:"created"`
	Config        *cfg      `json:"config"`
}

type cfg struct {
	Labels map[string]string `json:"labels"`
}

type TagManifest struct {
	Config   string            `json:"config"`
	Manifest TagManifestDetail `json:"manifest"`
}

type TagManifestDetail struct {
	MediaType     string       `json:"mediaType"`
	SchemaVersion int          `json:"schemaVersion"`
	Layers        []*TagLayers `json:"layers"`
}

type TagLayers struct {
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
	Size      int    `json:"size"`
}
