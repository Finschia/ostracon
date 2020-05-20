package unmarshaler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalElementJSON(t *testing.T) {
	var unmarshalJSONTests = []struct {
		json     string
		expected interface{}
	}{
		{
			`"30"`,
			"30",
		},
		{
			"30",
			float64(30),
		},
		{
			"null",
			nil,
		},
		{
			"[]",
			[]interface{}{},
		},
		{
			`["a", "b", "c"]`,
			[]interface{}{"a", "b", "c"},
		},
		{
			"{}",
			map[string]interface{}{},
		},
		{
			`{"key1":"value1", "key2":"value2"}`,
			map[string]interface{}{"key1": "value1", "key2": "value2"},
		},
		{
			testJSON1,
			map[string]interface{}{
				"key1": float64(119),
				"sub1": map[string]interface{}{"key2": "value2", "sub2": map[string]interface{}{"key3": "value3"}},
				"sub3": map[string]interface{}{"key4": "value4", "key5": "value5"}},
		},
		{
			testJSON2,
			map[string]interface{}{
				"key1": float64(119),
				"sub1": []interface{}{map[string]interface{}{
					"key2": "value2", "sub2": map[string]interface{}{"key3": []interface{}{"value2"}}}},
				"sub3": map[string]interface{}{"key4": "value2", "key5": "value2"}},
		},
	}

	for _, tt := range unmarshalJSONTests {
		t.Logf("unmarshal json test %s", tt.json)
		{
			unmarshaledJSON := UnmarshalJSON(&tt.json)
			require.Equal(t, tt.expected, unmarshaledJSON.Body)
		}
	}
}

func TestUnmarshalElementYAML(t *testing.T) {
	var unmarshalJSONTests = []struct {
		yaml     string
		expected interface{}
	}{
		{
			`"30"`,
			"30",
		},
		{
			"30",
			30,
		},
		{
			"null",
			nil,
		},
		{
			"[]",
			[]interface{}{},
		},
		{
			`["a", "b", "c"]`,
			[]interface{}{"a", "b", "c"},
		},
		{
			"{}",
			map[string]interface{}{},
		},
		{
			"key1: value1\nkey2: value2",
			map[string]interface{}{"key1": "value1", "key2": "value2"},
		},
		{
			testYAML1,
			map[string]interface{}{
				"key1": 119,
				"sub1": map[string]interface{}{"key2": "value2", "sub2": map[string]interface{}{"key3": "value3"}},
				"sub3": map[string]interface{}{"key4": "value4", "key5": "value5"}},
		},
		{
			testYAML2,
			map[string]interface{}{
				"key1": 119,
				"sub1": []interface{}{map[string]interface{}{
					"key2": "value2", "sub2": map[string]interface{}{"key3": []interface{}{"value3"}}}},
				"sub3": map[string]interface{}{"key4": "value4", "key5": "value5"}},
		},
	}

	for _, tt := range unmarshalJSONTests {
		t.Logf("unmarshal yaml test %s", tt.yaml)
		{
			unmarshaledYAML := UnmarshalYAML(&tt.yaml)
			require.Equal(t, tt.expected, unmarshaledYAML.Body)
		}
	}
}

func TestGetAndSetProperty(t *testing.T) {
	testJSON := testJSON1
	unmarshaledJSON := UnmarshalJSON(&testJSON)
	require.Equal(t, float64(119), unmarshaledJSON.GetProperty("key1"))
	require.Equal(t, "value2", unmarshaledJSON.GetProperty("sub1", "key2"))
	require.Equal(t, "value3", unmarshaledJSON.GetProperty("sub1", "sub2", "key3"))
	require.Equal(t, "value4", unmarshaledJSON.GetProperty("sub3", "key4"))
	require.Equal(t, "value5", unmarshaledJSON.GetProperty("sub3", "key5"))

	unmarshaledJSON.SetProperty([]string{"key1"}, "newValue1")
	unmarshaledJSON.SetProperty([]string{"sub1", "sub2", "key3"}, "newValue2")

	require.Equal(t, "newValue1", unmarshaledJSON.GetProperty("key1"))
	require.Equal(t, "newValue2", unmarshaledJSON.GetProperty("sub1", "sub2", "key3"))
}

func TestDeleteProposer(t *testing.T) {
	testJSON := testJSON1
	unmarshaledJSON := UnmarshalJSON(&testJSON)

	unmarshaledJSON.DeleteProperty("sub3", "key5")
	require.Nil(t, unmarshaledJSON.GetProperty("sub3", "key5"))
}

const (
	testJSON1 = `{"key1":119, "sub1":{"key2":"value2", "sub2":{"key3":"value3"}},
"sub3":{"key4":"value4", "key5":"value5"}}`
	testJSON2 = `{"key1":119, "sub1":[{"key2":"value2", "sub2":{"key3":["value2"]}}],
"sub3":{"key4":"value2", "key5":"value2"}}`
	testYAML1 = `
key1: 119
sub1:
  key2: value2
  sub2:
    key3: value3
sub3:
  key4: value4
  key5: value5
`
	testYAML2 = `
key1: 119
sub1:
  - key2: value2
    sub2:
      key3: [value3]
sub3:
  key4: value4
  key5: value5
`
)
