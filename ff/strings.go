package ff

import (
	"regexp"
	"strconv"
	"strings"
)

var timeRe = regexp.MustCompile(`time=(\d\d):(\d\d):(\d\d)\.(\d\d)`)

var nonAlpha = regexp.MustCompile(`[^\s\w\-]`)

var nums = regexp.MustCompile(`(\d+)`)

func mustInt(s string) (i int) {
	i, _ = strconv.Atoi(s)
	return
}

func getAudioExt(path string) string {
	for _, ext := range []string{".mp3", ".m4a", ".m4b", ".opus"} {
		if strings.HasSuffix(path, ext) {
			return ext
		}
	}
	return ""
}

func padNums(s string) string {
	// foo14bar -> foo0000000014bar
	// foo14 -> foo0000000014
	// foo -> foo
	// 14 -> 0000000014
	// 14bar -> 0000000014bar
	// 14bar15 -> 0000000014bar0000000015
	// 14bar15baz -> 0000000014bar0000000015baz
	return nums.ReplaceAllStringFunc(s, func(s string) string {
		sb := strings.Builder{}
		for i := 0; i < 9-len(s); i++ {
			sb.WriteRune('0')
		}
		return sb.String() + s
	})
}

// string set
type strSet struct {
	strings map[string]int
}

func (set *strSet) Add(s string) {
	if set.strings == nil {
		set.strings = map[string]int{}
	}
	set.strings[s] = set.strings[s] + 1
}

func (set *strSet) Get() string {
	n := -1
	s := ""
	for k, v := range set.strings {
		if v > n {
			n = v
			s = k
		}
	}
	return s
}
