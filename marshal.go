package dbmapping

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

var marshalerType = reflect.TypeOf((*Marshaler)(nil)).Elem()

func asMarshaler(v reflect.Value) (Marshaler, bool) {
	if !v.CanAddr() {
		return nil, false
	}

	va := v.Addr()
	if !va.CanInterface() || !va.Type().Implements(marshalerType) {
		return nil, false
	}

	return va.Interface().(Marshaler), true
}

func asMap(v reflect.Value, res interface{}) (map[string]interface{}, error) {
	resMap, ok := res.(map[string]interface{})
	if !ok {
		panic(fmt.Sprintf("Type %s should return an map[string]interface{} but %T", v.Type().Name(), res))
	}

	if _, ok := resMap["__pk"]; !ok {
		fmt.Println(resMap)
		panic("No primary key defined")
	}

	return resMap, nil
}

func Marshal(v interface{}) (map[string]interface{}, error) {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return nil, errors.New("dbmapping: interface must be a pointer to struct")
	}

	res, err := decodeStruct(val.Elem())
	if err != nil {
		return nil, err
	}

	return asMap(val, res)
}

func splitTag(f reflect.StructField) (fieldName string, option string, ok bool) {
	tag := strings.Split(f.Tag.Get("db"), ",")

	ok = true
	if len(tag) == 0 || tag[0] == "" {
		fieldName = strings.ToLower(f.Name)
	} else if tag[0] == "-" {
		ok = false
	} else {
		fieldName = tag[0]
	}
	if len(tag) > 1 {
		option = tag[1]
	}

	return
}

func decodeStruct(v reflect.Value) (interface{}, error) {
	k := v.Kind()
	if k != reflect.Struct {
		panic(fmt.Sprintf("element should be struct but received %s", k))
	}

	if m, ok := asMarshaler(v); ok {
		return m.MarshalDB()
	}

	res := map[string]interface{}{}

	var pkFromParent bool

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		fieldName, option, ok := splitTag(t.Field(i))
		if !ok {
			continue
		}

		fieldval, err := decodeField(v.Field(i))
		if err != nil {
			return nil, err
		}

		switch option {
		case "inline":
			fieldKind := v.Field(i).Kind()
			if fieldKind == reflect.Ptr || fieldKind == reflect.Struct {
				if fieldMap, ok := fieldval.(map[string]interface{}); ok {
					for k, v := range fieldMap {
						if k == "__pk" {
							pkFromParent = true
						}
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

		case "pk":
			if !pkFromParent {
				if _, ok := res["__pk"]; ok {
					panic(fmt.Sprintf("primary key already defined: %s", res["__pk"]))
				}
			}
			res["__pk"] = fieldName
		}

		res[fieldName] = fieldval
		if err != nil {
			return nil, err
		}
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
		return decodeStruct(v)

	case reflect.Array, reflect.Slice:
		array := make([]interface{}, v.Len())

		for i := 0; i < v.Len(); i++ {
			res, err := decodeField(v.Index(i))
			if err != nil {
				return nil, err
			}

			array[i] = res
		}

		return array, nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int(), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint(), nil

	default:
		if v.CanInterface() {
			return v.Interface(), nil
		}

		fmt.Println("decodeField: cannot get interface", k, v)
	}

	return nil, nil
}
