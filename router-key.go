package main

import (
	"encoding/json"
	"github.com/datasparq-ai/houston/model"
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

// PostKey is used to create a new key. The request does not need to contain a body if a random key should be generated.
// the newly created key is returned as bytes.
// PostKey godoc
// @Summary Create a new key.
// @Description The request does not need to contain a body if a random key should be generated. The newly created key is returned as bytes.
// @ID post-key
// @Tags Key
// @Param Body body model.Key true "The id, name and usage of key"
// @Success 200 {object} model.Success
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
// @ID get-list-keys
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
// @Summary Returns a list of all Houston keys.
// @Description
// @ID delete-key
// @Tags Key
// @Param x-access-key header string true "Houston Key"
// @Success 200 {object} model.Success
// @Failure 404,500 {object} model.Error
// @Router /api/v1/key [delete]
func (a *API) DeleteKey(w http.ResponseWriter, r *http.Request) {

  key := r.Header.Get("x-access-key")

  a.db.DeleteKey(key)

  payload, _ := json.Marshal(model.Success{Message: "Deleted key" + key})

  w.Header().Set("Content-Type", "application/json")
  w.Write(payload)
}