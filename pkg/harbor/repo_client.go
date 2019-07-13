package harbor

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/goharbor/harbor/src/common/utils"
	"github.com/goharbor/harbor/src/common/utils/registry"
	"github.com/goharbor/harbor/src/common/utils/registry/auth"
)

type RepoClient struct {
	Name     string
	Endpoint *url.URL
	client   *http.Client
}

type userAgentModifier struct {
	userAgent string
}

// Modify adds user-agent header to the request
func (u *userAgentModifier) Modify(req *http.Request) error {
	req.Header.Set(http.CanonicalHeaderKey("User-Agent"), u.userAgent)
	return nil
}

func NewRepoClient(client *Client, repository string) (*RepoClient, error) {

	transport := registry.GetHTTPTransport(true)

	authorizer := auth.NewStandardTokenAuthorizer(&http.Client{
		Transport: transport,
	}, auth.NewBasicAuthCredential(client.config.Auth.User, client.config.Auth.Password), "")

	uam := &userAgentModifier{
		userAgent: "harbor-registry-client",
	}

	u, err := utils.ParseEndpoint(client.baseURL)
	if err != nil {
		return nil, err
	}

	return &RepoClient{
		Name:     strings.TrimSpace(repository),
		Endpoint: u,
		client: &http.Client{
			Transport: registry.NewTransport(transport, authorizer, uam),
		},
	}, nil
}

func (r *RepoClient) PullManifest(reference string, acceptMediaTypes []string) (digest, mediaType string, payload []byte, err error) {
	req, err := http.NewRequest("GET", buildManifestURL(r.Endpoint.String(), r.Name, reference), nil)
	if err != nil {
		return
	}

	for _, mediaType := range acceptMediaTypes {
		req.Header.Add(http.CanonicalHeaderKey("Accept"), mediaType)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode == http.StatusOK {
		digest = resp.Header.Get(http.CanonicalHeaderKey("Docker-Content-Digest"))
		mediaType = resp.Header.Get(http.CanonicalHeaderKey("Content-Type"))
		payload = b
		return
	}

	err = fmt.Errorf("status %d, body: %s", resp.StatusCode, string(b))

	return
}

func (r *RepoClient) MountBlob(digest, from string) error {
	req, err := http.NewRequest("POST", buildMountBlobURL(r.Endpoint.String(), r.Name, digest, from), nil)
	req.Header.Set(http.CanonicalHeaderKey("Content-Length"), "0")

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("status %d, body: %s", resp.StatusCode, string(b))
	}

	return nil
}

func (r *RepoClient) ManifestExist(reference string) (digest string, exist bool, err error) {
	req, err := http.NewRequest("HEAD", buildManifestURL(r.Endpoint.String(), r.Name, reference), nil)
	if err != nil {
		return
	}

	req.Header.Add(http.CanonicalHeaderKey("Accept"), schema1.MediaTypeManifest)
	req.Header.Add(http.CanonicalHeaderKey("Accept"), schema2.MediaTypeManifest)

	resp, err := r.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		exist = true
		digest = resp.Header.Get(http.CanonicalHeaderKey("Docker-Content-Digest"))
		return
	}

	if resp.StatusCode == http.StatusNotFound {
		return
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = fmt.Errorf("status %d, body: %s", resp.StatusCode, string(b))

	return
}

func (r *RepoClient) PushManifest(reference, mediaType string, payload []byte) (digest string, err error) {
	req, err := http.NewRequest("PUT", buildManifestURL(r.Endpoint.String(), r.Name, reference),
		bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set(http.CanonicalHeaderKey("Content-Type"), mediaType)

	resp, err := r.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		digest = resp.Header.Get(http.CanonicalHeaderKey("Docker-Content-Digest"))
		return
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = fmt.Errorf("status %d, body: %s", resp.StatusCode, string(b))

	return
}

func buildManifestURL(endpoint, repoName, reference string) string {
	return fmt.Sprintf("%s/v2/%s/manifests/%s", endpoint, repoName, reference)
}

func buildMountBlobURL(endpoint, repoName, digest, from string) string {
	return fmt.Sprintf("%s/v2/%s/blobs/uploads/?mount=%s&from=%s", endpoint, repoName, digest, from)
}
