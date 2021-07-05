package unmarshaler

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

type UnmarshalledArbitraryObject struct {
	Body interface{}
}

func (obj *UnmarshalledArbitraryObject) GetProperty(keys ...string) interface{} {
	body := obj.Body
	for _, key := range keys {
		body = body.(map[string]interface{})[key]
	}
	return body
}

func (obj *UnmarshalledArbitraryObject) SetProperty(keys []string, value interface{}) {
	prevKeys := keys[:len(keys)-1]
	lastKey := keys[len(keys)-1]

	body := obj.Body
	for _, key := range prevKeys {
		body = body.(map[string]interface{})[key]
	}
	body.(map[string]interface{})[lastKey] = value
}

func (obj *UnmarshalledArbitraryObject) DeleteProperty(keys ...string) {
	prevKeys := keys[:len(keys)-1]
	lastKey := keys[len(keys)-1]

	body := obj.Body
	for _, key := range prevKeys {
		body = body.(map[string]interface{})[key]
	}
	delete(body.(map[string]interface{}), lastKey)
}

func UnmarshalJSON(str *string) UnmarshalledArbitraryObject {
	return UnmarshalledArbitraryObject{unmarshalArbitraryFormat(json.Unmarshal, str)}
}

func UnmarshalYAML(str *string) UnmarshalledArbitraryObject {
	return UnmarshalledArbitraryObject{unmarshalArbitraryFormat(yaml.Unmarshal, str)}
}

func unmarshalArbitraryFormat(unmarshal func([]byte, interface{}) error, str *string) interface{} {
	var body interface{}
	err := unmarshal([]byte(*str), &body)
	if err != nil {
		panic(err)
	}
	return body
}
