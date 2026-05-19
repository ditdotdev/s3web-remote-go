/*
 * Copyright Datadatdat.
 */
package s3web

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/datadatdat/remote-sdk-go/remote"
	"github.com/stretchr/testify/assert"
)

// trackingReadCloser wraps an io.Reader and counts Close() invocations so tests
// can assert that response bodies are properly closed.
type trackingReadCloser struct {
	io.Reader
	closes int32
}

func (t *trackingReadCloser) Close() error {
	atomic.AddInt32(&t.closes, 1)
	return nil
}

func (t *trackingReadCloser) Closes() int32 {
	return atomic.LoadInt32(&t.closes)
}

const testMetadata = `
{"id": "one", "properties": {"timestamp": "2019-09-20T13:45:36Z"}}
{"id": "two", "properties": {"timestamp": "2019-09-20T13:45:37Z"}}`

const (
	testURL    = "http://host/path"
	testBucket = "bucket"
	testPath   = "path"
)

func TestRegistered(t *testing.T) {
	r, _ := remote.Get("s3web")

	ret, err := r.Type()
	if assert.NoError(t, err) {
		assert.Equal(t, "s3web", ret)
	}
}

func TestFromURL(t *testing.T) {
	r, _ := remote.Get("s3web")

	props, err := r.FromURL("s3web://host/object/path", map[string]string{})
	if assert.NoError(t, err) {
		assert.Equal(t, "http://host/object/path", props[propURL])
	}
}

func TestNoPath(t *testing.T) {
	r, _ := remote.Get("s3web")

	props, err := r.FromURL("s3web://host", map[string]string{})
	if assert.NoError(t, err) {
		assert.Equal(t, "http://host", props[propURL])
	}
}

func TestBadProperty(t *testing.T) {
	r, _ := remote.Get("s3web")
	_, err := r.FromURL("s3web://host", map[string]string{"a": "b"})
	assert.Error(t, err)
}

func TestBadUrl(t *testing.T) {
	r, _ := remote.Get("s3web")
	_, err := r.FromURL("s3web://not\nhost", map[string]string{})
	assert.Error(t, err)
}

func TestBadScheme(t *testing.T) {
	r, _ := remote.Get("s3web")
	_, err := r.FromURL("s3web", map[string]string{})
	assert.Error(t, err)
}

func TestBadSchemeName(t *testing.T) {
	r, _ := remote.Get("s3web")
	_, err := r.FromURL("foo://bar", map[string]string{})
	assert.Error(t, err)
}

func TestBadUser(t *testing.T) {
	r, _ := remote.Get("s3web")
	_, err := r.FromURL("s3web://user@host/path", map[string]string{})
	assert.Error(t, err)
}

func TestBadUserPassword(t *testing.T) {
	r, _ := remote.Get("s3web")
	_, err := r.FromURL("s3web://user:password@host/path", map[string]string{})
	assert.Error(t, err)
}

func TestBadNoHost(t *testing.T) {
	r, _ := remote.Get("s3web")
	_, err := r.FromURL("s3web:///path", map[string]string{})
	assert.Error(t, err)
}

func TestPort(t *testing.T) {
	r, _ := remote.Get("s3web")

	props, err := r.FromURL("s3web://host:1023/object/path", map[string]string{})
	if assert.NoError(t, err) {
		assert.Equal(t, "http://host:1023/object/path", props[propURL])
	}
}

func TestToURL(t *testing.T) {
	r, _ := remote.Get("s3web")

	u, props, err := r.ToURL(map[string]interface{}{propURL: testURL})
	if assert.NoError(t, err) {
		assert.Equal(t, "s3web://host/path", u)
		assert.Empty(t, props)
	}
}

func TestParameters(t *testing.T) {
	r, _ := remote.Get("s3web")

	props, err := r.GetParameters(map[string]interface{}{propURL: testURL})
	if assert.NoError(t, err) {
		assert.Empty(t, props)
	}
}

func TestValidateRemoteRequired(t *testing.T) {
	r, _ := remote.Get("s3web")
	err := r.ValidateRemote(map[string]interface{}{propURL: propURL})
	assert.NoError(t, err)
}

func TestValidateRemoteMissingRequired(t *testing.T) {
	r, _ := remote.Get("s3web")
	err := r.ValidateRemote(map[string]interface{}{})
	assert.Error(t, err)
}

func TestValidateRemoteInvalid(t *testing.T) {
	r, _ := remote.Get("s3web")
	err := r.ValidateRemote(map[string]interface{}{propURL: propURL, "foo": "bar"})
	assert.Error(t, err)
}

func TestValidateParametersEmpty(t *testing.T) {
	r, _ := remote.Get("s3web")
	err := r.ValidateParameters(map[string]interface{}{})
	assert.NoError(t, err)
}

func TestValidateParametersInvalid(t *testing.T) {
	r, _ := remote.Get("s3web")
	err := r.ValidateParameters(map[string]interface{}{"foo": "bar"})
	assert.Error(t, err)
}

// setHTTPGet swaps the package-level httpGet for the duration of a test and
// restores it via t.Cleanup so a panicking test cannot leave the global in a
// mock state.
func setHTTPGet(t *testing.T, fn func(string) (*http.Response, error)) {
	t.Helper()
	prev := httpGet
	httpGet = fn
	t.Cleanup(func() { httpGet = prev })
}

func TestListCommitsBadGet(t *testing.T) {
	setHTTPGet(t, func(_ string) (resp *http.Response, err error) {
		return nil, errors.New("error")
	})
	r, _ := remote.Get("s3web")
	_, err := r.ListCommits(map[string]interface{}{propURL: testURL}, map[string]interface{}{},
		[]remote.Tag{})
	assert.Error(t, err)
}

func TestListCommitsNotFound(t *testing.T) {
	tracker := &trackingReadCloser{Reader: strings.NewReader("")}
	setHTTPGet(t, func(_ string) (resp *http.Response, err error) {
		return &http.Response{StatusCode: http.StatusNotFound, Body: tracker}, nil
	})
	r, _ := remote.Get("s3web")

	commits, err := r.ListCommits(map[string]interface{}{propURL: testURL}, map[string]interface{}{},
		[]remote.Tag{})
	if assert.NoError(t, err) {
		assert.Len(t, commits, 0)
	}
	assert.Equal(t, int32(1), tracker.Closes(), "404 path must close response body")
}

func TestListCommitsOtherError(t *testing.T) {
	tracker := &trackingReadCloser{Reader: strings.NewReader("bad request")}
	setHTTPGet(t, func(_ string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       tracker,
		}, nil
	})
	r, _ := remote.Get("s3web")
	_, err := r.ListCommits(map[string]interface{}{propURL: testURL}, map[string]interface{}{},
		[]remote.Tag{})
	assert.Error(t, err)
	assert.Equal(t, int32(1), tracker.Closes(), "error path must close response body")
}

type errReader int

func (errReader) Read(_ []byte) (n int, err error) {
	return 0, errors.New("test error")
}

func TestListCommitsErrorReadError(t *testing.T) {
	setHTTPGet(t, func(_ string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(errReader(0)),
		}, nil
	})
	r, _ := remote.Get("s3web")
	_, err := r.ListCommits(map[string]interface{}{propURL: testURL}, map[string]interface{}{},
		[]remote.Tag{})
	assert.Error(t, err)
}

// errScanReader returns valid data once, then errors on the next Read. Used to
// exercise the scanner.Err() check after the ListCommits loop.
type errScanReader struct {
	data []byte
	read bool
}

func (e *errScanReader) Read(p []byte) (int, error) {
	if !e.read {
		e.read = true
		n := copy(p, e.data)
		return n, nil
	}
	return 0, errors.New("scanner read error")
}

func TestListCommitsScannerError(t *testing.T) {
	// Provide a single line of bytes (no trailing newline) followed by a read
	// error. bufio.Scanner buffers the partial line and surfaces the error via
	// scanner.Err() after the loop terminates.
	body := &errScanReader{data: []byte("partial-line-no-newline")}
	setHTTPGet(t, func(_ string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(body),
		}, nil
	})
	r, _ := remote.Get("s3web")
	_, err := r.ListCommits(map[string]interface{}{propURL: testURL}, map[string]interface{}{},
		[]remote.Tag{})
	assert.Error(t, err, "ListCommits must propagate scanner errors")
}

func TestListCommits(t *testing.T) {
	metadata := testMetadata
	tracker := &trackingReadCloser{Reader: strings.NewReader(metadata)}
	setHTTPGet(t, func(_ string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       tracker,
		}, nil
	})
	r, _ := remote.Get("s3web")

	commits, err := r.ListCommits(map[string]interface{}{testBucket: testBucket, testPath: testPath}, map[string]interface{}{}, []remote.Tag{})
	if assert.NoError(t, err) {
		assert.Len(t, commits, 2)
		assert.Equal(t, "two", commits[0].ID)
		assert.Equal(t, "one", commits[1].ID)
	}
	assert.Equal(t, int32(1), tracker.Closes(), "success path must close response body exactly once")
}

func TestListCommitsInvalid(t *testing.T) {
	metadata := `
foo
{"id": "two", "properties": {"timestamp": "2019-09-20T13:45:37Z"}}`
	setHTTPGet(t, func(_ string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(metadata)),
		}, nil
	})
	r, _ := remote.Get("s3web")

	commits, err := r.ListCommits(map[string]interface{}{testBucket: testBucket, testPath: testPath}, map[string]interface{}{}, []remote.Tag{})
	if assert.NoError(t, err) {
		assert.Len(t, commits, 1)
		assert.Equal(t, "two", commits[0].ID)
	}
}

func TestListCommitsTags(t *testing.T) {
	metadata := `
{"id": "one", "properties": {"timestamp": "2019-09-20T13:45:36Z", "tags": { "a": "b" }}}
{"id": "two", "properties": {"timestamp": "2019-09-20T13:45:37Z", "tags": { "c": "d" }}}`
	setHTTPGet(t, func(_ string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(metadata)),
		}, nil
	})
	r, _ := remote.Get("s3web")

	commits, err := r.ListCommits(map[string]interface{}{testBucket: testBucket, testPath: testPath}, map[string]interface{}{}, []remote.Tag{{Key: "a"}})
	if assert.NoError(t, err) {
		assert.Len(t, commits, 1)
		assert.Equal(t, "one", commits[0].ID)
	}
}

func TestGetCommitError(t *testing.T) {
	setHTTPGet(t, func(_ string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader("bad request")),
		}, nil
	})
	r, _ := remote.Get("s3web")
	_, err := r.GetCommit(map[string]interface{}{propURL: testURL}, map[string]interface{}{}, "id")
	assert.Error(t, err)
}

func TestGetCommit(t *testing.T) {
	metadata := testMetadata
	setHTTPGet(t, func(_ string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(metadata)),
		}, nil
	})
	r, _ := remote.Get("s3web")

	commit, err := r.GetCommit(map[string]interface{}{testBucket: testBucket, testPath: testPath}, map[string]interface{}{}, "one")
	if assert.NoError(t, err) {
		assert.Equal(t, "one", commit.ID)
		assert.Equal(t, "2019-09-20T13:45:36Z", commit.Properties["timestamp"])
	}
}

func TestGetMissingCommit(t *testing.T) {
	metadata := testMetadata
	setHTTPGet(t, func(_ string) (resp *http.Response, err error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(metadata)),
		}, nil
	})
	r, _ := remote.Get("s3web")

	commit, err := r.GetCommit(map[string]interface{}{testBucket: testBucket, testPath: testPath}, map[string]interface{}{}, "three")
	assert.Nil(t, commit)
	assert.Error(t, err, "GetCommit must return an error when the requested commit ID is not in the metadata")
}

// TestToURLWrongType asserts that ToURL returns an error (rather than panicking)
// when the propURL value is not a string.
func TestToURLWrongType(t *testing.T) {
	r, _ := remote.Get("s3web")
	assert.NotPanics(t, func() {
		u, props, err := r.ToURL(map[string]interface{}{propURL: 42})
		assert.Error(t, err)
		assert.Empty(t, u)
		assert.Empty(t, props)
	})
}

// TestToURLMissing asserts that ToURL returns an error when propURL is absent.
func TestToURLMissing(t *testing.T) {
	r, _ := remote.Get("s3web")
	assert.NotPanics(t, func() {
		u, props, err := r.ToURL(map[string]interface{}{})
		assert.Error(t, err)
		assert.Empty(t, u)
		assert.Empty(t, props)
	})
}

// TestHTTPClientHasTimeout asserts that the package's underlying http.Client
// has a non-zero timeout so a hung remote cannot block indefinitely.
func TestHTTPClientHasTimeout(t *testing.T) {
	assert.NotZero(t, httpClient.Timeout, "httpClient must have a non-zero timeout")
}

// TestHTTPGetSendsUserAgent exercises the real (non-mocked) httpGet against a
// local test server. It verifies the User-Agent header is set on requests and
// that the function correctly issues GETs through httpClient. Bundled together
// to keep the test setup minimal while still hitting all branches of httpGet.
func TestHTTPGetSendsUserAgent(t *testing.T) {
	var gotUA string
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	resp, err := httpGet(srv.URL)
	if assert.NoError(t, err) {
		t.Cleanup(func() { _ = resp.Body.Close() })
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}
	assert.Equal(t, userAgent, gotUA, "httpGet must set User-Agent header")
	assert.Equal(t, http.MethodGet, gotMethod)
}

// TestHTTPGetBadURL covers the NewRequest error path in httpGet (invalid URL
// containing a control character).
func TestHTTPGetBadURL(t *testing.T) {
	_, err := httpGet("http://exa\x7fmple.com")
	assert.Error(t, err)
}
