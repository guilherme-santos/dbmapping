package mysql

import (
	"errors"
	"fmt"
	"strings"

	"github.com/guilherme-santos/dbmapping"
)

func (t *Table) genSelectSQL(fields []string, where []dbmapping.WhereClause, sort []dbmapping.OrderByClause, limit []int) (string, []string, error) {
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

func (t *Table) genUpdateSQL(doc map[string]interface{}) (string, []interface{}, []interface{}, error) {
	primaryKeyFields := make([]string, 0, 1)
	primaryKeyValues := make([]interface{}, 0, 1)
	fields := make([]string, 0, len(doc))
	values := make([]interface{}, 0, len(doc))

	for key, value := range doc {
		field, ok := t.Fields[key]
		if !ok {
			return "", nil, nil, fmt.Errorf("Trying to update unknown field[%s]", key)
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

	return sql, values, primaryKeyValues, nil
}

func (t *Table) genInsertSQL(doc map[string]interface{}, upsert bool) (string, []interface{}, error) {
	fields := make([]string, 0, len(doc))
	values := make([]interface{}, 0, len(doc))

	for key, value := range doc {
		if _, ok := t.Fields[key]; !ok {
			return "", nil, fmt.Errorf("Trying to add unknown field[%s]", key)
		}

		fields = append(fields, key)
		values = append(values, value)
	}

	sql := fmt.Sprintf(`
        INSERT INTO %s (%s)
        VALUES (%s)
    `, t.Name, strings.Join(fields, ","), strings.Repeat("?,", len(fields)-1)+"?")

	if upsert {
		sql += " ON DUPLICATE KEY UPDATE " + strings.Join(fields, "=?,") + "=?"
	}

	return sql, values, nil
}
