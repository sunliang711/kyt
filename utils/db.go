package utils

func MakeQuestion(n int) string {
	ret := "("
	for i := 0; i < n; i++ {
		ret += "?"
		if i < n-1 {
			ret += ","
		}
	}
	ret += ")"
	return ret
}
