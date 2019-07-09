package harbor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"

	"github.com/sirupsen/logrus"
)

type TagClienter interface {
	ListTags(projectName string, repoName string) ([]*HarborTag, error)
	ListTagNames(projectName, repoName string) ([]string, error)
	GetTag(projectName, repoName, tag string) (*HarborTag, error)
	GetTagVulnerabilities(projectName, repoName, tag string) ([]*HarborVulnerability, error)
	DeleteTag(projectName, repoName, tag string) error
	ScanImage(projectName, repoName, tag string) error
}

func (c *Client) ListTags(projectName string, repoName string) ([]*HarborTag, error) {
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
		tags := make([]*HarborTag, 0)
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

func (c *Client) ListTagNames(projectName, repoName string) ([]string, error) {
	tags, err := c.ListTags(projectName, repoName)
	if err != nil {
		return nil, err
	}

	tagStr := make([]string, 0, len(tags))
	for _, tag := range tags {
		tagStr = append(tagStr, tag.Name)
	}

	return tagStr, nil
}

func (c *Client) GetTag(projectName, repoName, tag string) (*HarborTag, error) {
	path := TagPath(projectName, repoName, tag)

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

	logrus.Infof("%s", body)

	if resp.StatusCode/100 == 2 {
		ret := &HarborTag{}
		err := json.Unmarshal(body, ret)
		if err != nil {
			logrus.Errorf("unmarshal tag error: %v", err)
			logrus.Infof("resp body: %s", body)
			return &HarborTag{}, nil
		}
		return ret, nil
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

func (c *Client) ScanImage(projectName, repoName, tag string) error {
	path := ImageScanPath(projectName, repoName, tag)

	logrus.Infof("%s %s", http.MethodPost, path)
	resp, err := c.do(http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 == 2 {
		return nil
	}

	return fmt.Errorf("unknown error")
}

func (c *Client) GetTagManifest(projectName, repoName, tag string) (*HarborTagManifest, error) {
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
		ret := &HarborTagManifest{}
		err := json.Unmarshal(body, &ret)
		if err != nil {
			return nil, err
		}

		return ret, nil
	}

	return nil, fmt.Errorf("%s", body)
}
