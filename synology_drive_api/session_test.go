package synology_drive_api

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpGet(t *testing.T) {
	s, err := NewSynologySession(getNasUser(), getNasPass(), getNasUrl())
	url := s.buildUrl("auth.cgi", map[string]string{
		"api":     "SYNO.API.Auth",
		"method":  "login",
		"version": "3",
		"account": getNasUser(),
		"passwd":  getNasPass(),
		"session": "SynologyDrive",
		"format":  "cookie",
	})
	require.NoError(t, err)

	// Verify the structure of the URL but exclude the password
	expectedUrl := getNasUrl() + "/webapi/auth.cgi"
	assert.Contains(t, url.String(), expectedUrl)
	assert.Contains(t, url.String(), "account="+getNasUser())
	assert.Contains(t, url.String(), "api=SYNO.API.Auth")
	assert.Contains(t, url.String(), "format=cookie")
	assert.Contains(t, url.String(), "method=login")
	assert.Contains(t, url.String(), "session=SynologyDrive")
	assert.Contains(t, url.String(), "version=3")
	// Password is included in the URL but not verified in the test
}

func TestNewSynologySession(t *testing.T) {
	// Create a session with a valid URL
	session, err := NewSynologySession("testuser", "testpass", "https://test.synology.com")
	require.NoError(t, err, "NewSynologySession should not fail with valid URL")
	assert.Equal(t, "testuser", session.username, "Username should match")
	assert.Equal(t, "testpass", session.password, "Password should match")
	assert.Equal(t, "test.synology.com", session.hostname, "Hostname should match")
	assert.Equal(t, "https", session.scheme, "Scheme should match")

	// Create a session with an invalid URL
	_, err = NewSynologySession("testuser", "testpass", ":invalid-url")
	assert.Error(t, err, "NewSynologySession should fail with invalid URL")
	assert.IsType(t, InvalidUrlError(""), err, "Error should be of type InvalidUrlError")
}

func TestSessionExpired(t *testing.T) {
	session, err := NewSynologySession("testuser", "testpass", "https://test.synology.com")
	require.NoError(t, err, "Failed to create test session")

	// Initially the session has no sid and is considered expired
	assert.True(t, session.sessionExpired(), "New session should be expired")

	// When sid is set, the session is not expired
	session.sid = "test-sid"
	assert.False(t, session.sessionExpired(), "Session with sid should not be expired")
}

func TestBuildUrl(t *testing.T) {
	session, err := NewSynologySession("testuser", "testpass", "https://test.synology.com")
	require.NoError(t, err, "Failed to create test session")

	// URL without parameters
	url := session.buildUrl("test.cgi", nil)
	expected := "https://test.synology.com/webapi/test.cgi"
	assert.Equal(t, expected, url.String(), "URL without parameters should match expected format")

	// URL with parameters
	params := map[string]string{
		"param1": "value1",
		"param2": "value2",
	}
	url = session.buildUrl("test.cgi", params)
	assert.Equal(t, "value1", url.Query().Get("param1"), "URL should contain correct param1 value")
	assert.Equal(t, "value2", url.Query().Get("param2"), "URL should contain correct param2 value")
}

// testSleeper is a test implementation of the sleeper interface
// testSleeper is a mock implementation of the sleeper interface for testing
type testSleeper struct {
	sleepCalls []time.Duration
}

// Sleep records the sleep duration
func (t *testSleeper) Sleep(d time.Duration) {
	t.sleepCalls = append(t.sleepCalls, d)
}

// testServer is a test HTTP server that can be used to test the Synology API client
type testServer struct {
	server    *httptest.Server
	requests  []*http.Request
	responses []response
	mu        sync.Mutex
	callCount int
}

type response struct {
	statusCode int
	body       string
}

// newTestServer creates a new test server
func newTestServer() *testServer {
	ts := &testServer{
		requests:  make([]*http.Request, 0),
		responses: make([]response, 0),
	}
	ts.server = httptest.NewServer(http.HandlerFunc(ts.handler))
	ts.server.URL = ts.server.URL + "/webapi/"
	return ts
}

// handler handles incoming HTTP requests
func (ts *testServer) handler(w http.ResponseWriter, r *http.Request) {
	ts.mu.Lock()

	// Clone the request to avoid data races
	req := *r
	req.Header = make(http.Header)
	for k, v := range r.Header {
		req.Header[k] = v
	}
	ts.requests = append(ts.requests, &req)

	// If we have responses configured, use them
	if ts.callCount < len(ts.responses) {
		resp := ts.responses[ts.callCount]
		ts.callCount++
		ts.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.statusCode)
		w.Write([]byte(resp.body))
		return
	}

	ts.mu.Unlock()

	// Default to 500 error if no response is configured
	http.Error(w, "no response configured", http.StatusInternalServerError)
}

// addResponse adds a response to the test server
func (ts *testServer) addResponse(statusCode int, body string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.responses = append(ts.responses, response{
		statusCode: statusCode,
		body:       body,
	})
}

// close closes the test server
func (ts *testServer) close() {
	if ts.server != nil {
		ts.server.Close()
	}
}

func TestHTTPGetJSONWithRetry_SuccessOnFirstTry(t *testing.T) {
	// Setup test server
	ts := newTestServer()
	defer ts.close()

	// Add a successful response
	ts.addResponse(http.StatusOK, `{"success": true}`)

	// Create a test session with a custom HTTP client that uses our test server
	session, err := NewSynologySession("test", "test", ts.server.URL)
	require.NoError(t, err)

	// Replace the HTTP client with one that uses our test server
	session.http_client = *ts.server.Client()

	// Create a test sleeper
	sleeper := &testSleeper{}

	// Call the method under test
	resp, err := session.httpGetJSONWithRetry("test.cgi", map[string]string{"key": "value"}, 3, time.Second, sleeper)

	// Verify results
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, sleeper.sleepCalls, "should not sleep on first success")
	assert.Len(t, ts.requests, 1, "should make one request")
}

func TestHTTPGetJSONWithRetry_SuccessOnRetry(t *testing.T) {
	var requestCount int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		t.Logf("Request path: %s", r.URL.Path)
		if requestCount == 1 {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer ts.Close()

	// Use a base URL that matches what the production code expects
	baseURL := ts.URL + "/webapi"
	session, err := NewSynologySession("test", "test", baseURL)
	require.NoError(t, err)
	session.http_client = *ts.Client()
	sleeper := &testSleeper{}

	resp, err := session.httpGetJSONWithRetry("test.cgi", map[string]string{"key": "value"}, 1, time.Second, sleeper)

	t.Logf("Handler was called %d times", requestCount)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Len(t, sleeper.sleepCalls, 1, "should sleep once before retry")
	assert.Equal(t, time.Second, sleeper.sleepCalls[0], "should sleep for the specified duration")
	assert.Equal(t, 2, requestCount, "handler should be called twice (1 fail, 1 success)")
}

func TestHTTPGetJSONWithRetry_AllAttemptsFail(t *testing.T) {
	// Setup test server
	ts := newTestServer()
	defer ts.close()

	// Add 4 failure responses
	for i := 0; i < 4; i++ {
		ts.addResponse(http.StatusInternalServerError, `{"error": {"code": 500, "message": "temporary error"}}`)
	}

	// Create a test session with a custom HTTP client that uses our test server
	session, err := NewSynologySession("test", "test", ts.server.URL)
	require.NoError(t, err)

	// Replace the HTTP client with one that uses our test server
	session.http_client = *ts.server.Client()

	// Create a test sleeper
	sleeper := &testSleeper{}

	// Call the method under test with maxRetries=3 (total 4 attempts)
	resp, err := session.httpGetJSONWithRetry("test.cgi", map[string]string{"key": "value"}, 3, time.Second, sleeper)

	// Verify results
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "after 4 attempts")
	assert.Len(t, sleeper.sleepCalls, 3, "should sleep before each retry")
}
