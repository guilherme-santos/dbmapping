package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/guilherme-santos/dbmapping"
)

type Table struct {
	db *sql.DB

	Name         string
	Fields       map[string]dbmapping.Field
	CharacterSet string
	Collate      string
	Engine       string
}

var ErrNoPrimaryKey = errors.New("No primary key was informed")

func (t *Table) CreateTable() error {
	primaryKey := make([]string, 0, 1)
	fields := make([]string, 0, len(t.Fields)+1)

	for _, field := range t.Fields {
		if field.PrimaryKey {
			primaryKey = append(primaryKey, field.Name)
		}

		fieldType := field.Type
		if !field.Null {
			fieldType += " NOT"
		}
		fieldType += " NULL"

		if !strings.EqualFold("", field.Default) {
			fieldType += " DEFAULT " + field.Default
		}

		if !strings.EqualFold("", field.Comment) {
			fieldType += fmt.Sprintf(" COMMENT '%s'", field.Comment)
		}

		fields = append(fields, fmt.Sprintf("%s %s", field.Name, fieldType))
	}

	if len(primaryKey) == 0 {
		return ErrNoPrimaryKey
	}

	fields = append(fields, fmt.Sprintf("PRIMARY KEY(%s)", strings.Join(primaryKey, ", ")))

	sql := fmt.Sprintf(`
        CREATE TABLE IF NOT EXISTS %s (%s)
        DEFAULT CHARACTER SET %s COLLATE %s
        ENGINE = %s
    `, t.Name, strings.Join(fields, ", "), t.CharacterSet, t.Collate, t.Engine)

	_, err := t.db.Exec(sql)
	return err
}

func (t *Table) Insert(doc map[string]interface{}) error {
	insertSQL, values, err := t.genInsertSQL(doc, false)
	if err != nil {
		return err
	}

	stmt, err := t.db.Prepare(insertSQL)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.Exec(values...)
	return err
}

func (t *Table) Update(doc map[string]interface{}) error {
	updateSQL, values, primaryKeyValues, err := t.genUpdateSQL(doc)
	if err != nil {
		return err
	}

	stmt, err := t.db.Prepare(updateSQL)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.Exec(append(values, primaryKeyValues...)...)
	return err
}

func (t *Table) Upsert(doc map[string]interface{}) error {
	upsertSQL, values, err := t.genInsertSQL(doc, true)
	if err != nil {
		return err
	}

	stmt, err := t.db.Prepare(upsertSQL)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.Exec(append(values, values...)...)
	return err
}

func (t *Table) QueryOne(fields []string, where []dbmapping.WhereClause, sort []dbmapping.OrderByClause) (map[string]interface{}, error) {
	querySQL, fields, err := t.genSelectSQL(fields, where, sort, nil)
	if err != nil {
		return nil, err
	}

	row := t.db.QueryRow(querySQL)

	doc, err := t.hydrate(fields, row)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return doc, nil
}

func (t *Table) Query(fields []string, where []dbmapping.WhereClause, sort []dbmapping.OrderByClause, limit []int) ([]map[string]interface{}, error) {
	querySQL, fields, err := t.genSelectSQL(fields, where, sort, limit)
	if err != nil {
		return nil, err
	}

	rows, err := t.db.Query(querySQL)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var results []map[string]interface{}

	for rows.Next() {
		doc, err := t.hydrate(fields, rows)
		if err != nil {
			return nil, err
		}

		results = append(results, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
