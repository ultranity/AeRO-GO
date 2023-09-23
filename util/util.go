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
	return result
}

func MapIndexed[T interface{}, R interface{}](elements []T, fn func(int, T) R) (result []R) {
	for i, e := range elements {
		result = append(result, fn(i, e))
	}
	return result
}
