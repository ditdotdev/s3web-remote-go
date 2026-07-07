// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

// Package s3web provides an S3-compatible web interface remote storage backend for Dit.
package s3web

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ditdotdev/remote-sdk-go/remote"
)

const (
	propURL   = "url"
	userAgent = "dit-s3web-remote-go"
	// httpTimeout bounds the time a single HTTP request can take so a hung
	// S3-website endpoint cannot block a plugin operation indefinitely.
	httpTimeout = 30 * time.Second
)

type s3webRemote struct {
}

func (s s3webRemote) Type() (string, error) {
	return "s3web", nil
}

func (s s3webRemote) FromURL(rawURL string, additionalProperties map[string]string) (map[string]interface{}, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "s3web" {
		return nil, errors.New("invalid remote scheme")
	}

	if u.User != nil {
		return nil, errors.New("remote username and password cannot be specified")
	}

	if u.Hostname() == "" {
		return nil, errors.New("missing remote host name")
	}

	if len(additionalProperties) != 0 {
		for k := range additionalProperties {
			return nil, fmt.Errorf("invalid property '%s'", k)
		}
	}

	res := fmt.Sprintf("http://%s%s", u.Host, u.Path)

	return map[string]interface{}{propURL: res}, nil
}

func (s s3webRemote) ToURL(properties map[string]interface{}) (string, map[string]string, error) {
	u, ok := properties[propURL].(string)
	if !ok {
		return "", nil, fmt.Errorf("missing or invalid property '%s'", propURL)
	}
	return strings.Replace(u, "http", "s3web", 1), map[string]string{}, nil
}

func (s s3webRemote) GetParameters(_ map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (s s3webRemote) ValidateRemote(properties map[string]interface{}) error {
	return remote.ValidateFields(properties, []string{propURL}, []string{})
}

func (s s3webRemote) ValidateParameters(parameters map[string]interface{}) error {
	return remote.ValidateFields(parameters, []string{}, []string{})
}

// httpClient is the http.Client used by httpGet. The explicit timeout prevents
// a slow or hung remote endpoint from blocking a plugin operation indefinitely.
var httpClient = &http.Client{Timeout: httpTimeout}

// httpGet issues a GET request via httpClient with a User-Agent header set so
// the request is identifiable in server logs. It is a package-level variable
// so tests can swap in a mock implementation.
var httpGet = func(rawURL string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	return httpClient.Do(req)
}

// MetadataCommit represents a commit metadata structure from the S3 web interface.
type MetadataCommit struct {
	ID         string                 `json:"id"`
	Properties map[string]interface{} `json:"properties"`
}

func (s s3webRemote) ListCommits(properties map[string]interface{}, _ map[string]interface{}, tags []remote.Tag) ([]remote.Commit, error) {
	metadataPath := fmt.Sprintf("%s/%s", properties[propURL], "dit")

	resp, err := httpGet(metadataPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return []remote.Commit{}, nil
	}

	if resp.StatusCode >= 300 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("failed to get '%s': %s", metadataPath, string(b))
	}

	var ret []remote.Commit

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if (line) != "" {
			commit := MetadataCommit{}

			err = json.Unmarshal([]byte(line), &commit)
			if err == nil && commit.Properties != nil && commit.ID != "" && remote.MatchTags(commit.Properties, tags) {
				ret = append(ret, remote.Commit{ID: commit.ID, Properties: commit.Properties})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read commit metadata from '%s': %w", metadataPath, err)
	}

	remote.SortCommits(ret)

	return ret, nil
}

func (s s3webRemote) GetCommit(properties map[string]interface{}, parameters map[string]interface{}, commitID string) (*remote.Commit, error) {
	commits, err := s.ListCommits(properties, parameters, []remote.Tag{})
	if err != nil {
		return nil, err
	}

	for _, c := range commits {
		if c.ID == commitID {
			return &c, nil
		}
	}

	return nil, fmt.Errorf("commit %s not found in remote", commitID)
}

func init() {
	remote.Register(s3webRemote{})
}
