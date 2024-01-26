package client

import (
	"encoding/json"
	"fmt"
	"github.com/datasparq-ai/houston/model"
	"io"
	"net/http"
	"os"
	"strings"
)

func handleInvalidResponse(err error) error {
	return fmt.Errorf("response from Houston API had wrong format; error when parsing JSON: %v", err)
}

func handleErrorResponse(responseBody []byte) error {
	var errorResponse model.Error
	err := json.Unmarshal(responseBody, &errorResponse)
	if err != nil {
		return handleInvalidResponse(err)
	}

	// this case statement is the inverse of model.ErrorCode
	switch errorResponse.Code {
	case 572:
		err = &model.TransactionFailedError{}
	case http.StatusTooManyRequests:
		err = &model.TooManyRequestsError{}
	case http.StatusUnauthorized:
		err = &model.KeyNotProvidedError{}
	case 470:
		err = &model.KeyNotFoundError{}
	case http.StatusNotFound:
		// extract plan name from error message
		if strings.Count(errorResponse.Message, "'") == 2 {
			planName := errorResponse.Message[strings.Index(errorResponse.Message, "'")+1 : strings.LastIndex(errorResponse.Message, "'")]
			err = &model.PlanNotFoundError{PlanName: planName}
		} else {
			err = &model.PlanNotFoundError{}
		}
	case http.StatusForbidden:
		err = &model.BadCredentialsError{}
	case http.StatusInternalServerError:
		err = &model.InternalError{}
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

// HandleCommandLineError prints a helpful messages for common errors and then exits.
// This is only used when Houston is being run from the command line.
func HandleCommandLineError(err error) {

	errorText := "\u001B[31mError: "
	end := "\u001B[0m"
	switch err.(type) {
	case *model.KeyNotProvidedError:
		fmt.Println(errorText + err.Error() + " The 'HOUSTON_KEY' environment variable can be used to provide the API key." + end)
	case *model.KeyNotFoundError:
		fmt.Println(
			errorText + err.Error() +
				" The API key provided with the 'HOUSTON_KEY' environment variable does not exist on this server." +
				" See the docs for a guide on creating keys: https://github.com/datasparq-ai/houston/blob/main/docs/keys.md" + end)
	case *model.PlanNotFoundError:
		fmt.Println(errorText + err.Error() + end)
	case *json.SyntaxError:
		fmt.Println(
			errorText + "Couldn't parse JSON string. " + err.Error() + end)
	default:
		fmt.Printf("Unhandled %T\n", err)
		panic(err)
	}

	os.Exit(1)
}
