package duplicate

import "analyticDataCenter/analytics-data-center/internal/domain/models"

func RemoveDuplicate[T comparable](sliceList []T) (cleanList []T, duplicate []T) {
	allKeys := make(map[T]bool)
	list := []T{}
	duplicateList := []T{}
	for _, item := range sliceList {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		} else {
			allKeys[item] = false
			duplicateList = append(duplicateList, item)
		}
	}
	return list, duplicateList
}

func RemoveDuplicateColumns(columns []models.Column) (clean []models.Column, duplicates []string) {
	seen := make(map[string]bool)
	var result []models.Column
	var dups []string

	for _, col := range columns {
		key := col.Alias
		if key == "" {
			key = col.Name
		}

		if !seen[key] {
			seen[key] = true
			result = append(result, col)
		} else {
			dups = append(dups, key)
		}
	}

	return result, dups
}
