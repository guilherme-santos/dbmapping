package dbmapping

type DB interface {
}

type Row interface {
}

type Query interface {
}

type Marshaler interface {
	MarshalDB() (map[string]interface{}, error)
}

type Unmarshaler interface {
	UnmarshalDB(map[string]interface{}) error
}
