package mysql

import (
	"strings"

	"github.com/guilherme-santos/dbmapping"
)

func typeIsString(field dbmapping.Field) bool {
	return strings.HasPrefix(field.Type, "CHAR") || strings.HasPrefix(field.Type, "VARCHAR")
}
