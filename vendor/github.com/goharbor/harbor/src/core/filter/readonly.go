// Copyright 2018 Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filter

import (
	"net/http"
	"regexp"

	"github.com/astaxie/beego/context"
	"github.com/goharbor/harbor/src/common/utils/log"
	"github.com/goharbor/harbor/src/core/config"
)

const (
	repoURL  = `^/api/repositories/((?:[a-z0-9]+(?:[._-][a-z0-9]+)*/)+)(?:[a-z0-9]+(?:[._-][a-z0-9]+)*)$`
	tagsURL  = `^/api/repositories/((?:[a-z0-9]+(?:[._-][a-z0-9]+)*/)+)tags$`
	tagURL   = `^/api/repositories/((?:[a-z0-9]+(?:[._-][a-z0-9]+)*/)+)tags/([\w][\w.-]{0,127})$`
	labelURL = `^/api/repositories/((?:[a-z0-9]+(?:[._-][a-z0-9]+)*/)+)tags/([\w][\w.-]{0,127})/labels/[0-9]+$`
)

// ReadonlyFilter filters the deletion or creation (e.g. retag) of repo/tag requests and returns 503.
func ReadonlyFilter(ctx *context.Context) {
	filter(ctx.Request, ctx.ResponseWriter)
}

func filter(req *http.Request, resp http.ResponseWriter) {
	if !config.ReadOnly() {
		return
	}

	if matchRepoTagDelete(req) || matchRetag(req) {
		resp.WriteHeader(http.StatusServiceUnavailable)
		_, err := resp.Write([]byte("The system is in read only mode. Any modification is prohibited."))
		if err != nil {
			log.Errorf("failed to write response body: %v", err)
		}
	}
}

// matchRepoTagDelete checks whether a request is a repository or tag deletion request,
// it should be blocked in read-only mode.
func matchRepoTagDelete(req *http.Request) bool {
	if req.Method != http.MethodDelete {
		return false
	}

	if inWhiteList(req) {
		return false
	}

	re := regexp.MustCompile(tagURL)
	s := re.FindStringSubmatch(req.URL.Path)
	if len(s) == 3 {
		return true
	}

	re = regexp.MustCompile(repoURL)
	s = re.FindStringSubmatch(req.URL.Path)
	if len(s) == 2 {
		return true
	}

	return false
}

// matchRetag checks whether a request is a retag request, it should be blocked in read-only mode.
func matchRetag(req *http.Request) bool {
	if req.Method != http.MethodPost {
		return false
	}

	re := regexp.MustCompile(tagsURL)
	return re.MatchString(req.URL.Path)
}

func inWhiteList(req *http.Request) bool {
	re := regexp.MustCompile(labelURL)
	s := re.FindStringSubmatch(req.URL.Path)
	if len(s) == 3 {
		return true
	}
	return false
}
