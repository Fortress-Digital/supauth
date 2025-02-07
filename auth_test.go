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

var dateTime = time.Now().UTC()

type clientMock struct {
	mock.Mock
}

func (c *clientMock) createAndSendRequest(method, endpoint string, data, successValue any) (*AuthResponse, error) {
	args := c.Called(method, endpoint, data, successValue)
	return args.Get(0).(*AuthResponse), args.Error(1)
}

func (c *clientMock) createRequest(method, endpoint string, data any) (*http.Request, error) {
	args := c.Called(method, endpoint, data)
	return args.Get(0).(*http.Request), args.Error(1)
}

func (c *clientMock) sendRequest(req *http.Request, successValue any) (*AuthResponse, error) {
	args := c.Called(req, successValue)
	return args.Get(0).(*AuthResponse), args.Error(1)
}

func TestNewAuth(t *testing.T) {
	project := "test"
	apiKey := "abc123"

	auth := NewAuth(project, apiKey)

	assert.NotEqual(t, nil, auth.client)
}

var signUpTests = []struct {
	name           string
	authResponse   *AuthResponse
	sendRequestErr error
	resultErr      error
}{
	{
		name: "successful signup",
		authResponse: &AuthResponse{
			Status: http.StatusOK,
			Data: User{
				ID:    "abc123",
				Email: "test@example.com",
			},
		},
		sendRequestErr: nil,
		resultErr:      nil,
	},
	{
		name:           "failed sign up with send request error",
		authResponse:   nil,
		sendRequestErr: errors.New("send request error"),
		resultErr:      errors.New("send request error"),
	},
}

func TestAuth_SignUp(t *testing.T) {
	for _, tt := range signUpTests {
		client := new(clientMock)
		sut := &Auth{
			client: client,
		}
		creds := UserCredentials{
			Email:    "test@example.com",
			Password: "password",
		}

		client.On("createAndSendRequest", http.MethodPost, "signup", creds, &SignUp{}).
			Return(tt.authResponse, tt.sendRequestErr)

		result, err := sut.SignUp(creds)

		if err != nil {
			assert.Equal(t, err.Error(), tt.resultErr.Error())
			assert.Equal(t, result, tt.authResponse)
		} else {
			assert.Equal(t, err, nil)
			assert.Equal(t, result, tt.authResponse)
		}
	}
}

var signInTests = []struct {
	name           string
	authResponse   *AuthResponse
	sendRequestErr error
	resultErr      error
}{
	{
		name: "successful sign in",
		authResponse: &AuthResponse{
			Status: http.StatusOK,
			Data: Authenticated{
				AccessToken: "cba321",
				User: User{
					ID:    "abc123",
					Email: "test@example.com",
				},
			},
		},
		sendRequestErr: nil,
		resultErr:      nil,
	},
	{
		name:           "failed sign in with send request error",
		authResponse:   nil,
		sendRequestErr: errors.New("send request error"),
		resultErr:      errors.New("send request error"),
	},
}

func TestAuth_SignIn(t *testing.T) {
	for _, tt := range signInTests {
		client := new(clientMock)
		sut := &Auth{
			client: client,
		}
		creds := UserCredentials{
			Email:    "test@example.com",
			Password: "password",
		}

		client.On("createAndSendRequest", http.MethodPost, "token?grant_type=password", creds, &Authenticated{}).
			Return(tt.authResponse, tt.sendRequestErr)

		result, err := sut.SignIn(creds)

		if err != nil {
			assert.Equal(t, err.Error(), tt.resultErr.Error())
			assert.Equal(t, result, tt.authResponse)
		} else {
			assert.Equal(t, err, nil)
			assert.Equal(t, result, tt.authResponse)
		}
	}
}

var signOutTests = []struct {
	name           string
	createReqErr   error
	authResponse   *AuthResponse
	sendRequestErr error
	resultErr      error
}{
	{
		name:           "successful sign out",
		createReqErr:   nil,
		authResponse:   &AuthResponse{Status: http.StatusNoContent},
		sendRequestErr: nil,
		resultErr:      nil,
	},
	{
		name:           "error on sign out create request",
		createReqErr:   errors.New("create request error"),
		authResponse:   nil,
		sendRequestErr: nil,
		resultErr:      errors.New("create request error"),
	},
	{
		name:           "error on sign out send request",
		createReqErr:   nil,
		authResponse:   nil,
		sendRequestErr: errors.New("send request error"),
		resultErr:      errors.New("send request error"),
	},
}

func TestAuth_SignOut(t *testing.T) {
	for _, tt := range signOutTests {
		client := new(clientMock)
		sut := &Auth{
			client: client,
		}

		req := httptest.NewRequest(http.MethodPost, "/logout", nil)

		client.On("createRequest", http.MethodPost, "logout", nil).Return(req, tt.createReqErr)
		client.On("sendRequest", req, nil).Return(tt.authResponse, tt.sendRequestErr)

		result, err := sut.SignOut("abc123")

		if err != nil {
			assert.Equal(t, err.Error(), tt.resultErr.Error())
			assert.Equal(t, result, tt.authResponse)
		} else {
			assert.Equal(t, err, nil)
			assert.Equal(t, result, tt.authResponse)
		}
	}
}

var refreshTokenTests = []struct {
	name           string
	authResponse   *AuthResponse
	sendRequestErr error
	resultErr      error
}{
	{
		name: "successful reset",
		authResponse: &AuthResponse{
			Status: http.StatusOK,
			Data: Authenticated{
				AccessToken: "cba321",
				User: User{
					ID:    "abc123",
					Email: "test@example.com",
				},
			},
		},
		sendRequestErr: nil,
		resultErr:      nil,
	},
	{
		name:           "failed reset with send request error",
		authResponse:   nil,
		sendRequestErr: errors.New("send request error"),
		resultErr:      errors.New("send request error"),
	},
}

func TestAuth_RefreshToken(t *testing.T) {
	for _, tt := range refreshTokenTests {
		client := new(clientMock)
		sut := &Auth{
			client: client,
		}

		refreshToken := "cba987"
		reqBody := map[string]string{"refresh_token": refreshToken}

		client.On("createAndSendRequest", http.MethodPost, "token?grant_type=refresh_token", reqBody, &Authenticated{}).
			Return(tt.authResponse, tt.sendRequestErr)

		result, err := sut.RefreshToken(refreshToken)

		if err != nil {
			assert.Equal(t, err.Error(), tt.resultErr.Error())
			assert.Equal(t, result, tt.authResponse)
		} else {
			assert.Equal(t, err, nil)
			assert.Equal(t, result, tt.authResponse)
		}
	}
}

var forgottenPasswordTests = []struct {
	name           string
	authResponse   *AuthResponse
	sendRequestErr error
	resultErr      error
}{
	{
		name: "successful forgotten password",
		authResponse: &AuthResponse{
			Status: http.StatusNoContent,
		},
		sendRequestErr: nil,
		resultErr:      nil,
	},
	{
		name:           "failed forgotten password with send request error",
		authResponse:   nil,
		sendRequestErr: errors.New("send request error"),
		resultErr:      errors.New("send request error"),
	},
}

func TestAuth_ForgottenPassword(t *testing.T) {
	for _, tt := range forgottenPasswordTests {
		client := new(clientMock)
		sut := &Auth{
			client: client,
		}

		email := "test@example.com"
		reqBody := map[string]string{"email": email}

		client.On("createAndSendRequest", http.MethodPost, "recover", reqBody, nil).
			Return(tt.authResponse, tt.sendRequestErr)

		result, err := sut.ForgottenPassword(email)

		if err != nil {
			assert.Equal(t, err.Error(), tt.resultErr.Error())
			assert.Equal(t, result, tt.authResponse)
		} else {
			assert.Equal(t, err, nil)
			assert.Equal(t, result, tt.authResponse)
		}
	}
}

var resetPasswordTests = []struct {
	name           string
	createReqErr   error
	authResponse   *AuthResponse
	sendRequestErr error
	resultErr      error
}{
	{
		name:           "successful password reset",
		createReqErr:   nil,
		authResponse:   &AuthResponse{Status: http.StatusNoContent},
		sendRequestErr: nil,
		resultErr:      nil,
	},
	{
		name:           "error on reset password create request",
		createReqErr:   errors.New("create request error"),
		authResponse:   nil,
		sendRequestErr: nil,
		resultErr:      errors.New("create request error"),
	},
	{
		name:           "error on password reset",
		createReqErr:   nil,
		authResponse:   nil,
		sendRequestErr: errors.New("send request error"),
		resultErr:      errors.New("send request error"),
	},
}

func TestAuth_ResetPassword(t *testing.T) {
	for _, tt := range resetPasswordTests {
		client := new(clientMock)
		sut := &Auth{
			client: client,
		}
		password := "newPassword"
		reqBody := map[string]string{"password": password}

		req := httptest.NewRequest(http.MethodPut, "/user?type=recovery", nil)

		client.On("createRequest", http.MethodPut, "user?type=recovery", reqBody).Return(req, tt.createReqErr)
		client.On("sendRequest", req, nil).Return(tt.authResponse, tt.sendRequestErr)

		result, err := sut.ResetPassword("abc123", password)

		if err != nil {
			assert.Equal(t, err.Error(), tt.resultErr.Error())
			assert.Equal(t, result, tt.authResponse)
		} else {
			assert.Equal(t, err, nil)
			assert.Equal(t, result, tt.authResponse)
		}
	}
}
