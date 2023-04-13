package database

// Database represents the store of API Keys, Missions, and Plans in either Redis or a local in memory database
// all database types should behave exactly the same. The schemas are described in docs/database_schema.md.
type Database interface {
	Ping() error
	CreateKey(key string) error
	DeleteKey(key string) error
	ListKeys() ([]string, error)
	Set(key string, field string, value string) error
	Get(key string, field string) (string, bool)
	Delete(key string, field string) bool
	List(key string, prefix string) ([]string, error)
	DoTransaction(transactionFunc func(string) (string, error), key string, field string) error
	Health() error
	//ListMissions(key string) ([]string, error)
}

type MemoryUsageError struct {
}

func (e *MemoryUsageError) Error() string {
	return "Memory usage is above the safe limit"
}
