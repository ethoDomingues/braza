package braza

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
)

var (
	ErrHttpAbort        = errors.New("aborted")
	ErrorNotFound       = errors.New("404 Not Found")
	ErrorMethodMismatch = errors.New("405 Method Not Allowed")
)

type MatchInfo struct {
	Func
	Match            bool
	Route            *Route
	Router           *Router
	MethodNotAllowed error
}

type _re struct {
	str   *regexp.Regexp
	all   *regexp.Regexp
	digit *regexp.Regexp

	isVar    *regexp.Regexp
	isVarOpt *regexp.Regexp
	dot2     *regexp.Regexp
	slash2   *regexp.Regexp
}

var (
	reMethods = regexp.MustCompile("^(?i)(GET|PUT|HEAD|POST|TRACE|PATCH|DELETE|CONNECT|OPTIONS)$")

	re = _re{
		str:    regexp.MustCompile(`{\w+(:str)?}`),
		all:    regexp.MustCompile(`(\{\*\})|(\{\w+:path\})`),
		isVar:  regexp.MustCompile(`{\w+(\:(int|str|path))?\}|\{\*\}`),
		digit:  regexp.MustCompile(`{\w+:int}`),
		dot2:   regexp.MustCompile(`[.]{2,}`),
		slash2: regexp.MustCompile(`[\/]{2,}`),
	}
)

// if 'str' is a var (example: {id:int} ), return 'id', else return str
func (r *_re) getVarName(str string) string {
	if r.isVar.MatchString(str) {
		str = strings.Replace(str, "{", "", -1)
		str = strings.Replace(str, "}", "", -1)
		str = strings.Split(str, ":")[0]
	}
	if strings.HasSuffix(str, "?") {
		return strings.TrimSuffix(str, "?")
	}
	return str
}

func (r _re) getUrlValues(url, requestUrl string) map[string]string {
	req := strings.Split(requestUrl, "/")
	kv := map[string]string{}
	for i, str := range strings.Split(url, "/") {
		if i < len(req) {
			if re.isVar.MatchString(str) {
				varName := re.getVarName(str)
				if varName == str {
					continue
				} else if re.all.MatchString(str) {
					strs := bytes.NewBufferString("")
					for c := i; c < len(req); c++ {
						strs.WriteString("/" + req[c])
					}
					if strings.HasSuffix(requestUrl, "/") {
						strs.WriteString("/")
					}
					kv[varName] = strs.String()
					continue
				} else {
					kv[varName] = req[i]
				}
			}
		}
	}
	return kv
}
