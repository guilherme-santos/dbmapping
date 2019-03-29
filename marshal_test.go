package dbmapping_test

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/guilherme-santos/dbmapping"
	"github.com/stretchr/testify/assert"
)

type User struct {
	Person    `db:",inline"`
	Username  string
	Password  string    `db:"passwd,omitempty"`
	Admin     bool      `db:"-"`
	LastLogin time.Time `db:"last_login"`
}

type Person struct {
	ID              int `db:",pk"`
	Name            string
	Age             int
	Phone           []Phone  `db:"phones"`
	DeliveryAddress *Address `db:"delivery_address"`
	BillingAddress  *Address `db:"billing_address"`
}

type Phone struct {
	CountryCode int
	CityCode    int
	Number      string
}

func (p *Phone) MarshalDB() (interface{}, error) {
	return fmt.Sprintf("+%d%d%s", p.CountryCode, p.CityCode, p.Number), nil
}

type Address struct {
	ID      int `db:",pk"`
	Street  string
	ZipCode string
	City    string
}

func (u *User) MarshalDB() (interface{}, error) {
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

	if u.Password != "" {
		userMap["passwd"] = fmt.Sprintf("%x", md5.Sum([]byte(u.Password)))
	}

	return userMap, nil
}

func TestMarshal(t *testing.T) {
	u := &User{
		Person: Person{
			ID:   1,
			Name: "Guilherme",
			Age:  22,
			Phone: []Phone{
				{55, 11, "33333333"},
				{49, 173, "3333333"},
			},
			DeliveryAddress: &Address{
				ID:      2,
				Street:  "Oranienburger Str. 70",
				ZipCode: "10117",
				City:    "Berlin",
			},
		},
		Username: "guilherme-santos",
		Password: "my-passwd",
		Admin:    true,
	}

	userMap, err := dbmapping.Marshal(u)
	assert.NoError(t, err)
	assert.NotNil(t, userMap)
	assert.Nil(t, userMap["person"])
	assert.Equal(t, u.ID, userMap["id"])
	assert.Equal(t, u.Name, userMap["name"])
	assert.Equal(t, u.Age, userMap["age"])
	assert.Equal(t, []string{
		"+551133333333",
		"+491733333333",
	}, userMap["phones"])
	assert.Equal(t, map[string]interface{}{
		"__pk":    "id",
		"id":      u.DeliveryAddress.ID,
		"street":  u.DeliveryAddress.Street,
		"zipcode": u.DeliveryAddress.ZipCode,
		"city":    u.DeliveryAddress.City,
	}, userMap["delivery_address"])
	assert.Nil(t, userMap["billing_address"])
	assert.Equal(t, fmt.Sprintf("%x", md5.Sum([]byte(u.Password))), userMap["passwd"])
	assert.Nil(t, userMap["admin"])

	j, _ := json.MarshalIndent(userMap, "", "    ")
	fmt.Println(string(j))
}

func TestMarshal_LowerCaseName(t *testing.T) {
	type Entity struct {
		ID          int `db:",pk"`
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
		ID     int          `db:",pk"`
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
		ID    int    `db:",pk"`
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
		ID      int `db:",pk"`
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
		ID     int `db:",pk"`
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

func TestMarshal_PrimaryKey(t *testing.T) {
	type Entity struct {
		ID int `db:",pk"`
	}

	e := Entity{
		ID: 1,
	}
	entityMap, err := dbmapping.Marshal(&e)
	assert.NoError(t, err)
	assert.NotNil(t, entityMap)
	assert.Equal(t, e.ID, entityMap["id"])
	assert.Equal(t, "id", entityMap["__pk"])
}

func TestMarshal_NoPrimaryKey(t *testing.T) {
	type Entity struct {
		ID int
	}

	e := Entity{
		ID: 1,
	}

	assert.Panics(t, func() {
		dbmapping.Marshal(&e)
	})
}

func TestMarshal_TwoPrimaryKey(t *testing.T) {
	type Entity struct {
		ID       int `db:",pk"`
		PersonID int `db:",pk"`
	}

	e := Entity{
		ID: 1,
	}

	assert.Panics(t, func() {
		dbmapping.Marshal(&e)
	})
}

func TestMarshal_DelegatePrimaryKey(t *testing.T) {
	type Parent struct {
		ID int `db:",pk"`
	}
	type Entity struct {
		Parent `db:",inline"`
	}

	e := Entity{
		Parent: Parent{
			ID: 1,
		},
	}

	entityMap, err := dbmapping.Marshal(&e)
	assert.NoError(t, err)
	assert.NotNil(t, entityMap)
	assert.Equal(t, e.ID, entityMap["id"])
	assert.Equal(t, "id", entityMap["__pk"])
}

func TestMarshal_OverridePrimaryKey(t *testing.T) {
	type Parent struct {
		ID int `db:",pk"`
	}
	type Entity struct {
		Parent   `db:",inline"`
		EntityID int `db:"entity_id,pk"`
	}

	e := Entity{
		EntityID: 1,
		Parent: Parent{
			ID: 2,
		},
	}

	entityMap, err := dbmapping.Marshal(&e)
	assert.NoError(t, err)
	assert.NotNil(t, entityMap)
	assert.Equal(t, e.EntityID, entityMap["entity_id"])
	assert.Equal(t, e.Parent.ID, entityMap["id"])
	assert.Equal(t, "entity_id", entityMap["__pk"])
}
