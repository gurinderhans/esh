package main

func IndexOf(item string, inArr []string) int {
	for i, v := range inArr {
		if v == item {
			return i
		}
	}
	return -1
}
