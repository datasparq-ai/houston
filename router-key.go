package main

import (
	"encoding/json"
	"github.com/datasparq-ai/houston/model"
	"io"
	"net/http"
)

// GetKey returns key information (name and usage)
func (a *API) GetKey(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware

	keyName, _ := a.db.Get(key, "n")
	keyUsage, _ := a.db.Get(key, "u")

	keyObj := model.Key{
		Id:    key,
		Name:  keyName,
		Usage: keyUsage,
	}
	keyBytes, err := json.Marshal(keyObj)
	if err != nil {
		handleError(err, w)
	}
	w.Write(keyBytes)
}

// PostKey is used to create a new key. The request does not need to contain a body if a random key should be generated.
// the newly created key is returned as bytes.
func (a *API) PostKey(w http.ResponseWriter, r *http.Request) {
	var key model.Key

	reqBody, _ := io.ReadAll(r.Body)

	if len(reqBody) == 0 {
		key = model.Key{}
	} else {
		err := json.Unmarshal(reqBody, &key)
		if err != nil {
			handleError(err, w)
			return
		}
	}

	keyId, err := a.CreateKey(key.Id, key.Name)
	if err != nil {
		handleError(err, w)
		return
	}
	w.Write([]byte(keyId))
}
