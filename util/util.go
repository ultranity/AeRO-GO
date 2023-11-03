package util

import "strings"

func SplitX(s, sep string) []string {
	if len(s) == 0 {
		return []string{}
	}
	return strings.Split(s, sep)
}

func JoinX(elems []string, sep string) string {
	if len(elems) == 0 {
		return ""
	} else if len(elems) == 1 {
		return elems[0]
	}
	return strings.Join(elems, sep)
}

func Map[T interface{}, R interface{}](elements []T, fn func(T) R) (result []R) {
	for _, e := range elements {
		result = append(result, fn(e))
	}
	return
}

func MapIndexed[T interface{}, R interface{}](elements []T, fn func(int, T) R) (result []R) {
	for i, e := range elements {
		result = append(result, fn(i, e))
	}
	return
}

func MapFiltered[T interface{}, R interface{}](elements []T, fn func(T) (R, bool)) (result []R) {
	for _, e := range elements {
		if r, ok := fn(e); ok {
			result = append(result, r)
		}
	}
	return
}
func Filter[T interface{}](elements []T, fn func(T) bool) (result []T) {
	for _, e := range elements {
		if fn(e) {
			result = append(result, e)
		}
	}
	return
}

func DeDuplicate[T comparable](sliceList []T) []T {
	allKeys := make(map[T]bool)
	list := []T{}
	for _, item := range sliceList {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func CheckLegal(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') &&
			(r < '0' || r > '9') && (r != '-') && (r != '_') {
			return false
		}
	}
	return true
}
