package database

import (
  "fmt"
  "github.com/datasparq-ai/houston/model"
  "sync"
)

// LocalDatabase is an in memory database for development and testing purposes only. A single mutex is used per key to
// ensure that mission updates are transactional. This impacts performance when there are multiple users.
type LocalDatabase struct {
  Database
  kv  map[string]map[string]string
  mux map[string]*sync.Mutex
}

func NewLocalDatabase() *LocalDatabase {
  d := LocalDatabase{kv: make(map[string]map[string]string), mux: make(map[string]*sync.Mutex)}
  return &d
}

func (d *LocalDatabase) Ping() error {
  return nil
}

func (d *LocalDatabase) CreateKey(key string) error {
  d.kv[key] = make(map[string]string)
  d.mux[key] = &sync.Mutex{}
  return nil
}

func (d *LocalDatabase) DeleteKey(key string) error {
  delete(d.kv, key)
  delete(d.mux, key)
  return nil
}

func (d *LocalDatabase) Set(key string, field string, value string) error {
  if _, ok := d.kv[key]; !ok {
    return fmt.Errorf("key '%v' not found", key)
  }
  d.mux[key].Lock()
  defer d.mux[key].Unlock()
  d.kv[key][field] = value
  return nil
}

// Get returns the value for the key and field specified, along with a boolean to say whether that key and value exist
func (d *LocalDatabase) Get(key string, field string) (string, bool) {
  if _, ok := d.kv[key]; !ok {
    return "", ok
  }
  val, ok := d.kv[key][field]
  return val, ok
}

// Delete returns true if the field specified was successfully deleted or did not exist
func (d *LocalDatabase) Delete(key string, field string) bool {
  if _, ok := d.kv[key]; !ok {
    return false // key does not exist
  }
  delete(d.kv[key], field)
  return true
}

func (d *LocalDatabase) listFields(key string) ([]string, error) {
  if _, ok := d.kv[key]; !ok {
    return []string{}, fmt.Errorf("key '%v' not found", key)
  }
  var fieldList []string
  for field := range d.kv[key] {
    fieldList = append(fieldList, field)
  }
  return fieldList, nil
}

func (d *LocalDatabase) ListKeys() ([]string, error) {
  var keyList []string
  for key := range d.kv {
    keyList = append(keyList, key)
  }
  return keyList, nil
}

func (d *LocalDatabase) ListPlans(key string) ([]string, error) {
  allFields, err := d.listFields(key)
  if err != nil {
    return allFields, err
  }
  var plans []string
  for _, field := range allFields {
    if len(field) >= 2 && field[:2] == "p|" {
      plans = append(plans, field[2:])
    }
  }
  return plans, nil
}

func (d *LocalDatabase) ListMissions(key string) ([]string, error) {
  allFields, err := d.listFields(key)
  if err != nil {
    return allFields, err
  }
  var missions []string
Loop:
  for _, field := range allFields {
    // remove system keys
    for _, k := range []string{"n", "u"} {
      if field == k {
        continue Loop
      }
    }
    // remove plans
    if len(field) >= 2 && field[:2] == "p|" {
      continue Loop
    }
    missions = append(missions, field)
  }
  return missions, nil
}

func (d *LocalDatabase) DoTransaction(transactionFunc func(string) (string, error), key string, field string) error {

  d.mux[key].Lock()
  defer d.mux[key].Unlock()

  value, ok := d.Get(key, field)
  var err error
  if !ok {
    return &model.KeyNotFoundError{}
  }
  value, err = transactionFunc(value)
  if err != nil {
    return err
  }
  d.kv[key][field] = value
  return err
}
