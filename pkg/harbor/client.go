package harbor

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/cd1989/harbor-cleaner/pkg/config"
)

const (
	tickerDuration = time.Minute * 30
)

type Client struct {
	config   *config.C
	baseURL  string
	client   *http.Client
	coockies []*http.Cookie
}

var APIClient *Client

func NewClient(conf *config.C, closing <-chan struct{}) (*Client, error) {
	baseURL := strings.TrimRight(conf.Host, "/")
	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	cookies, err := LoginAndGetCookies(client, conf)
	if err != nil {
		logrus.Errorf("login harbor: %s error: %v during background", conf.Host, err)
		return nil, err
	}
	logrus.Infof("harbor %s cookies has been refreshed", conf.Host)

	c := &Client{
		config:   conf,
		baseURL:  baseURL,
		client:   client,
		coockies: cookies,
	}

	go c.refreshLoop(closing)
	return c, nil
}

// do creates request and authorizes it if authorizer is not nil
func (c *Client) do(method, relativePath string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + relativePath
	logrus.Infof("%s %s", method, url)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if body != nil || method == http.MethodPost || method == http.MethodPut {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := range c.coockies {
		req.AddCookie(c.coockies[i])
	}

	resp, err := c.client.Do(req)
	if err != nil {
		logrus.Errorf("unexpected error: %v", err)
		return nil, err
	}

	if resp.StatusCode/100 == 5 || resp.StatusCode == 401 {
		b, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			return nil, err
		}

		logrus.Errorf("unexpected %d error from harbor: %s", resp.StatusCode, b)
		logrus.Errorf("need to refresh harbor: %s 's cookies now! refreshCookies error: %v", c.config.Host, c.refreshCookies())
		return nil, fmt.Errorf("harbor internal error: %s", b)
	}
	return resp, nil
}

func (c *Client) refreshCookies() error {
	cookies, err := LoginAndGetCookies(c.client, c.config)
	if err != nil {
		logrus.Errorf("refresh harbor: %s 's cookies error: %v", c.config.Host, err)
		return err
	}
	c.coockies = cookies
	return nil
}

func (c *Client) refreshLoop(closing <-chan struct{}) {
	ticker := time.NewTicker(tickerDuration)

	for {
		select {
		case <-ticker.C:
			err := c.refreshCookies()
			if err != nil {
				logrus.Errorf("Refresh cookies error: %v", err)
			}
		case <-closing:
			logrus.Info("capture closing signal, refresh goroutine will exit")
			return
		}
	}
}

func LoginAndGetCookies(client *http.Client, conf *config.C) ([]*http.Cookie, error) {
	url := LoginUrl(conf.Host, conf.Version, conf.Auth.User, conf.Auth.Password)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		logrus.Errorf("login harbor: %s error: %s", conf.Host, b)
		return nil, fmt.Errorf("%s", b)
	}

	// If status code is 200 and no cookies set, it's not a valid harbor. For example, 1.1.1.1
	if string(b) != "" || len(resp.Cookies()) == 0 {
		return nil, fmt.Errorf("%s is not a valid harbor", conf.Host)
	}

	return resp.Cookies(), nil
}
