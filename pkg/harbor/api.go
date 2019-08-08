package harbor

import (
	"fmt"
	"net/url"
)

const (
	APIProjects      = "/api/projects"
	APIProject       = "/api/projects/%d"
	APIRepositories  = "/api/repositories"
	APITags          = "/api/repositories/%s/%s/tags"
	APITag           = "/api/repositories/%s/%s/tags/%s"
	APIImageManifest = "/api/repositories/%s/%s/tags/%s/manifest"
	APITarget        = "/api/targets/%d"
	APITagsV04       = "/api/repositories/tags?repo_name=%s"
	APIDeleteTagV04  = "/api/repositories?repo_name=%s&tag=%s"
)

func ProjectsPath(page, pageSize int, name, public string) string {
	return fmt.Sprintf("%s?page=%d&page_size=%d&name=%s&public=%s", APIProjects, page, pageSize, url.QueryEscape(name), public)
}

func ProjectPath(pid int64) string {
	return fmt.Sprintf(APIProject, pid)
}

func TagsPath(project, repo, version string) string {
	if version <= "0.4" {
		return fmt.Sprintf(APITagsV04, fmt.Sprintf("%s/%s", project, repo))
	}
	return fmt.Sprintf(APITags, project, repo)
}

func TagPath(project, repo, tag, version string) string {
	if version <= "0.4" {
		return fmt.Sprintf(APIDeleteTagV04, fmt.Sprintf("%s/%s", project, repo), tag)
	}
	return fmt.Sprintf(APITag, project, repo, tag)
}

func ImageManifestPath(project, repo, tag, version string) string {
	if version <= "0.4" {
		return fmt.Sprintf("/api/repositories/manifests?repo_name=%s/%s&tag=%s", project, repo, tag)
	}
	return fmt.Sprintf(APIImageManifest, project, repo, tag)
}

func LoginUrl(host, version, user, pwd string) string {
	if version >= "1.7" {
		return fmt.Sprintf("%s/c/login?principal=%s&password=%s", host, user, pwd)
	}
	return fmt.Sprintf("%s/login?principal=%s&password=%s", host, user, pwd)
}

func ReposPath(pid int64, query string, page, pageSize int) string {
	return fmt.Sprintf("%s?project_id=%d&q=%s&page=%d&page_size=%d", APIRepositories, pid, url.QueryEscape(query), page, pageSize)
}

func TargetPath(tid int64) string {
	return fmt.Sprintf(APITarget, tid)
}
