package mysql

import (
	"database/sql"
	"errors"

	"github.com/guilherme-santos/dbmapping"
)

type Mapping struct {
	db     *sql.DB
	tables map[string]dbmapping.Table
}

var (
	DefaultCharacterSet = "utf8"
	DefaultCollate      = "utf8_unicode_ci"
	DefaultEngine       = "InnoDB"

	ErrTableAlreadyExists = errors.New("Mapping to this table was already defined")
)

func NewMapping(db *sql.DB) dbmapping.Mapping {
	return &Mapping{
		db:     db,
		tables: make(map[string]dbmapping.Table),
	}
}

func (m *Mapping) NewTable(name string, fields []dbmapping.Field) (dbmapping.Table, error) {
	if _, ok := m.tables[name]; ok {
		return nil, ErrTableAlreadyExists
	}

	fieldsMap := make(map[string]dbmapping.Field)
	for _, field := range fields {
		fieldsMap[field.Name] = field
	}

	table := &Table{
		db:           m.db,
		Name:         name,
		Fields:       fieldsMap,
		CharacterSet: DefaultCharacterSet,
		Collate:      DefaultCollate,
		Engine:       DefaultEngine,
	}
	m.tables[name] = table

	return table, nil
}
