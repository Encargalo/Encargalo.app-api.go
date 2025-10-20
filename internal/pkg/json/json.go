package json

import (
	"encoding/json"
	"fmt"
	"reflect"
)

func StructToMap(st interface{}) (map[string]interface{}, error) {
	if vc := reflect.ValueOf(st); vc.Kind() != reflect.Struct {
		return nil, fmt.Errorf("the parameter received is of type '%v', it should be a 'struct'", vc.Kind())
	}

	jsonRaw, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}

	var toMap map[string]interface{}
	if err = json.Unmarshal(jsonRaw, &toMap); err != nil {
		return nil, err
	}

	return toMap, nil
}
