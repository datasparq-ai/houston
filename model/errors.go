package model

type TransactionFailedError struct{}

func (m *TransactionFailedError) Error() string {
  return "The key was modified during the transaction."
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
