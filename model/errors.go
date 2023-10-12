package model

import "net/http"

// ErrorCode maps all Houston errors to unique codes for easy identification
func ErrorCode(err error) int {
	switch err.(type) {
	case *TransactionFailedError:
		return 572
	case *TooManyRequestsError:
		return http.StatusTooManyRequests
	case *KeyNotProvidedError:
		return http.StatusUnauthorized
	case *KeyNotFoundError:
		return 470
	case *PlanNotFoundError:
		return http.StatusNotFound
	case *BadCredentialsError:
		return http.StatusForbidden
	case *InternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusBadRequest
	}
}

type TransactionFailedError struct{}

func (m *TransactionFailedError) Error() string {
	return "The key was modified during the transaction."
}

type KeyNotProvidedError struct{}

func (m *KeyNotProvidedError) Error() string {
	return "Key was not provided in the request."
}

type KeyNotFoundError struct{}

func (m *KeyNotFoundError) Error() string {
	return "Key not found."
}

type PlanNotFoundError struct {
	PlanName string
}

func (m *PlanNotFoundError) Error() string {
	return "Plan '" + m.PlanName + "' not found."
}

type TooManyRequestsError struct{}

func (m *TooManyRequestsError) Error() string {
	return "Too many requests."
}

type BadCredentialsError struct{}

func (m *BadCredentialsError) Error() string {
	return "Incorrect username/password."
}

type InternalError struct{}

func (m *InternalError) Error() string {
	return "Houston, we have a problem. There was an error in the API server when processing the request."
}
