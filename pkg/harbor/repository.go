package harbor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
)

func (c *Client) getRepos(projectId int64, query string, page, pageSize int) (int, []*Repo, error) {
	// Query with a large enough page size will retrieve all repos
	path := ReposPath(projectId, query, page, pageSize)
	logrus.Infof("%s %s", http.MethodGet, path)
	resp, err := c.do(http.MethodGet, path, nil)
	if err != nil {
		logrus.Info(err)
		return 0, nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	if resp.StatusCode/100 != 2 {
		logrus.Errorf("list harbor repositories from projectId: %s error: %s, StatusCode: %d", projectId, body, resp.StatusCode)
		return 0, nil, fmt.Errorf(string(body))
	}

	repos := make([]*Repo, 0)
	err = json.Unmarshal(body, &repos)
	if err != nil {
		logrus.Errorf("unmarshal repositories error: %v", err)
		logrus.Infof("resp body: %s", body)
		return 0, nil, err
	}
	total, err := getTotalFromResp(resp)
	if err != nil {
		logrus.Errorf("get total from resp error: %v", err)
		return 0, nil, err
	}

	return total, repos, nil
}

func (c *Client) allRepos(projectId int64, query string) (int, []*Repo, error) {
	page, total := 1, 0
	result := make([]*Repo, 0)
	for {
		t, repos, err := c.getRepos(projectId, query, page, MaxPageSize)
		if err != nil {
			return 0, nil, err
		}
		result = append(result, repos...)
		if t <= page*MaxPageSize {
			total = t
			break
		}
		page++
	}
	return total, result, nil
}

func (c *Client) ListAllRepositories(projectId int64) ([]*Repo, error) {
	_, repos, err := c.allRepos(projectId, "")
	return repos, err
}
