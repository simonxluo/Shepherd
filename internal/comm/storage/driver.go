package storage

import "fmt"

// DriverFactory is a function that creates a Store from a StorageConfig.
type DriverFactory func(config *StorageConfig) (Store, error)

// registry holds all registered storage drivers.
var registry = map[StorageType]DriverFactory{}

// RegisterDriver registers a storage driver factory for a given type.
func RegisterDriver(t StorageType, factory DriverFactory) {
	registry[t] = factory
}

// createStore uses the driver registry to create a Store instance.
func createStore(config *StorageConfig) (Store, error) {
	factory, ok := registry[config.Type]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrInvalidStorageType, config.Type)
	}
	return factory(config)
}

func init() {
	RegisterDriver(StorageTypeMemory, func(config *StorageConfig) (Store, error) {
		return NewMemoryStore()
	})

	RegisterDriver(StorageTypeSQLite, func(config *StorageConfig) (Store, error) {
		if config.SQLite == nil {
			return nil, ErrMissingSQLiteConfig
		}
		return NewSQLiteStore(config.SQLite)
	})

	RegisterDriver(StorageTypePostgreSQL, func(config *StorageConfig) (Store, error) {
		if config.PostgreSQL == nil {
			return nil, ErrMissingPostgreSQLConfig
		}
		return NewPostgreSQLStore(config.PostgreSQL)
	})

	RegisterDriver(StorageTypeMySQL, func(config *StorageConfig) (Store, error) {
		if config.MySQL == nil {
			return nil, ErrMissingMySQLConfig
		}
		return NewMySQLStore(config.MySQL)
	})
}
