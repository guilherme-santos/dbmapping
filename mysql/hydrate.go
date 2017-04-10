package mysql

import (
	"database/sql"
	"fmt"
	"strings"

	mysqldriver "github.com/go-sql-driver/mysql"
)

type dbScan interface {
	Scan(dest ...interface{}) error
}

func (t *Table) hydrate(fields []string, row dbScan) (map[string]interface{}, error) {
	result := make([]interface{}, 0, len(fields))

	for _, field := range fields {
		fieldMap, ok := t.Fields[field]
		if !ok {
			return nil, fmt.Errorf("Cannot select field[%s]", field)
		}

		if typeIsString(fieldMap) {
			result = append(result, &sql.NullString{})
		} else if strings.HasPrefix(fieldMap.Type, "INT") {
			result = append(result, &sql.NullInt64{})
		} else if strings.HasPrefix(fieldMap.Type, "DATE") || strings.HasPrefix(fieldMap.Type, "TIMESTAMP") {
			result = append(result, &mysqldriver.NullTime{})
		} else {
			return nil, fmt.Errorf("Cannot read type[%s] from database", fieldMap.Type)
		}
	}

	err := row.Scan(result...)
	if err != nil {
		return nil, err
	}

	doc := make(map[string]interface{})

	for k, field := range fields {
		switch value := result[k].(type) {
		case *sql.NullString:
			if value.Valid {
				doc[field] = value.String
			}
		case *sql.NullInt64:
			if value.Valid {
				doc[field] = value.Int64
			}
		case *mysqldriver.NullTime:
			if value.Valid {
				doc[field] = value.Time
			}
		}
	}

	return doc, nil
}
