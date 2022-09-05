package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools

	s := testTools.RandomString(10)

	if len(s) != 10 {
		t.Error("wrong length random string returned")
	}
}

var uploadTests = []struct {
	name             string
	allowedFileTypes []string
	renameFile       bool
	errorExpected    bool
}{
	{
		name:             "allowed no rename",
		allowedFileTypes: []string{"image/jpeg", "image/png"},
		renameFile:       false,
		errorExpected:    false},
	{
		name:             "allowed rename",
		allowedFileTypes: []string{"image/jpeg", "image/png"},
		renameFile:       true,
		errorExpected:    false},
	{
		name:             "not allowed",
		allowedFileTypes: []string{"image/jpeg"},
		renameFile:       false,
		errorExpected:    true,
	},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		// Set up a pipe to avoid buffering
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer writer.Close()
			defer wg.Done()

			// Create the form data field 'file'
			part, err := writer.CreateFormFile("file", "./testData/img.png")
			if err != nil {
				t.Error(err)
			}

			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()

		// Read from the pipe which receives data
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = e.allowedFileTypes

		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}

		if !e.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: expected file to exist: %s", e.name, err.Error())
			}

			// Clean up
			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error received but none expected", e.name)
		}

		if e.errorExpected && err == nil {
			t.Errorf("%s: error expected but none received", e.name)
		}

		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	// Set up a pipe to avoid buffering
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()

		// Create the form data field 'file'
		part, err := writer.CreateFormFile("file", "./testData/img.png")
		if err != nil {
			t.Error(err)
		}

		f, err := os.Open("./testdata/img.png")
		if err != nil {
			t.Error(err)
		}
		defer f.Close()

		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("error decoding image", err)
		}

		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}
	}()

	// Read from the pipe which receives data
	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	var testTools Tools

	uploadedFile, err := testTools.UploadOneFile(request, "./testdata/uploads", true)
	if err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", err.Error())
	}
}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var testTool Tools

	err := testTool.CreateDirIfNotExist("./testdata/myDir")
	if err != nil {
		t.Error(err)
	}

	err = testTool.CreateDirIfNotExist("./testdata/myDir")
	if err != nil {
		t.Error(err)
	}

	_ = os.Remove("./textdata/myDir")
}

var slugTests = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{
		name:          "valid string",
		s:             "now is the time",
		expected:      "now-is-the-time",
		errorExpected: false,
	},
	{
		name:          "empty string",
		s:             "",
		expected:      "",
		errorExpected: true,
	},
	{
		name:          "complex string",
		s:             "Now is the time for all GOOD men! _ $%# taco & fish ##@$ 123",
		expected:      "now-is-the-time-for-all-good-men-taco-fish-123",
		errorExpected: false,
	},
	{
		name:          "korean string",
		s:             "한국어",
		expected:      "",
		errorExpected: true,
	},
	{
		name:          "korean and roman string",
		s:             "한국어 now is the time %%% 123",
		expected:      "now-is-the-time-123",
		errorExpected: false,
	},
}

func TestTools_Slugify(t *testing.T) {
	var testTool Tools

	for _, e := range slugTests {
		slug, err := testTool.Slugify(e.s)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: error received when none expected: %s", e.name, err.Error())
		}

		if !e.errorExpected && slug != e.expected {
			t.Errorf("%s: wrong slug returned; expected %s but got %s", e.name, e.expected, slug)
		}

		if err == nil && e.errorExpected {
			t.Errorf("%s: error expected when none receivded: %s", e.name, err.Error())
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	var testTool Tools

	testTool.DownloadStaticFile(rr, req, "./testdata/pic.jpg", "downloaded_pic.jpg")

	res := rr.Result()
	defer res.Body.Close()

	if res.Header["Content-Length"][0] != "98827" {
		t.Error("wrong content length of", res.Header["Content-Length"][0])
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"downloaded_pic.jpg\"" {
		t.Error("wrong content dispostion")
	}

	_, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
}

var jsonTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       int
	allowUnknown  bool
}{
	{
		name:          "good json",
		json:          `{"foo": "bar"}`,
		errorExpected: false,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "badly formatted json",
		json:          `{"foo":}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "incorrect type",
		json:          `{"foo": 1}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "two json files",
		json:          `{"foo": "1"}{"alpha": "beta"}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "empty body",
		json:          ``,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "syntax error in json",
		json:          `{"foo": bar"}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "unknown field in json",
		json:          `{"fooooo": "bar"}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "allow unknown field in json",
		json:          `{"fooooo": "bar"}`,
		errorExpected: false,
		maxSize:       1024,
		allowUnknown:  true,
	},
	{
		name:          "missing field name",
		json:          `{foo: "bar"}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  true,
	},
	{
		name:          "not json",
		json:          `this is not json.`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  true,
	},
	{
		name:          "file too large",
		json:          `{"foo": "barrrrrrrrrrrrrrr"}`,
		errorExpected: true,
		maxSize:       1,
		allowUnknown:  false,
	},
}

func TestTools_ReadJSON(t *testing.T) {
	var testTool Tools

	for _, e := range jsonTests {
		// Set the max file size
		testTool.MaxJSONSize = e.maxSize

		// Set allow/disallow unknown fields
		testTool.AllowUnknownFields = e.allowUnknown

		// Declare a variable to read the decoded json into
		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		// Create a request with the body
		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Log("Error:", err)
		}

		// Create a recorder
		rr := httptest.NewRecorder()

		err = testTool.ReadJSON(rr, req, &decodedJSON)

		if e.errorExpected && err == nil {
			t.Errorf("%s: error expected, but none received", e.name)
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error not expected, but one received: %s", e.name, err.Error())
		}

		req.Body.Close()

	}
}

func TestTools_WriteJSON(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}

	headers := make(http.Header)
	headers.Add("FOO", "BAR")

	err := testTools.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("failed to write JSON: %v", err)
	}
}

func TestTools_ErrorJSON(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()
	err := testTools.ErrorJSON(rr, errors.New("some error"), http.StatusServiceUnavailable)
	if err != nil {
		t.Error(err)
	}

	var payload JSONResponse
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		t.Error("received error when decoding JSON", err)
	}

	if !payload.Error {
		t.Error("error set to false in JSON, and it should be true")
	}

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("wrong status code returned; expected 503, but got %d", rr.Code)
	}
}

// Transport specifies the mechanism by which individual HTTP requests are made.
// Instead of using the default http.Transport, we'll replace it with our own
// implementation. To implement a transport, we'll have to implement http.RoundTripper interface.
// This interface has just one method RoundTrip(*Request) (*Response, error).
// Using this approach, we don't have to spin up a HTTP server before each test and
// replace the service url with test server url.
type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// NewTestClient returns *http.Client with Transport replaced to avoid making real calls.
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestTools_PushJSONToRemote(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		// Test Request Parameters
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("ok")),
			Header:     make(http.Header),
		}
	})

	var testTools Tools

	var foo struct {
		Bar string `json:"bar"`
	}
	foo.Bar = "bar"

	_, _, err := testTools.PushJSONToRemote("http://example.com/some/path", foo, client)
	if err != nil {
		t.Error("failed to call remote url:", err)
	}

}
