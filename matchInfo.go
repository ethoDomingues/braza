package braza

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrHttpAbort        = errors.New("abort")
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

	dot2           *regexp.Regexp
	isVar          *regexp.Regexp
	slash2         *regexp.Regexp
	isVarOpt       *regexp.Regexp
	httpPort       *regexp.Regexp
	dynamicSubVars *regexp.Regexp
}

var (
	reMethods = regexp.MustCompile("^(?i)(GET|PUT|HEAD|POST|TRACE|PATCH|DELETE|CONNECT|OPTIONS)$")
	re        = _re{
		str:            regexp.MustCompile(`{\w+(:str)?}`),
		all:            regexp.MustCompile(`(\{(\w+:)?\*\})|(\{\w+:path\})`),
		dot2:           regexp.MustCompile(`[.]{2,}`),
		digit:          regexp.MustCompile(`{\w+:int}`),
		isVar:          regexp.MustCompile(`{\w+(\:(int|str|path|[*]))?\}|\{\*\}`),
		slash2:         regexp.MustCompile(`[\/]{2,}`),
		httpPort:       regexp.MustCompile(`^([:]?[\d]{1,})$`),
		dynamicSubVars: regexp.MustCompile(`^(\{\w+(\:(int|str))?\})$`),
	}
)

// if 'str' is a var, example: {id:int} -> return 'id', else return str.
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

/*
example:

	url := "/user/{id:int}/"
	requestUrl := "/user/123"
	_re.getUrlValues -> return map[string]string{"id":"123"}
*/
func (r _re) getUrlValues(url, urlReq string) map[string]string {
	if !r.isVar.MatchString(url) {
		return map[string]string{}
	}
	req := strings.Split(urlReq, "/")
	kv := map[string]string{}
	urlSplit := strings.Split(url, "/")
	rqSplit := strings.Split(urlReq, "/")
	if len(urlSplit) != len(rqSplit) {
		panic(fmt.Errorf("url unmatch with requestUrl. url: '%s' -- urlReq: '%s'", url, urlReq))
	}
	for i, str := range urlSplit {
		if re.isVar.MatchString(str) {
			varName := re.getVarName(str)
			if re.all.MatchString(str) { // if /{filepath:*} || /{filepath:path}
				strs := bytes.NewBufferString("")
				for c := i; c < len(req); c++ {
					strs.WriteString("/" + req[c])
				}
				if strings.HasSuffix(urlReq, "/") {
					strs.WriteString("/")
				}
				kv[varName] = strs.String()
				continue
			} else { // else /{var:int||str}
				kv[varName] = req[i]
			}
		}
	}
	return kv
}

func (r _re) getSubdomainValues(subdomain, requestHost string) map[string]string {
	kv := map[string]string{}
	sub := strings.Split(requestHost, ".")
	for i, str := range strings.Split(subdomain, ".") {
		if re.isVar.MatchString(str) {
			varName := re.getVarName(str)
			if varName == str {
				continue
			} else {
				kv[varName] = sub[i]
			}
		}
	}
	return kv
}
