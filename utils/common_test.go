package utils

import "testing"

func TestUniqStringSlice(t *testing.T) {
	a := []string{"hello","world","hello"}

	ret := UniqStringSlice(a)
	t.Logf("%v\n",ret)

}
