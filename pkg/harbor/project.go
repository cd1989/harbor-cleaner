package harbor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
)

// Get a page of projects that match the given parameters:
// - name: Project with name that contains substring given by the parameter 'name'. If set to empty string, all project
// names match.
// - public: Whether project is a public project, if set to 'true', only public project will match, if set to 'false',
// only private projects match, and if set to empty string, both private and public projects match.
func (c *Client) ListProjects(page, pageSize int, name, public string) (int, []*Project, error) {
	path := ProjectsPath(page, pageSize, name, public)

	logrus.Infof("%s %s", http.MethodGet, path)
	resp, err := c.do(http.MethodGet, path, nil)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	if resp.StatusCode/100 == 2 {
		ret := make([]*Project, 0)
		err := json.Unmarshal(body, &ret)
		if err != nil {
			return 0, nil, err
		}
		total, err := getTotalFromResp(resp)
		if err != nil {
			logrus.Errorf("get total from resp error: %v", err)
			return 0, nil, err
		}
		return total, ret, nil
	}
	logrus.Errorf("list harbor projects error: %s", body)

	return 0, nil, fmt.Errorf("%s", body)
}

// Get all projects matching the given parameters.
// - name: Project with name that contains substring given by the parameter 'name'. If set to empty string, all project
// names match.
// - public: Whether project is a public project, if set to 'true', only public project will match, if set to 'false',
// only private projects match, and if set to empty string, both private and public projects match.
// Here are some examples:
// * Get all projects: AllProjects("", "")
// * Get all public projects: AllProjects("", "true")
// * Get all private projects whose names include "devops": AllProjects("devops", "false")
func (c *Client) AllProjects(name, public string) ([]*Project, error) {
	page := 1
	ret := make([]*Project, 0)
	for {
		total, projects, err := c.ListProjects(page, MaxPageSize, name, public)
		if err != nil {
			return nil, err
		}
		ret = append(ret, projects...)
		if total <= page*MaxPageSize {
			break
		}
		page++
	}
	return ret, nil
}
