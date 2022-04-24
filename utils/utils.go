package utils

import "strings"

func SliceRemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func SliceExcludeSlice(src []string, exclude []string) (result []string) {
	for _, item := range exclude {
		if SliceContainsString(src, item) {
			src = SliceRemoveString(src, item)
		}
	}
	result = src
	return
}

func SliceContainsString(slice []string, s string) bool {
	if slice == nil {
		return false
	}
	for _, item := range slice {
		if strings.Replace(item, "\n", "", -1) == s {
			return true
		}
	}
	return false
}
