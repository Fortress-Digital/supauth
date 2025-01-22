package supauth

import (
	"errors"
	"github.com/go-playground/assert/v2"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var dateTime = time.Now().UTC()

type HttpClientMock struct {
	mock.Mock
}

func (m *HttpClientMock) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestNew(t *testing.T) {
	project := "test"
	apiKey := "abc123"

	client := New(project, "abc123")

	assert.NotEqual(t, nil, client)
	assert.Equal(t, "https://test.supabase.co/auth/v1", client.BaseUrl)
	assert.Equal(t, apiKey, client.ApiKey)
	assert.Equal(t, client.HttpClient, &http.Client{
		Timeout: time.Second * 10,
	})
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
		data:        nil,
		expectedErr: nil,
		expectReq:   true,
	},
	{
		name:        "successfully creates request json data",
		data:        `{"foo": "bar"}`,
		expectedErr: nil,
		expectReq:   true,
	},
	{
		name:        "error while marshalling json",
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
}

func TestCreateRequest(t *testing.T) {
	for _, tt := range createRequestTests {
		sut := New("test", "abc123")

		if tt.url != "" {
			sut.BaseUrl = tt.url
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
		name:         "successfully sends request but receives error",
		statusCode:   http.StatusBadRequest,
		jsonRequest:  `{"foo": "bar"}`,
		jsonResponse: `{"error": "Bad request"}`,
		expectedErr:  nil,
		expectedData: &map[string]any{"error": "Bad request"},
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
		client := new(HttpClientMock)
		sut := Client{
			BaseUrl:    "http://localhost",
			HttpClient: client,
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

		client.On("Do", mock.Anything).Return(w.Result(), clientError)

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

func TestPostRequest(t *testing.T) {
	sut := New("test", "abc123")
	req, err := sut.post("test", nil)

	assert.Equal(t, err, nil)
	assert.NotEqual(t, req, nil)
	assert.Equal(t, req.Method, http.MethodPost)
}

func TestPutRequest(t *testing.T) {
	sut := New("test", "abc123")
	req, err := sut.put("test", nil)

	assert.Equal(t, err, nil)
	assert.NotEqual(t, req, nil)
	assert.Equal(t, req.Method, http.MethodPut)
}

var signupTests = []struct {
	name         string
	baseUrl      string
	jsonResponse string
	statusCode   int
	expectedData any
	expectedErr  error
}{
	{
		name:    "Successful signup response",
		baseUrl: "https://test.supabase.co/auth/v1",
		jsonResponse: `{
			"id": "1234",
			"email": "test@example.com",
        	"confirmed_at": "` + dateTime.Format(time.RFC3339Nano) + `",
        	"confirmation_sent_at": "` + dateTime.Format(time.RFC3339Nano) + `",
			"created_at": "` + dateTime.Format(time.RFC3339Nano) + `",
			"updated_at": "` + dateTime.Format(time.RFC3339Nano) + `"
		}`,
		statusCode: http.StatusOK,
		expectedData: &SignUp{
			ID:                 "1234",
			Email:              "test@example.com",
			ConfirmedAt:        dateTime,
			ConfirmationSentAt: dateTime,
			CreatedAt:          dateTime,
			UpdatedAt:          dateTime,
		},
		expectedErr: nil,
	},
	{
		name:    "Failed signup response",
		baseUrl: "https://test.supabase.co/auth/v1",
		jsonResponse: `{
			"code": "400",
			"error_code": "email_address_invalid",
        	"msg": "Invalid email address"
		}`,
		statusCode: http.StatusBadRequest,
		expectedData: &map[string]any{
			"code":       "400",
			"error_code": "email_address_invalid",
			"msg":        "Invalid email address",
		},
		expectedErr: nil,
	},
	{
		name:    "error creating request",
		baseUrl: "",
		jsonResponse: `{
			"foo": "bar",
		}`,
		statusCode:   http.StatusOK,
		expectedData: nil,
		expectedErr:  errors.New("supabase api url is empty"),
	},
	{
		name:    "Json decode error",
		baseUrl: "https://test.supabase.co/auth/v1",
		jsonResponse: `{
			"code": "400",
		}`,
		statusCode:   http.StatusBadRequest,
		expectedData: nil,
		expectedErr:  errors.New("invalid character '}' looking for beginning of object key string"),
	},
}

func TestSignUp(t *testing.T) {
	for _, tt := range signupTests {
		client := new(HttpClientMock)
		sut := Client{
			ApiKey:     "abc123",
			BaseUrl:    tt.baseUrl,
			HttpClient: client,
		}

		creds := UserCredentials{
			Email:    "test@example.com",
			Password: "cba321",
		}

		w := httptest.NewRecorder()
		w.WriteHeader(tt.statusCode)
		w.Write([]byte(tt.jsonResponse))

		res := w.Result()
		req, err := sut.post("signup", creds)
		if err == nil {
			req.Header.Set("apikey", sut.ApiKey)
		}

		client.On("Do", mock.MatchedBy(func(r *http.Request) bool {
			return strings.Contains(r.URL.RequestURI(), "signup") && r.Method == http.MethodPost
		})).Return(res, nil)

		result, err := sut.SignUp(creds)

		if err != nil {
			assert.Equal(t, err.Error(), tt.expectedErr.Error())
		} else {
			assert.Equal(t, err, nil)
			assert.Equal(t, result.Status, tt.statusCode)
			assert.Equal(t, result.Data, tt.expectedData)
		}
	}
}

var signInTests = []struct {
	name         string
	baseUrl      string
	statusCode   int
	jsonResponse string
	expectedData any
	expectedErr  error
}{
	{
		name:       "Successful sign in",
		baseUrl:    "https://test.supabase.co/auth/v1",
		statusCode: http.StatusOK,
		jsonResponse: `{
			"access_token": "abc123",
			"user": {
				"id": "1234",
				"email": "test@example.com"
			}
		}`,
		expectedData: &Authenticated{
			AccessToken: "abc123",
			User: User{
				ID:    "1234",
				Email: "test@example.com",
			},
		},
		expectedErr: nil,
	},
	{
		name:    "error creating request",
		baseUrl: "",
		jsonResponse: `{
			"foo": "bar",
		}`,
		statusCode:   http.StatusOK,
		expectedData: nil,
		expectedErr:  errors.New("supabase api url is empty"),
	},
	{
		name:    "Json decode error",
		baseUrl: "https://test.supabase.co/auth/v1",
		jsonResponse: `{
			"code": "400",
		}`,
		statusCode:   http.StatusBadRequest,
		expectedData: nil,
		expectedErr:  errors.New("invalid character '}' looking for beginning of object key string"),
	},
}

func TestSignIn(t *testing.T) {
	for _, tt := range signInTests {
		client := new(HttpClientMock)
		sut := Client{
			ApiKey:     "abc123",
			BaseUrl:    tt.baseUrl,
			HttpClient: client,
		}

		creds := UserCredentials{
			Email:    "test@example.com",
			Password: "cba321",
		}

		w := httptest.NewRecorder()
		w.WriteHeader(tt.statusCode)
		w.Write([]byte(tt.jsonResponse))

		res := w.Result()
		req, err := sut.post("signup", creds)
		if err == nil {
			req.Header.Set("apikey", sut.ApiKey)
		}

		client.On("Do", mock.MatchedBy(func(r *http.Request) bool {
			return strings.Contains(r.URL.RequestURI(), "token?grant_type=password") && r.Method == http.MethodPost
		})).Return(res, nil)

		result, err := sut.SignIn(creds)

		if err != nil {
			assert.Equal(t, err.Error(), tt.expectedErr.Error())
		} else {
			assert.Equal(t, err, nil)
			assert.Equal(t, result.Status, tt.statusCode)
			assert.Equal(t, result.Data, tt.expectedData)
		}
	}
}

var signOutTests = []struct {
	name         string
	baseUrl      string
	statusCode   int
	token        string
	jsonResponse string
	expectedErr  error
}{
	{
		name:         "Successful sign out",
		baseUrl:      "https://test.supabase.co/auth/v1",
		statusCode:   http.StatusNoContent,
		token:        "abc123",
		jsonResponse: `{}`,
		expectedErr:  nil,
	},
	{
		name:         "error creating request",
		baseUrl:      "",
		statusCode:   http.StatusNoContent,
		token:        "abc123",
		jsonResponse: `{}`,
		expectedErr:  errors.New("supabase api url is empty"),
	},
	{
		name:       "Json decode error",
		baseUrl:    "https://test.supabase.co/auth/v1",
		statusCode: http.StatusBadRequest,
		token:      "abc123",
		jsonResponse: `{
			"code": "400",
		}`,
		expectedErr: errors.New("invalid character '}' looking for beginning of object key string"),
	},
}

func TestSignOut(t *testing.T) {
	for _, tt := range signOutTests {
		client := new(HttpClientMock)
		sut := Client{
			ApiKey:     "abc123",
			BaseUrl:    tt.baseUrl,
			HttpClient: client,
		}

		w := httptest.NewRecorder()
		w.WriteHeader(tt.statusCode)
		w.Write([]byte(tt.jsonResponse))

		res := w.Result()
		client.On("Do", mock.MatchedBy(func(r *http.Request) bool {
			return strings.Contains(r.URL.RequestURI(), "logout") && r.Method == http.MethodPost
		})).Return(res, nil)

		result, err := sut.SignOut(tt.token)

		if err != nil {
			assert.Equal(t, err.Error(), tt.expectedErr.Error())
		} else {
			assert.Equal(t, err, nil)
			assert.Equal(t, result.Status, tt.statusCode)
			assert.Equal(t, result.Data, nil)
		}
	}
}

var refreshTokenTests = []struct {
	name         string
	baseUrl      string
	statusCode   int
	token        string
	jsonResponse string
	expectedData any
	expectedErr  error
}{
	{
		name:       "Successful token refresh",
		baseUrl:    "https://test.supabase.co/auth/v1",
		statusCode: http.StatusOK,
		token:      "abc123",
		jsonResponse: `{
			"access_token": "abc123",
			"user": {
				"id": "1234",
				"email": "test@example.com"
			}
		}`,
		expectedData: &Authenticated{
			AccessToken: "abc123",
			User: User{
				ID:    "1234",
				Email: "test@example.com",
			},
		},
		expectedErr: nil,
	},
	{
		name:         "error creating request",
		baseUrl:      "",
		statusCode:   http.StatusOK,
		token:        "abc123",
		jsonResponse: `{}`,
		expectedErr:  errors.New("supabase api url is empty"),
	},
	{
		name:       "Json decode error",
		baseUrl:    "https://test.supabase.co/auth/v1",
		statusCode: http.StatusBadRequest,
		token:      "abc123",
		jsonResponse: `{
			"code": "400",
		}`,
		expectedErr: errors.New("invalid character '}' looking for beginning of object key string"),
	},
}

func TestRefreshToken(t *testing.T) {
	for _, tt := range refreshTokenTests {
		client := new(HttpClientMock)
		sut := Client{
			ApiKey:     "abc123",
			BaseUrl:    tt.baseUrl,
			HttpClient: client,
		}

		w := httptest.NewRecorder()
		w.WriteHeader(tt.statusCode)
		w.Write([]byte(tt.jsonResponse))

		res := w.Result()
		client.On("Do", mock.MatchedBy(func(r *http.Request) bool {
			return strings.Contains(r.URL.RequestURI(), "token?grant_type=refresh_token") && r.Method == http.MethodPost
		})).Return(res, nil)

		result, err := sut.RefreshToken(tt.token)
		if err != nil {
			assert.Equal(t, err.Error(), tt.expectedErr.Error())
		} else {
			assert.Equal(t, err, nil)
			assert.Equal(t, result.Status, tt.statusCode)
			assert.Equal(t, result.Data, tt.expectedData)
		}
	}
}

var forgottenPasswordTests = []struct {
	name         string
	baseUrl      string
	statusCode   int
	jsonResponse string
	expectedErr  error
}{
	{
		name:         "Successful forgotten password request",
		baseUrl:      "https://test.supabase.co/auth/v1",
		statusCode:   http.StatusOK,
		jsonResponse: `{}`,
		expectedErr:  nil,
	},
	{
		name:         "error creating request",
		baseUrl:      "",
		statusCode:   http.StatusOK,
		jsonResponse: `{}`,
		expectedErr:  errors.New("supabase api url is empty"),
	},
	{
		name:       "Json decode error",
		baseUrl:    "https://test.supabase.co/auth/v1",
		statusCode: http.StatusBadRequest,
		jsonResponse: `{
			"code": "400",
		}`,
		expectedErr: errors.New("invalid character '}' looking for beginning of object key string"),
	},
}

func TestForgottenPassword(t *testing.T) {
	for _, tt := range forgottenPasswordTests {
		client := new(HttpClientMock)
		sut := Client{
			ApiKey:     "abc123",
			BaseUrl:    tt.baseUrl,
			HttpClient: client,
		}

		w := httptest.NewRecorder()
		w.WriteHeader(tt.statusCode)
		w.Write([]byte(tt.jsonResponse))

		res := w.Result()
		client.On("Do", mock.MatchedBy(func(r *http.Request) bool {
			return strings.Contains(r.URL.RequestURI(), "recover") && r.Method == http.MethodPost
		})).Return(res, nil)

		result, err := sut.ForgottenPassword("test@example.com")
		if err != nil {
			assert.Equal(t, err.Error(), tt.expectedErr.Error())
		} else {
			assert.Equal(t, err, nil)
			assert.Equal(t, result.Status, tt.statusCode)
			assert.Equal(t, result.Data, nil)
		}
	}
}

var resetPasswordTests = []struct {
	name         string
	baseUrl      string
	statusCode   int
	jsonResponse string
	expectedErr  error
}{
	{
		name:         "Successful reset password request",
		baseUrl:      "https://test.supabase.co/auth/v1",
		statusCode:   http.StatusOK,
		jsonResponse: `{}`,
		expectedErr:  nil,
	},
	{
		name:         "error creating request",
		baseUrl:      "",
		statusCode:   http.StatusOK,
		jsonResponse: `{}`,
		expectedErr:  errors.New("supabase api url is empty"),
	},
	{
		name:       "Json decode error",
		baseUrl:    "https://test.supabase.co/auth/v1",
		statusCode: http.StatusBadRequest,
		jsonResponse: `{
			"code": "400",
		}`,
		expectedErr: errors.New("invalid character '}' looking for beginning of object key string"),
	},
}

func TestResetPassword(t *testing.T) {
	for _, tt := range resetPasswordTests {
		client := new(HttpClientMock)
		sut := Client{
			ApiKey:     "abc123",
			BaseUrl:    tt.baseUrl,
			HttpClient: client,
		}

		w := httptest.NewRecorder()
		w.WriteHeader(tt.statusCode)
		w.Write([]byte(tt.jsonResponse))
		res := w.Result()

		client.On("Do", mock.MatchedBy(func(r *http.Request) bool {
			return strings.Contains(r.URL.RequestURI(), "user?type=recovery") && r.Method == http.MethodPut
		})).Return(res, nil)

		result, err := sut.ResetPassword("abc123", "Password1")
		if err != nil {
			assert.Equal(t, err.Error(), tt.expectedErr.Error())
		} else {
			assert.Equal(t, err, nil)
			assert.Equal(t, result.Status, tt.statusCode)
			assert.Equal(t, result.Data, nil)
		}
	}
}
