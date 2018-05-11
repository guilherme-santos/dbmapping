package dbmapping

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

func Marshal(v interface{}) (map[string]interface{}, error) {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return nil, errors.New("dbmapping: interface must be a pointer to struct")
	}

	return decode(val.Elem())
}

var marshalerType = reflect.TypeOf((*Marshaler)(nil)).Elem()

func decode(v reflect.Value) (map[string]interface{}, error) {
	res := map[string]interface{}{}

	k := v.Kind()
	switch k {
	case reflect.Struct:
		if v.CanAddr() {
			va := v.Addr()
			if va.CanInterface() && va.Type().Implements(marshalerType) {
				return va.Interface().(Marshaler).MarshalDB()
			}
		}

		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)
			dbTag := strings.Split(sf.Tag.Get("db"), ",")

			var fieldName string
			if len(dbTag) > 0 {
				if dbTag[0] == "" {
					fieldName = strings.ToLower(sf.Name)
				} else if dbTag[0] == "-" && len(dbTag) == 1 {
					// db:"-," it means the name is -
					continue
				} else {
					fieldName = dbTag[0]
				}
			} else {
				fieldName = strings.ToLower(sf.Name)
			}

			fieldval, err := decodeField(v.Field(i))
			if err != nil {
				return nil, err
			}
			if fieldval == nil {
				continue
			}

			if len(dbTag) > 1 {
				switch dbTag[1] {
				case "inline":
					fieldKind := v.Field(i).Kind()

					if fieldKind == reflect.Ptr || fieldKind == reflect.Struct {
						if fieldMap, ok := fieldval.(map[string]interface{}); ok {
							for k, v := range fieldMap {
								res[k] = v
							}
						}

						continue
					}

				case "omitempty":
					zeroValue := reflect.Zero(v.Type()).Interface()
					if reflect.DeepEqual(zeroValue, v.Interface()) {
						continue
					}
				}
			}

			res[fieldName] = fieldval
			if err != nil {
				return nil, err
			}
		}
	default:
		fmt.Println("decode: kind:", k)
	}

	return res, nil
}

func decodeField(v reflect.Value) (interface{}, error) {
	k := v.Kind()
	switch k {
	case reflect.Ptr:
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
		fallthrough
	case reflect.Struct:
		return decode(v)
	default:
		return v.Interface(), nil
	}

	return nil, nil
}
