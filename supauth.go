package supauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const authEndpoint = "auth/v1"

type ErrorResponse = map[string]any

type UserCredentials struct {
	Email    string
	Password string
}

type AuthResponse struct {
	Status int `json:"status"`
	Data   any `json:"data"`
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

type HttpClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

type Auth interface {
	SignUp(credentials UserCredentials) (*AuthResponse, error)
	SignIn(credentials UserCredentials) (*AuthResponse, error)
	SignOut(token string) (*AuthResponse, error)
	RefreshToken(refreshToken string) (*AuthResponse, error)
	ForgottenPassword(email string) (*AuthResponse, error)
	ResetPassword(token, password string) (*AuthResponse, error)
}

type Client struct {
	BaseUrl    string
	ApiKey     string
	HttpClient HttpClientInterface
}

func New(projectId string, apiKey string) Auth {
	baseUrl := fmt.Sprintf("https://%s.supabase.co/%s", projectId, authEndpoint)

	return &Client{
		BaseUrl: baseUrl,
		ApiKey:  apiKey,
		HttpClient: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

func (c *Client) SignUp(credentials UserCredentials) (*AuthResponse, error) {
	req, err := c.post("signup", credentials)
	if err != nil {
		return nil, err
	}

	successResponse := SignUp{}

	authResponse, err := c.sendRequest(req, &successResponse)
	if err != nil {
		return nil, err
	}

	return authResponse, nil
}

func (c *Client) SignIn(credentials UserCredentials) (*AuthResponse, error) {
	req, err := c.post("token?grant_type=password", credentials)
	if err != nil {
		return nil, err
	}

	successResponse := Authenticated{}

	authResponse, err := c.sendRequest(req, &successResponse)
	if err != nil {
		return nil, err
	}

	return authResponse, nil
}

func (c *Client) SignOut(token string) (*AuthResponse, error) {
	req, err := c.post("logout", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	authResponse, err := c.sendRequest(req, nil)
	if err != nil {
		return nil, err
	}

	return authResponse, nil
}

func (c *Client) RefreshToken(refreshToken string) (*AuthResponse, error) {
	reqBody := map[string]string{"refresh_token": refreshToken}
	req, err := c.post("token?grant_type=refresh_token", reqBody)
	if err != nil {
		return nil, err
	}

	successResponse := Authenticated{}
	authResponse, err := c.sendRequest(req, &successResponse)
	if err != nil {
		return nil, err
	}

	return authResponse, nil
}

func (c *Client) ForgottenPassword(email string) (*AuthResponse, error) {
	reqBody := map[string]string{"email": email}
	req, err := c.post("recover", reqBody)
	if err != nil {
		return nil, err
	}

	authResponse, err := c.sendRequest(req, nil)
	if err != nil {
		return nil, err
	}

	return authResponse, nil
}

func (c *Client) ResetPassword(token, password string) (*AuthResponse, error) {
	reqBody := map[string]string{"password": password}
	req, err := c.put("user?type=recovery", reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	authResponse, err := c.sendRequest(req, nil)
	if err != nil {
		return nil, err
	}

	return authResponse, nil
}

func (c *Client) post(endpoint string, data any) (*http.Request, error) {
	return c.createRequest(http.MethodPost, endpoint, data)
}

func (c *Client) put(endpoint string, data any) (*http.Request, error) {
	return c.createRequest(http.MethodPut, endpoint, data)
}

func (c *Client) createRequest(method, endpoint string, data any) (*http.Request, error) {
	if c.BaseUrl == "" {
		return nil, errors.New("supabase api url is empty")
	}

	reqUrl := c.BaseUrl

	if endpoint != "" {
		reqUrl = fmt.Sprintf("%s/%s", reqUrl, endpoint)
	}

	reqBody, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, reqUrl, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return req, nil
}

func (c *Client) sendRequest(req *http.Request, successValue any) (*AuthResponse, error) {
	req.Header.Set("apikey", c.ApiKey)

	res, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	response := AuthResponse{
		Status: res.StatusCode,
	}

	ok := res.StatusCode >= 200 && res.StatusCode < 300
	if !ok {
		errorValue := ErrorResponse{}
		err = json.NewDecoder(res.Body).Decode(&errorValue)
		if err != nil {
			return nil, err
		}

		response.Data = errorValue

		return &response, nil
	}

	if res.StatusCode != http.StatusNoContent && successValue != nil {
		err = json.NewDecoder(res.Body).Decode(&successValue)
		if err != nil {
			return nil, err
		}

		response.Data = successValue
	}

	return &response, nil
}
