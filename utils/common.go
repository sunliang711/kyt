package utils

func UniqStringSlice(s []string) []string {
	var ret []string
	key := make(map[string]bool)

	for _, v := range s {
		if _, ok := key[v]; !ok {
			key[v] = true
			ret = append(ret, v)
		}
	}

	return ret
}
