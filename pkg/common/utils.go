package common

func FindFirstIndex[t comparable](slice []t, item t) int {
	for index, value := range slice {
		if value == item {
			return index
		}
	}

	return -1
}
