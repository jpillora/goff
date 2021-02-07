package ff

import (
	"regexp"
	"strconv"
	"strings"
)

var timeRe = regexp.MustCompile(`time=(\d\d):(\d\d):(\d\d)\.(\d\d)`)

var nonAlpha = regexp.MustCompile(`[^\s\w\-]`)

func mustInt(s string) (i int) {
	i, _ = strconv.Atoi(s)
	return
}

func getAudioExt(path string) string {
	for _, ext := range []string{".mp3", ".m4a", ".m4b"} {
		if strings.HasSuffix(path, ext) {
			return ext
		}
	}
	return ""
}

//string set
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
