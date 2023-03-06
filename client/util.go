package client

import (
	"encoding/json"
	"fmt"
	"github.com/datasparq-ai/houston/model"
	"io"
	"net/http"
)

func handleInvalidResponse(err error) error {
	return fmt.Errorf("response from houston API was wrong format: %v", err)
}

func handleErrorResponse(responseBody []byte) error {
	var errorResponse model.Error
	fmt.Println(string(responseBody))
	err := json.Unmarshal(responseBody, &errorResponse)
	if err != nil {
		return handleInvalidResponse(err)
	}
	switch errorResponse.Type {
	case "model.KeyNotFoundError":
		err = &model.KeyNotFoundError{}
	case "model.PlanNotFoundError":
		err = &model.PlanNotFoundError{}
	default:
		err = fmt.Errorf(errorResponse.Message)
	}
	return err
}

func parseResponse(resp *http.Response, parsedResponse interface{}) error {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return handleInvalidResponse(err)
	}
	if resp.StatusCode != 200 {
		if resp.StatusCode == http.StatusTooManyRequests {
			return fmt.Errorf("reached the maximum number of 429 error responses (100) when making request to " + resp.Request.URL.String())
		} else {
			return handleErrorResponse(responseBody)
		}
	}
	err = json.Unmarshal(responseBody, parsedResponse)
	if err != nil {
		return handleInvalidResponse(err)
	}
	return err
}
