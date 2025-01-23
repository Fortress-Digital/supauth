package supauth

import (
	"fmt"
	"net/http"
	"time"
)

type UserCredentials struct {
	Email    string
	Password string
}

type SignUp struct {
	ID                 string    `json:"id"`
	Email              string    `json:"email"`
	ConfirmedAt        time.Time `json:"confirmed_at"`
	ConfirmationSentAt time.Time `json:"confirmation_sent_at"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type User struct {
	ID                 string                    `json:"id"`
	Aud                string                    `json:"aud"`
	Role               string                    `json:"role"`
	Email              string                    `json:"email"`
	InvitedAt          time.Time                 `json:"invited_at"`
	ConfirmedAt        time.Time                 `json:"confirmed_at"`
	ConfirmationSentAt time.Time                 `json:"confirmation_sent_at"`
	AppMetadata        struct{ provider string } `json:"app_metadata"`
	UserMetadata       map[string]interface{}    `json:"user_metadata"`
	CreatedAt          time.Time                 `json:"created_at"`
	UpdatedAt          time.Time                 `json:"updated_at"`
}

type Authenticated struct {
	AccessToken          string `json:"access_token"`
	TokenType            string `json:"token_type"`
	ExpiresIn            int    `json:"expires_in"`
	RefreshToken         string `json:"refresh_token"`
	User                 User   `json:"user"`
	ProviderToken        string `json:"provider_token"`
	ProviderRefreshToken string `json:"provider_refresh_token"`
}

type AuthInterface interface {
	SignUp(credentials UserCredentials) (*AuthResponse, error)
	SignIn(credentials UserCredentials) (*AuthResponse, error)
	SignOut(token string) (*AuthResponse, error)
	RefreshToken(refreshToken string) (*AuthResponse, error)
	ForgottenPassword(email string) (*AuthResponse, error)
	ResetPassword(token, password string) (*AuthResponse, error)
}

type Auth struct {
	client clientInterface
}

func NewAuth(projectId string, apiKey string) AuthInterface {
	client := newClient(projectId, apiKey).(*client)

	return &Auth{
		client: client,
	}
}

func (a *Auth) SignUp(credentials UserCredentials) (*AuthResponse, error) {
	successResponse := &SignUp{}

	return a.client.createAndSendRequest(http.MethodPost, "signup", credentials, successResponse)
}

func (a *Auth) SignIn(credentials UserCredentials) (*AuthResponse, error) {
	successResponse := &Authenticated{}

	return a.client.createAndSendRequest(http.MethodPost, "token?grant_type=password", credentials, successResponse)
}

func (a *Auth) SignOut(token string) (*AuthResponse, error) {
	req, err := a.client.createRequest(http.MethodPost, "logout", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	authResponse, err := a.client.sendRequest(req, nil)
	if err != nil {
		return nil, err
	}

	return authResponse, nil
}

func (a *Auth) RefreshToken(refreshToken string) (*AuthResponse, error) {
	reqBody := map[string]string{"refresh_token": refreshToken}

	successResponse := &Authenticated{}

	return a.client.createAndSendRequest(http.MethodPost, "token?grant_type=refresh_token", reqBody, successResponse)
}

func (a *Auth) ForgottenPassword(email string) (*AuthResponse, error) {
	reqBody := map[string]string{"email": email}

	return a.client.createAndSendRequest(http.MethodPost, "recover", reqBody, nil)
}

func (a *Auth) ResetPassword(token, password string) (*AuthResponse, error) {
	reqBody := map[string]string{"password": password}
	req, err := a.client.createRequest(http.MethodPut, "user?type=recovery", reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	authResponse, err := a.client.sendRequest(req, nil)
	if err != nil {
		return nil, err
	}

	return authResponse, nil
}
