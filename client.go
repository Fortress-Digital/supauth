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

type httpClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

type clientInterface interface {
	createAndSendRequest(method, endpoint string, data, successValue any) (*AuthResponse, error)
	createRequest(method, endpoint string, data any) (*http.Request, error)
	sendRequest(req *http.Request, successValue any) (*AuthResponse, error)
}

type AuthResponse struct {
	Status int `json:"status"`
	Data   any `json:"data"`
}

type ErrorResponse struct {
	Status    int    `json:"code"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"msg"`
}

type client struct {
	BaseUrl    string
	ApiKey     string
	HttpClient httpClientInterface
}

func newClient(projectId, apiKey string) clientInterface {
	baseUrl := fmt.Sprintf("https://%s.supabase.co/%s", projectId, authEndpoint)

	return &client{
		BaseUrl: baseUrl,
		ApiKey:  apiKey,
		HttpClient: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

func (c *client) createAndSendRequest(method, endpoint string, data, successValue any) (*AuthResponse, error) {
	req, err := c.createRequest(method, endpoint, data)
	if err != nil {
		return nil, err
	}

	return c.sendRequest(req, successValue)
}

func (c *client) createRequest(method, endpoint string, data any) (*http.Request, error) {
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

func (c *client) sendRequest(req *http.Request, successValue any) (*AuthResponse, error) {
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
		errorValue := &ErrorResponse{}
		err = json.NewDecoder(res.Body).Decode(errorValue)
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
