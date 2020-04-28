package data

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONText is a json.RawMessage, which is a []byte underneath.
// Value() validates the json format in the source, and returns an error if
// the json is not valid.  Scan does no validation.  JSONText additionally
// implements `Unmarshal`, which unmarshals the json within to an interface{}
type JSONText json.RawMessage

var EmptyJSONText = JSONText("{}")

func StructToJSONText(v interface{}) (JSONText, error) {
	j, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return j, nil
}

// MarshalJSON returns the *j as the JSON encoding of j.
func (j JSONText) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return EmptyJSONText, nil
	}
	return j, nil
}

// UnmarshalJSON sets *j to a copy of Repo
func (j *JSONText) UnmarshalJSON(data []byte) error {
	if j == nil {
		return fmt.Errorf("JSONText: UnmarshalJSON on nil pointer")
	}
	*j = append((*j)[0:0], data...)
	return nil
}

// Value returns j as a value.  This does a validating unmarshal into another
// RawMessage.  If j is invalid json, it returns an error.
func (j JSONText) Value() (driver.Value, error) {
	var m json.RawMessage
	var err = j.Unmarshal(&m)
	if err != nil {
		return []byte{}, err
	}
	return []byte(j), nil
}

// Scan stores the src in *j.  No validation is done.
func (j *JSONText) Scan(src interface{}) error {
	var source []byte
	switch t := src.(type) {
	case string:
		source = []byte(t)
	case []byte:
		if len(t) == 0 {
			source = EmptyJSONText
		} else {
			source = t
		}
	case nil:
		*j = EmptyJSONText
	default:
		return fmt.Errorf("incompatible type for JSONText")
	}
	*j = append((*j)[0:0], source...)
	return nil
}

// Unmarshal unmarshal's the json in j to v, as in json.Unmarshal.
func (j *JSONText) Unmarshal(v interface{}) error {
	if len(*j) == 0 {
		*j = EmptyJSONText
	}
	return json.Unmarshal(*j, v)
}

func (j *JSONText) AsMap() map[string]interface{} {
	var hash map[string]interface{}
	err := j.Unmarshal(hash)
	if err != nil {
		// TODO: maybe log this, don't want to return an error
	}
	return hash
}

// String supports pretty printing for JSONText types.
func (j JSONText) String() string {
	return string(j)
}
