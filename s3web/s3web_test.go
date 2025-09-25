/*
 * Copyright The Titan Project Contributors.
 */
package s3web

import (
	"errors"
	"github.com/datadatdat/remote-sdk-go/remote"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"strings"
	"testing"
)

const testMetadata = `
{"id": "one", "properties": {"timestamp": "2019-09-	httpGet	httpGet = func(_ string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(metadata)),
		}, nil
	}
	r := remote.Get("s3web")

	commits, err := r.ListCommits(map[string]interface{}{"url": "http://host/path"}, map[string]interface{}{}, string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(metadata)),
		}, nil
	}
	r := remote.Get("s3web")

	commits, err := r.ListCommits(map[string]interface{}{"url": "http://host/path"}, map[string]interface{}{},36Z"}}
{"id": "two", "properties": {"timestamp": "2019-09-20T13:45:37Z"}}`

func TestRegistered(t *testing.T) {
	r := remote.Get("s3web")

	ret, err := r.Type()
	if assert.NoError(t, err) {
		assert.Equal(t, "s3web", ret)
	}
}

func TestFromURL(t *testing.T) {
	r := remote.Get("s3web")

	props, err := r.FromURL("s3web://host/object/path", map[string]string{})
	if assert.NoError(t, err) {
		assert.Equal(t, "http://host/object/path", props["url"])
	}
}

func TestNoPath(t *testing.T) {
	r := remote.Get("s3web")

	props, err := r.FromURL("s3web://host", map[string]string{})
	if assert.NoError(t, err) {
		assert.Equal(t, "http://host", props["url"])
	}
}

func TestBadProperty(t *testing.T) {
	r := remote.Get("s3web")
	_, err := r.FromURL("s3web://host", map[string]string{"a": "b"})
	assert.Error(t, err)
}

func TestBadUrl(t *testing.T) {
	r := remote.Get("s3web")
	_, err := r.FromURL("s3web://not\nhost", map[string]string{})
	assert.Error(t, err)
}

func TestBadScheme(t *testing.T) {
	r := remote.Get("s3web")
	_, err := r.FromURL("s3web", map[string]string{})
	assert.Error(t, err)
}

func TestBadSchemeName(t *testing.T) {
	r := remote.Get("s3web")
	_, err := r.FromURL("foo://bar", map[string]string{})
	assert.Error(t, err)
}

func TestBadUser(t *testing.T) {
	r := remote.Get("s3web")
	_, err := r.FromURL("s3web://user@host/path", map[string]string{})
	assert.Error(t, err)
}

func TestBadUserPassword(t *testing.T) {
	r := remote.Get("s3web")
	_, err := r.FromURL("s3web://user:password@host/path", map[string]string{})
	assert.Error(t, err)
}

func TestBadNoHost(t *testing.T) {
	r := remote.Get("s3web")
	_, err := r.FromURL("s3web:///path", map[string]string{})
	assert.Error(t, err)
}

func TestPort(t *testing.T) {
	r := remote.Get("s3web")

	props, err := r.FromURL("s3web://host:1023/object/path", map[string]string{})
	if assert.NoError(t, err) {
		assert.Equal(t, "http://host:1023/object/path", props["url"])
	}
}

func TestToURL(t *testing.T) {
	r := remote.Get("s3web")

	u, props, err := r.ToURL(map[string]interface{}{"url": "http://host/path"})
	if assert.NoError(t, err) {
		assert.Equal(t, "s3web://host/path", u)
		assert.Empty(t, props)
	}
}

func TestParameters(t *testing.T) {
	r := remote.Get("s3web")

	props, err := r.GetParameters(map[string]interface{}{"url": "http://host/path"})
	if assert.NoError(t, err) {
		assert.Empty(t, props)
	}
}

func TestValidateRemoteRequired(t *testing.T) {
	r := remote.Get("s3web")
	err := r.ValidateRemote(map[string]interface{}{"url": "url"})
	assert.NoError(t, err)
}

func TestValidateRemoteMissingRequired(t *testing.T) {
	r := remote.Get("s3web")
	err := r.ValidateRemote(map[string]interface{}{})
	assert.Error(t, err)
}

func TestValidateRemoteInvalid(t *testing.T) {
	r := remote.Get("s3web")
	err := r.ValidateRemote(map[string]interface{}{"url": "url", "foo": "bar"})
	assert.Error(t, err)
}

func TestValidateParametersEmpty(t *testing.T) {
	r := remote.Get("s3web")
	err := r.ValidateParameters(map[string]interface{}{})
	assert.NoError(t, err)
}

func TestValidateParametersInvalid(t *testing.T) {
	r := remote.Get("s3web")
	err := r.ValidateParameters(map[string]interface{}{"foo": "bar"})
	assert.Error(t, err)
}

func TestListCommitsBadGet(t *testing.T) {
	httpGet = func(_ string) (resp *http.Response, err error) {
		return nil, errors.New("error")
	}
	r := remote.Get("s3web")
	_, err := r.ListCommits(map[string]interface{}{"url": "http://host/path"}, map[string]interface{}{},
		[]remote.Tag{})
	assert.Error(t, err)

	httpGet = http.Get
}

func TestListCommitsNotFound(t *testing.T) {
	httpGet = func(_ string) (resp *http.Response, err error) {
		return &http.Response{StatusCode: http.StatusNotFound}, nil
	}
	r := remote.Get("s3web")

	commits, err := r.ListCommits(map[string]interface{}{"url": "http://host/path"}, map[string]interface{}{},
		[]remote.Tag{})
	if assert.NoError(t, err) {
		assert.Len(t, commits, 0)
	}

	httpGet = http.Get
}

func TestListCommitsOtherError(t *testing.T) {
	httpGet = func(_ string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader("bad request")),
		}, nil
	}
	r := remote.Get("s3web")
	_, err := r.ListCommits(map[string]interface{}{"url": "http://host/path"}, map[string]interface{}{},
		[]remote.Tag{})
	assert.Error(t, err)

	httpGet = http.Get
}

type errReader int

func (errReader) Read(_ []byte) (n int, err error) {
	return 0, errors.New("test error")
}

func TestListCommitsErrorReadError(t *testing.T) {
	httpGet = func(_ string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(errReader(0)),
		}, nil
	}
	r := remote.Get("s3web")
	_, err := r.ListCommits(map[string]interface{}{"url": "http://host/path"}, map[string]interface{}{},
		[]remote.Tag{})
	assert.Error(t, err)

	httpGet = http.Get
}

func TestListCommits(t *testing.T) {
	metadata := testMetadata
	httpGet = func(_ string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(metadata)),
		}, nil
	}
	r := remote.Get("s3web")

	commits, err := r.ListCommits(map[string]interface{}{"bucket": "bucket", "path": "path"}, map[string]interface{}{}, []remote.Tag{})
	if assert.NoError(t, err) {
		assert.Len(t, commits, 2)
		assert.Equal(t, "two", commits[0].Id)
		assert.Equal(t, "one", commits[1].Id)
	}

	httpGet = http.Get
}

func TestListCommitsInvalid(t *testing.T) {
	metadata := `
foo
{"id": "two", "properties": {"timestamp": "2019-09-20T13:45:37Z"}}`
	httpGet = func(_ string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(metadata)),
		}, nil
	}
	r := remote.Get("s3web")

	commits, err := r.ListCommits(map[string]interface{}{"bucket": "bucket", "path": "path"}, map[string]interface{}{}, []remote.Tag{})
	if assert.NoError(t, err) {
		assert.Len(t, commits, 1)
		assert.Equal(t, "two", commits[0].Id)
	}

	httpGet = http.Get
}

func TestListCommitsTags(t *testing.T) {
	metadata := `
{"id": "one", "properties": {"timestamp": "2019-09-20T13:45:36Z", "tags": { "a": "b" }}}
{"id": "two", "properties": {"timestamp": "2019-09-20T13:45:37Z", "tags": { "c": "d" }}}`
	httpGet = func(_ string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(metadata)),
		}, nil
	}
	r := remote.Get("s3web")

	commits, err := r.ListCommits(map[string]interface{}{"bucket": "bucket", "path": "path"}, map[string]interface{}{}, []remote.Tag{{Key: "a"}})
	if assert.NoError(t, err) {
		assert.Len(t, commits, 1)
		assert.Equal(t, "one", commits[0].Id)
	}

	httpGet = http.Get
}

func TestGetCommitError(t *testing.T) {
	httpGet = func(_ string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader("bad request")),
		}, nil
	}
	r := remote.Get("s3web")
	_, err := r.GetCommit(map[string]interface{}{"url": "http://host/path"}, map[string]interface{}{}, "id")
	assert.Error(t, err)

	httpGet = http.Get
}

func TestGetCommit(t *testing.T) {
	metadata := testMetadata
	httpGet = func(_ string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(metadata)),
		}, nil
	}
	r := remote.Get("s3web")

	commit, err := r.GetCommit(map[string]interface{}{"bucket": "bucket", "path": "path"}, map[string]interface{}{}, "one")
	if assert.NoError(t, err) {
		assert.Equal(t, "one", commit.Id)
		assert.Equal(t, "2019-09-20T13:45:36Z", commit.Properties["timestamp"])
	}

	httpGet = http.Get
}

func TestGetMissingCommit(t *testing.T) {
	metadata := testMetadata
	httpGet = func(_ string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(metadata)),
		}, nil
	}
	r := remote.Get("s3web")

	commit, err := r.GetCommit(map[string]interface{}{"bucket": "bucket", "path": "path"}, map[string]interface{}{}, "three")
	if assert.NoError(t, err) {
		assert.Nil(t, commit)
	}

	httpGet = http.Get
}
