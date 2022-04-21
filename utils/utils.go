package utils

func SliceRemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func SliceContainsString(slice []string, s string) bool {
	if slice == nil {
		return false
	}
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
