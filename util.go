package main

func contains(slice []string, member string) bool {
	for _, s := range slice {
		if s == member {
			return true
		}
	}
	return false
}

func indexOf(slice []string, member string) int {
	for i, s := range slice {
		if s == member {
			return i
		}
	}
	return -1
}

// https://stackoverflow.com/a/37334775/
func remove(slice []string, index int) []string {
	return append(slice[:index], slice[index+1:]...)
}
