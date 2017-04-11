package dbmapping

type (
	Mapping interface {
		NewTable(name string, fields []Field) (Table, error)
	}

	Table interface {
		CreateTable() error
		Insert(doc map[string]interface{}) error
		Update(doc map[string]interface{}) error
		QueryOne(fields []string, where []WhereClause, sort []OrderByClause) (map[string]interface{}, error)
		Query(fields []string, where []WhereClause, sort []OrderByClause, limit []int) ([]map[string]interface{}, error)
	}

	Field struct {
		Name       string
		PrimaryKey bool
		Type       string
		Null       bool
		Default    string
		Comment    string
	}

	WhereClause struct {
		Field string
		Type  string
		Value string
	}

	OrderByClause struct {
		Field string
		Type  string
	}
)

var (
	DiffOperator           = "<>"
	EqualOperator          = "="
	LessOperator           = "<"
	LessOrEqualOperator    = "<="
	GreaterOperator        = ">"
	GreaterOrEqualOperator = ">="
	OrderByASC             = "ASC"
	OrderByDESC            = "DESC"
)
