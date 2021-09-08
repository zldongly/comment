package convert

import (
	"strconv"
	"strings"
)

func StringToInt64s(str string) []int64 {
	ss := strings.Split(str, ",")
	is := make([]int64, 0, len(ss))
	for _, s := range ss {
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			continue
		}
		is = append(is, i)
	}

	return is
}

func Int64sToString(is []int64) string {
	var s string

	for idx, i := range is {
		if idx != 0 {
			s += ","
		}
		s += strconv.FormatInt(i, 10)
	}

	return s
}

func Offset(pageNo, pageSize int32) int32 {
	return (pageNo - 1) * pageSize
}
