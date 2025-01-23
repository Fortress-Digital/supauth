package supauth

import (
	"errors"
	"github.com/go-playground/assert/v2"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type HttpClientMock struct {
	mock.Mock
}

func (m *HttpClientMock) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestNewClient(t *testing.T) {
	project := "test"
	apiKey := "abc123"

	client := newClient(project, "abc123").(*client)

	assert.NotEqual(t, nil, client)
	assert.Equal(t, "https://test.supabase.co/auth/v1", client.BaseUrl)
	assert.Equal(t, apiKey, client.ApiKey)
	assert.Equal(t, client.HttpClient, &http.Client{
		Timeout: time.Second * 10,
	})
}

var varCreateAndSendRequestTests = []struct {
	name         string
	url          string
	statusCode   int
	jsonResponse string
	expectedData any
	expectedErr  error
}{
	{
		name:         "successfully creates and sends request",
		url:          "https://test.supabase.co/auth/v1",
		statusCode:   http.StatusOK,
		jsonResponse: `{"foo": "bar"}`,
		expectedData: &map[string]any{"foo": "bar"},
		expectedErr:  nil,
	},
	{
		name:         "fails to create and sends request",
		url:          "",
		statusCode:   http.StatusBadRequest,
		jsonResponse: `{"foo": "bar"}`,
		expectedData: nil,
		expectedErr:  errors.New("supabase api url is empty"),
	},
}

func TestCreateAndSendRequest(t *testing.T) {
	for _, tt := range varCreateAndSendRequestTests {
		httpClient := new(HttpClientMock)
		sut := client{
			BaseUrl:    tt.url,
			HttpClient: httpClient,
		}

		var successValue = map[string]any{}

		w := httptest.NewRecorder()
		w.WriteHeader(tt.statusCode)
		w.Write([]byte(tt.jsonResponse))

		httpClient.On("Do", mock.Anything).Return(w.Result(), nil)

		result, err := sut.createAndSendRequest(http.MethodPost, "test", nil, successValue)

		if err != nil {
			assert.Equal(t, err.Error(), tt.expectedErr.Error())
			assert.Equal(t, result, tt.expectedData)
		} else {
			assert.Equal(t, err, nil)
			assert.Equal(t, result.Status, tt.statusCode)
			assert.Equal(t, result.Data, tt.expectedData)
		}
	}
}

var createRequestTests = []struct {
	name        string
	url         string
	data        any
	expectedErr error
	expectReq   bool
}{
	{
		name:        "successfully creates request nil data",
		url:         "https://test.supabase.co/auth/v1",
		data:        nil,
		expectedErr: nil,
		expectReq:   true,
	},
	{
		name:        "successfully creates request json data",
		url:         "https://test.supabase.co/auth/v1",
		data:        `{"foo": "bar"}`,
		expectedErr: nil,
		expectReq:   true,
	},
	{
		name:        "error while marshalling json",
		url:         "https://test.supabase.co/auth/v1",
		data:        make(chan int),
		expectedErr: errors.New("json: unsupported type: chan int"),
		expectReq:   false,
	},
	{
		name:        "invalid url",
		url:         "https://test.supabase.co{auth/v1",
		data:        nil,
		expectedErr: errors.New("parse \"https://test.supabase.co{auth/v1/test\": invalid character \"{\" in host name"),
		expectReq:   false,
	},
	{
		name:        "error no base url",
		url:         "",
		data:        nil,
		expectedErr: errors.New("supabase api url is empty"),
		expectReq:   false,
	},
}

func TestCreateRequest(t *testing.T) {
	for _, tt := range createRequestTests {
		httpClient := new(HttpClientMock)
		sut := client{
			BaseUrl:    tt.url,
			HttpClient: httpClient,
		}

		req, err := sut.createRequest(http.MethodGet, "test", tt.data)

		if tt.expectReq {
			assert.Equal(t, err, nil)
			assert.NotEqual(t, req, nil)
			assert.Equal(t, req.Method, http.MethodGet)
			assert.Equal(t, req.URL.String(), "https://test.supabase.co/auth/v1/test")
			assert.Equal(t, req.Header.Get("Content-Type"), "application/json")
			assert.Equal(t, req.Header.Get("Accept"), "application/json")
		} else {
			assert.Equal(t, err.Error(), tt.expectedErr.Error())
			assert.Equal(t, req, nil)
		}
	}
}

var sendRequestsTests = []struct {
	name         string
	statusCode   int
	jsonRequest  string
	jsonResponse string
	expectedErr  error
	expectedData any
}{
	{
		name:         "successfully sends request",
		statusCode:   http.StatusOK,
		jsonRequest:  `{"foo": "bar"}`,
		jsonResponse: `{"foo": "bar"}`,
		expectedErr:  nil,
		expectedData: &map[string]any{"foo": "bar"},
	},
	{
		name:         "successfully sends request with no content response",
		statusCode:   http.StatusNoContent,
		jsonRequest:  `{"foo": "bar"}`,
		jsonResponse: "",
		expectedErr:  nil,
		expectedData: nil,
	},
	{
		name:        "successfully sends request but receives error",
		statusCode:  http.StatusBadRequest,
		jsonRequest: `{"foo": "bar"}`,
		jsonResponse: `{
			"code": 400,
			"error_code": "used_foo_bar",
			"msg": "Bad Request"
		}`,
		expectedErr: nil,
		expectedData: &ErrorResponse{
			Status:    400,
			ErrorCode: "used_foo_bar",
			Message:   "Bad Request",
		},
	},
	{
		name:         "simulates a client error",
		statusCode:   http.StatusInternalServerError,
		jsonRequest:  `{"foo": "bar"}`,
		jsonResponse: `{"error": "Bad request"}`,
		expectedErr:  errors.New("http client error"),
		expectedData: nil,
	},
	{
		name:         "successfully sends request but receives a bad request response with invalid json",
		statusCode:   http.StatusBadRequest,
		jsonRequest:  `{"foo": "bar"}`,
		jsonResponse: `!`,
		expectedErr:  errors.New("invalid character '!' looking for beginning of value"),
		expectedData: nil,
	},
	{
		name:         "successfully sends request and gets ok response, but with invalid json",
		statusCode:   http.StatusOK,
		jsonRequest:  `{"foo": "bar"}`,
		jsonResponse: `!`,
		expectedErr:  errors.New("invalid character '!' looking for beginning of value"),
		expectedData: nil,
	},
}

func TestSendRequest(t *testing.T) {
	for _, tt := range sendRequestsTests {
		httpClient := new(HttpClientMock)
		sut := client{
			BaseUrl:    "http://localhost",
			HttpClient: httpClient,
		}

		req, _ := sut.createRequest(http.MethodGet, "test", tt.jsonRequest)

		var successValue = map[string]any{}

		w := httptest.NewRecorder()
		var clientError error

		if tt.statusCode == http.StatusInternalServerError {
			clientError = errors.New("http client error")
		} else {
			w.WriteHeader(tt.statusCode)
			w.Write([]byte(tt.jsonResponse))
		}

		httpClient.On("Do", mock.Anything).Return(w.Result(), clientError)

		response, err := sut.sendRequest(req, &successValue)

		if err != nil {
			assert.Equal(t, err.Error(), tt.expectedErr.Error())
		} else {
			assert.Equal(t, err, nil)
			assert.Equal(t, response.Status, tt.statusCode)
			assert.Equal(t, response.Data, tt.expectedData)
		}
	}
}
