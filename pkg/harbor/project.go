package harbor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/sirupsen/logrus"
)

// Get a page of projects that match the given parameters:
// - name: Project with name that contains substring given by the parameter 'name'. If set to empty string, all project
// names match.
// - public: Whether project is a public project, if set to 'true', only public project will match, if set to 'false',
// only private projects match, and if set to empty string, both private and public projects match.
func (c *Client) ListProjects(page, pageSize int, name, public string) (int, []*HarborProject, error) {
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
		ret := make([]*HarborProject, 0)
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
func (c *Client) AllProjects(name, public string) ([]*HarborProject, error) {
	page := 1
	ret := make([]*HarborProject, 0)
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

func (c *Client) GetProject(pid int64) (*HarborProject, error) {
	path := ProjectPath(pid)

	logrus.Infof("%s %s", http.MethodGet, path)
	resp, err := c.do(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode/100 == 2 {
		ret := &HarborProject{}
		err := json.Unmarshal(body, &ret)
		if err != nil {
			return nil, err
		}

		if count, err := c.GetRepoCount(pid); err == nil {
			ret.RepoCount = count
		} else {
			logrus.Warningf("get repo count for project %s error: %v", pid, err)
		}

		return ret, nil
	}
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("project: %d", pid)
	}
	logrus.Errorf("get harbor project '%d' error: %s", pid, body)

	return nil, fmt.Errorf("%s", body)
}

func (c *Client) GetProjectByName(name string) (*HarborProject, error) {
	projects, err := c.AllProjects(name, "")
	if err != nil {
		return nil, err
	}
	for _, p := range projects {
		if p.Name == name {
			return p, nil
		}
	}
	return nil, nil
}

func (c *Client) DeleteProject(pid int64) error {
	path := ProjectPath(pid)

	logrus.Infof("%s %s", http.MethodDelete, path)
	resp, err := c.do(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode/100 == 2 {
		logrus.Infof("delete project: %d sucess", pid)
		return nil
	}
	if resp.StatusCode == 404 {
		return fmt.Errorf("project: %d", pid)
	}
	logrus.Errorf("delete harbor project: %d error: %s", pid, body)

	return fmt.Errorf("%s", body)
}

func (c *Client) GetProjectDeleteable(pid int64) (*HarborProjectDeletableResp, error) {
	path := ProjectDeletablePath(pid)

	logrus.Infof("%s %s", http.MethodGet, path)
	resp, err := c.do(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode/100 == 2 {
		ret := &HarborProjectDeletableResp{}
		err := json.Unmarshal(body, &ret)
		if err != nil {
			return nil, err
		}
		return ret, nil
	}

	logrus.Errorf("check project: %d whether is deletable error: %s", pid, body)
	return nil, fmt.Errorf("%s", body)
}

func (c *Client) GetRepoCount(pid int64) (int, error) {
	path := ReposPath(pid, "", 1, 1)
	resp, err := c.do(http.MethodGet, path, nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return strconv.Atoi(resp.Header.Get(RespHeaderTotal))
}
