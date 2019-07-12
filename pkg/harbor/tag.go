package harbor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"

	"github.com/sirupsen/logrus"
)

func (c *Client) ListTags(projectName string, repoName string) ([]*Tag, error) {
	path := TagsPath(projectName, repoName)

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
		tags := make([]*Tag, 0)
		err := json.Unmarshal(body, &tags)
		if err != nil {
			logrus.Errorf("unmarshal tags error: %v", err)
			logrus.Infof("resp body: %s", body)
			return nil, err
		}
		sort.Sort(TagsSortByDateDes(tags))
		return tags, nil
	}

	return nil, fmt.Errorf("%s", body)
}

func (c *Client) DeleteTag(projectName, repoName, tag string) error {
	path := TagPath(projectName, repoName, tag)

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
		return nil
	}

	return fmt.Errorf("%s", body)
}

func (c *Client) GetTagManifest(projectName, repoName, tag string) (*TagManifest, error) {
	path := ImageManifestPath(projectName, repoName, tag)

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
		ret := &TagManifest{}
		err := json.Unmarshal(body, &ret)
		if err != nil {
			return nil, err
		}

		return ret, nil
	}

	return nil, fmt.Errorf("%s", body)
}
