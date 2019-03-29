package dbmapping

type DB interface {
}

type Row interface {
}

type Query interface {
}

type Marshaler interface {
	MarshalDB() (interface{}, error)
}

type Unmarshaler interface {
	UnmarshalDB(map[string]interface{}) error
}

// https://gowalker.org/github.com/golang/appengine/datastore#Query
