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
	path := TagsPath(projectName, repoName, c.config.Version)

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

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("%s", string(body))
	}

	tags := make([]*Tag, 0)
	if c.config.Version > "0.4" {
		err = json.Unmarshal(body, &tags)
		if err != nil {
			logrus.Errorf("unmarshal tags error: %v", err)
			logrus.Infof("resp body: %s", body)
			return nil, err
		}

		sort.Sort(TagsSortByDateDes(tags))
		return tags, nil
	}

	var tagNames []string
	err = json.Unmarshal(body, &tagNames)
	if err != nil {
		logrus.Errorf("unmarshal tags error: %v", err)
		logrus.Infof("resp body: %s", body)
		return nil, err
	}
	for _, t := range tagNames {
		m, err := c.Get04TagDigest(projectName, repoName, t)
		if err != nil {
			return nil, err
		}
		tags = append(tags, &Tag{
			tagDetail: tagDetail{
				Name:   t,
				Digest: m,
			},
		})
	}
	sort.Sort(TagsSortByNameDes(tags))

	return tags, nil
}

func (c *Client) DeleteTag(projectName, repoName, tag string) error {
	path := TagPath(projectName, repoName, tag, c.config.Version)

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
	path := ImageManifestPath(projectName, repoName, tag, c.config.Version)

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

func (c *Client) Get04TagDigest(projectName, repoName, tag string) (string, error) {
	path := ImageManifestPath(projectName, repoName, tag, c.config.Version)

	resp, err := c.do(http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode/100 == 2 {
		ret := &Manifest04{}
		err := json.Unmarshal(body, &ret)
		if err != nil {
			return "", err
		}

		return ret.Manifest.Config.Digest, nil
	}

	return "", fmt.Errorf("%s", body)
}
