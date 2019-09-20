package harbor

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
)

// ListAllAccessLogs get all access logs from Harbor
func (c *Client) ListAllAccessLogs(startTime, endTime int64) ([]*AccessLog, error) {
	var page int64 = 1
	var pageSize int64 = 500
	var logs []*AccessLog
	for {
		pageLogs, err := c.listAccessLogsPage(startTime, endTime, page, pageSize)
		if err != nil {
			return nil, err
		}
		logs = append(logs, pageLogs...)

		if len(pageLogs) < int(pageSize) {
			break
		}

		page += 1
	}

	return logs, nil
}

func (c *Client) listAccessLogsPage(startTime, endTime, page, pageSize int64) ([]*AccessLog, error) {
	path := AccessLogsPath(startTime, endTime, "", page, pageSize)

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
		logs := make([]*AccessLog, 0)
		err := json.Unmarshal(body, &logs)
		if err != nil {
			logrus.Errorf("unmarshal tags error: %v", err)
			logrus.Infof("resp body: %s", string(body))
			return nil, err
		}
		return logs, nil
	}

	return nil, fmt.Errorf("%s", string(body))
}
