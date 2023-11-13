package api

import (
	"encoding/json"
	"fmt"
	"github.com/datasparq-ai/houston/model"
	"github.com/gorilla/mux"
	"io"
	"net/http"
)

// GetKey godoc
// @Summary Get key information.
// @Description Returns key information (name and usage).
// @ID get-key
// @Tags Key
// @Param x-access-key header string true "Houston Key"
// @Success 200 {object} model.Success
// @Failure 404,500 {object} model.Error
// @Router /api/v1/key [get]
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

// TODO: test this in the swagger UI

// GetKeyWebhook godoc
// @Summary Redirect to the dashboard UI and auto sign in with this key
// @Description Handles a GET request using the full base URL + key URL, i.e. "https://houston.example.com/api/v1/key/myhoustonkey".
// Redirects to the dashboard with the key as a parameter to automatically sign in with that key.
// @ID get-key-webhook
// @Tags Key
// @Param key path string true "Houston Key ID"
// @Success 200 {object} model.Success
// @Failure 404,500 {object} model.Error
// @Router /api/v1/key [get]
func (a *API) GetKeyWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"] // key has been checked by checkKey middleware
	http.Redirect(w, r, fmt.Sprintf("/?key=%v", key), http.StatusMovedPermanently)
}

// PostKey creates a new key.
// PostKey godoc
// @Summary Create a new key.
// @Description The request does not need to contain a body if a random key should be generated. The newly created key is returned as bytes.
// @ID post-key
// @Tags Key
// @Param Body body model.Key true "The id, name and usage of key"
// @Success 200 string key
// @Failure 404,500 {object} model.Error
// @Router /api/v1/key [post]
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

// ListKeys godoc
// @Summary Returns a list of all Houston keys.
// @Description
// @ID list-keys
// @Tags Key
// @Success 200 {object} model.Success
// @Failure 404,500 {object} model.Error
// @Router /api/v1/key/all [get]
func (a *API) ListKeys(w http.ResponseWriter, r *http.Request) {

	keyList, err := a.db.ListKeys()
	if err != nil {
		handleError(err, w)
		return
	}
	payload, _ := json.Marshal(keyList)
	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}

// DeleteKey godoc
// @Summary Deletes key.
// @Description Deletes key extracted from header
// @ID delete-key
// @Tags Key
// @Param x-access-key header string true "Houston Key"
// @Success 200 {object} model.Success
// @Failure 404,500 {object} model.Error
// @Router /api/v1/key [delete]
func (a *API) deleteKey(w http.ResponseWriter, r *http.Request) {

	key := r.Header.Get("x-access-key")

	// Delete key extracted from header
	err := a.DeleteKey(key)

	if err != nil {
		handleError(err, w)
		return
	}

	payload, _ := json.Marshal(model.Success{Message: "Deleted key" + key})

	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}
