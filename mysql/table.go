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
	fields := make([]string, 0, len(doc))
	values := make([]interface{}, 0, len(doc))

	for key, value := range doc {
		if _, ok := t.Fields[key]; !ok {
			return fmt.Errorf("Trying to add unknown field[%s]", key)
		}

		fields = append(fields, key)
		values = append(values, value)
	}

	sql := fmt.Sprintf(`
        INSERT INTO %s (%s)
        VALUES (%s)
    `, t.Name, strings.Join(fields, ","), strings.Repeat("?,", len(fields)-1)+"?")

	stmt, err := t.db.Prepare(sql)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.Exec(values...)
	return err
}

func (t *Table) Update(doc map[string]interface{}) error {
	primaryKeyFields := make([]string, 0, 1)
	primaryKeyValues := make([]interface{}, 0, 1)
	fields := make([]string, 0, len(doc))
	values := make([]interface{}, 0, len(doc))

	for key, value := range doc {
		field, ok := t.Fields[key]
		if !ok {
			return fmt.Errorf("Trying to update unknown field[%s]", key)
		}

		if field.PrimaryKey {
			primaryKeyFields = append(primaryKeyFields, key+"=?")
			primaryKeyValues = append(primaryKeyValues, value)
			continue
		}

		fields = append(fields, key+"=?")
		values = append(values, value)
	}

	sql := fmt.Sprintf(`
        UPDATE %s
        SET %s
        WHERE %s
    `, t.Name, strings.Join(fields, ","), strings.Join(primaryKeyFields, " AND "))

	stmt, err := t.db.Prepare(sql)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.Exec(append(values, primaryKeyValues...)...)
	return err
}

func (t *Table) generateSQL(fields []string, where []dbmapping.WhereClause, sort []dbmapping.OrderByClause, limit []int) (string, []string, error) {
	if fields == nil {
		fields = make([]string, 0, len(t.Fields))

		for field := range t.Fields {
			fields = append(fields, field)
		}
	}

	sql := fmt.Sprintf(`SELECT %s FROM %s`, strings.Join(fields, ","), t.Name)

	if len(where) > 0 {
		clauses := make([]string, 0, len(where))

		for _, clause := range where {
			fieldMap := t.Fields[clause.Field]

			var value string
			if typeIsString(fieldMap) {
				value = fmt.Sprintf("'%s'", clause.Value)
			} else {
				value = clause.Value
			}

			clauses = append(clauses, fmt.Sprintf("%s%s%s", clause.Field, clause.Type, value))
		}

		sql += " WHERE " + strings.Join(clauses, " AND ")
	}

	if len(sort) > 0 {
		ordersBy := make([]string, 0, len(sort))

		for _, orderBy := range sort {
			ordersBy = append(ordersBy, fmt.Sprintf("%s %s", orderBy.Field, orderBy.Type))
		}

		sql += " ORDER BY " + strings.Join(ordersBy, ", ")
	}

	if limit != nil {
		if len(limit) > 0 {
			sql += fmt.Sprintf(" LIMIT %d", limit[0])
		}
		if len(limit) > 1 {
			sql += fmt.Sprintf(", %d", limit[1])
		}
		if len(limit) > 2 {
			return "", nil, errors.New("Invalid format to limit use []int{offset} or []int{offset, length}")
		}
	}

	return sql, fields, nil
}

func (t *Table) QueryOne(fields []string, where []dbmapping.WhereClause, sort []dbmapping.OrderByClause) (map[string]interface{}, error) {
	querySQL, fields, err := t.generateSQL(fields, where, sort, nil)
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
	querySQL, fields, err := t.generateSQL(fields, where, sort, limit)
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
