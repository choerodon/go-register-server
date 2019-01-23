package utils

import (
	"fmt"
	"testing"
)

func TestConvertRecursiveMapToSingleMap(t *testing.T) {
	recursiveMap := map[string]interface{}{"spring": map[string]interface{}{"application":
	map[string]interface{}{"name": "test-service"}}, "hello": "hi", "age": 234}
	singleMap := ConvertRecursiveMapToSingleMap(recursiveMap)
	fmt.Printf("ConvertRecursiveMapToSingleMap singleMap : %v", singleMap)
	if singleMap["spring.application.name"] != "test-service" || singleMap["hello"] != "hi" || singleMap["age"] != 234 {
		t.Errorf("ConvertRecursiveMapToSingleMap error")
	}
}
