package harbor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
)

const (
	SortByNameAsc = "+name"
	SortByNameDes = "-name"
	SortByDateAsc = "+date"
	SortByDateDes = "-date"
)

// Get a page of repos that match the 'query', the match here means the repo name contains the 'query' value.
// Result returned is sorted by the given 'sorting' method, 'name' and 'date' sorting methods supported. Since
// Harbor 1.4.0 doesn't support repos list sorting, we need to perform sorting here after retrieve all repos from
// Harbor. This may be a little inefficient when project is large.
func (c *Client) ListRepos(projectId int64, query string, page, pageSize int) (int, []*HarborRepo, error) {
	if page <= 0 || pageSize <= 0 {
		return 0, nil, fmt.Errorf("Invalid page or pageSize")
	}

	total, repos, err := c.allRepos(projectId, query)
	if err != nil {
		return 0, nil, fmt.Errorf("get repos from harbor error")
	}

	// If the pagination is out of bound, return empty result
	if total <= pageSize*(page-1) {
		return total, make([]*HarborRepo, 0), nil
	}

	to := pageSize * page
	if to > total {
		to = total
	}
	return total, repos[pageSize*(page-1) : to], nil
}

func (c *Client) getRepos(projectId int64, query string, page, pageSize int) (int, []*HarborRepo, error) {
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

	repos := make([]*HarborRepo, 0)
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

func (c *Client) allRepos(projectId int64, query string) (int, []*HarborRepo, error) {
	page, total := 1, 0
	result := make([]*HarborRepo, 0)
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

func (c *Client) ListAllRepositories(projectId int64) ([]*HarborRepo, error) {
	_, repos, err := c.allRepos(projectId, "")
	return repos, err
}

func (c *Client) GetRepository(projectId int64, projectName, repoName string) (*HarborRepo, error) {
	name := fmt.Sprintf("%s/%s", projectName, repoName)
	path := ReposPath(projectId, name, 1, 1)

	logrus.Infof("%s %s", http.MethodGet, path)
	resp, err := c.do(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode/100 == 2 {
		ret := make([]*HarborRepo, 0)
		err := json.Unmarshal(body, &ret)
		if err != nil {
			logrus.Errorf("unmarshal repositories error: %v", err)
			logrus.Infof("resp body: %s", body)
			return nil, err
		}
		if len(ret) == 0 {
			logrus.Errorf("not found the repository: %s from project: %d", name, projectId)
			return nil, fmt.Errorf("repository: %s not found", repoName)
		}
		if ret[0].Name != fmt.Sprintf("%s/%s", projectName, repoName) {
			logrus.Errorf("not found the repository: %s from project: %d", name, projectId)
			logrus.Errorf("the first repository is %s, but the expected result: %s", ret[0].Name, name)
			return nil, fmt.Errorf("repository: %s not found", repoName)
		}
		return ret[0], nil
	}
	logrus.Errorf("get harbor repositories: %s error: %s, StatusCode: %d", name, body, resp.StatusCode)

	return nil, fmt.Errorf(string(body))
}

func (c *Client) DeleteRepository(projectName, repoName string) error {
	path := RepoPath(projectName, repoName)

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
		logrus.Infof("delete repository: %s/%s sucess", projectName, repoName)
		return nil
	}
	if resp.StatusCode == 404 {
		return fmt.Errorf("repository: %s/%s not found", projectName, repoName)
	}
	logrus.Errorf("delete harbor repository: %s/%s error: %s, StatusCode: %d", projectName, repoName, body, resp.StatusCode)

	return fmt.Errorf(string(body))
}
