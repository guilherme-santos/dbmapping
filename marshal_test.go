package dbmapping_test

import (
	"crypto/md5"
	"fmt"
	"testing"

	"github.com/guilherme-santos/dbmapping"
	"github.com/stretchr/testify/assert"
)

type Person struct {
	ID              int `db:",pk"`
	Name            string
	Age             int
	DeliveryAddress *Address `db:"delivery_address"`
	BillingAddress  *Address `db:"billing_address"`
}

type Address struct {
	Street  string
	ZipCode string
	City    string
}

type User struct {
	Person   `db:",inline"`
	Password string `db:"passwd,omitempty"`
	Logged   bool   `db:"-"`
}

func (u *User) MarshalDB() (map[string]interface{}, error) {
	type Orig User
	hack := struct {
		*Orig `db:",inline"`
	}{
		(*Orig)(u),
	}

	userMap, err := dbmapping.Marshal(&hack)
	if err != nil {
		return nil, err
	}

	if passwd, ok := userMap["passwd"]; ok {
		userMap["passwd"] = fmt.Sprintf("%x", md5.Sum([]byte(passwd.(string))))
	}

	return userMap, nil
}

func TestMarshal(t *testing.T) {
	u := &User{
		Person: Person{
			ID:   1,
			Name: "Guilherme",
			Age:  22,
			DeliveryAddress: &Address{
				Street:  "Oranienburger Str. 70",
				ZipCode: "10117",
				City:    "Berlin",
			},
		},
		Password: "my-passwd",
		Logged:   true,
	}

	userMap, err := dbmapping.Marshal(u)
	assert.NoError(t, err)
	assert.NotNil(t, userMap)

	assert.Equal(t, 1, userMap["id"])
	assert.Equal(t, u.Name, userMap["name"])
	assert.Equal(t, u.Age, userMap["age"])
	assert.Equal(t, map[string]interface{}{
		"street":  u.DeliveryAddress.Street,
		"zipcode": u.DeliveryAddress.ZipCode,
		"city":    u.DeliveryAddress.City,
	}, userMap["delivery_address"])
	assert.Nil(t, userMap["billing_address"])
	assert.Equal(t, fmt.Sprintf("%x", md5.Sum([]byte(u.Password))), userMap["passwd"])
	assert.Nil(t, userMap["logged"])
	assert.Nil(t, userMap["-"])

	// j, _ := json.MarshalIndent(userMap, "", "    ")
	// fmt.Println(string(j))
}

func TestMarshal_LowerCaseName(t *testing.T) {
	type Entity struct {
		MyFieldTest string
	}

	e := Entity{
		MyFieldTest: "test",
	}
	entityMap, err := dbmapping.Marshal(&e)
	assert.NoError(t, err)
	assert.NotNil(t, entityMap)
	assert.Equal(t, e.MyFieldTest, entityMap["myfieldtest"])
}

func TestMarshal_Inline(t *testing.T) {
	type InlineEntity struct {
		Field string
	}
	type Entity struct {
		Inline InlineEntity `db:",inline"`
	}

	e := Entity{
		Inline: InlineEntity{
			Field: "test",
		},
	}
	entityMap, err := dbmapping.Marshal(&e)
	assert.NoError(t, err)
	assert.NotNil(t, entityMap)
	assert.Nil(t, entityMap["inline"])
	assert.Equal(t, e.Inline.Field, entityMap["field"])
}

func TestMarshal_OmitEmpty(t *testing.T) {
	type Entity struct {
		Field string `db:",omitempty"`
	}

	var e Entity
	entityMap, err := dbmapping.Marshal(&e)
	assert.NoError(t, err)
	assert.NotNil(t, entityMap)
	assert.Nil(t, entityMap["field"])

	e.Field = "test"
	entityMap, err = dbmapping.Marshal(&e)
	assert.NoError(t, err)
	assert.NotNil(t, entityMap)
	assert.Equal(t, e.Field, entityMap["field"])
}

func TestMarshal_PointersAreOmitEmptyByDefault(t *testing.T) {
	type Entity struct {
		Field   string
		Address *Address
	}

	e := Entity{
		Field: "test",
	}
	entityMap, err := dbmapping.Marshal(&e)
	assert.NoError(t, err)
	assert.NotNil(t, entityMap)
	assert.Equal(t, e.Field, entityMap["field"])
	assert.Nil(t, entityMap["address"])
}

func TestMarshal_IgnoreField(t *testing.T) {
	type Entity struct {
		Field  string
		Ignore bool `db:"-"`
	}

	e := Entity{
		Field:  "test",
		Ignore: true,
	}
	entityMap, err := dbmapping.Marshal(&e)
	assert.NoError(t, err)
	assert.NotNil(t, entityMap)
	assert.Equal(t, e.Field, entityMap["field"])
	assert.Nil(t, entityMap["-"])
	assert.Nil(t, entityMap["ignore"])
}
