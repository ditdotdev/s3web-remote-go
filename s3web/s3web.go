/*
 * Copyright The Titan Project Contributors.
 */
package s3web

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/datadatdat/remote-sdk-go/remote"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

type s3webRemote struct {
}

func (s s3webRemote) Type() (string, error) {
	return "s3web", nil
}

func (s s3webRemote) FromURL(rawUrl string, additionalProperties map[string]string) (map[string]interface{}, error) {
	u, err := url.Parse(rawUrl)
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
		return nil, errors.New(fmt.Sprintf("invalid property '%s'", reflect.ValueOf(additionalProperties).MapKeys()[0].String()))
	}

	res := fmt.Sprintf("http://%s%s", u.Host, u.Path)
	return map[string]interface{}{"url": res}, nil
}

func (s s3webRemote) ToURL(properties map[string]interface{}) (string, map[string]string, error) {
	u := properties["url"].(string)
	return strings.Replace(u, "http", "s3web", 1), map[string]string{}, nil
}

func (s s3webRemote) GetParameters(remoteProperties map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (s s3webRemote) ValidateRemote(properties map[string]interface{}) error {
	return remote.ValidateFields(properties, []string{"url"}, []string{})
}

func (s s3webRemote) ValidateParameters(parameters map[string]interface{}) error {
	return remote.ValidateFields(parameters, []string{}, []string{})
}

var httpGet = http.Get

type MetadataCommit struct {
	Id         string                 `json:"id"`
	Properties map[string]interface{} `json:"properties"`
}

func (s s3webRemote) ListCommits(properties map[string]interface{}, parameters map[string]interface{}, tags []remote.Tag) ([]remote.Commit, error) {
	metadataPath := fmt.Sprintf("%s/%s", properties["url"], "titan")
	resp, err := httpGet(metadataPath)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return []remote.Commit{}, nil
	}

	if resp.StatusCode >= 300 {
		b, err := ioutil.ReadAll(resp.Body)
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
			if err == nil && commit.Properties != nil && commit.Id != "" && remote.MatchTags(commit.Properties, tags) {
				ret = append(ret, remote.Commit{Id: commit.Id, Properties: commit.Properties})
			}
		}
	}

	remote.SortCommits(ret)

	return ret, nil
}

func (s s3webRemote) GetCommit(properties map[string]interface{}, parameters map[string]interface{}, commitId string) (*remote.Commit, error) {
	commits, err := s.ListCommits(properties, parameters, []remote.Tag{})
	if err != nil {
		return nil, err
	}
	for _, c := range commits {
		if c.Id == commitId {
			return &c, nil
		}
	}

	return nil, nil
}

func init() {
	remote.Register(s3webRemote{})
}
