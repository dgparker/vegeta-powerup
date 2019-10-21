package powerup

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dgparker/vegeta-powerup/postman"
	vegeta "github.com/tsenart/vegeta/lib"
)

var (
	// ENVLoaded indicates whether a postman env has been loaded
	ENVLoaded = false
	re1       = regexp.MustCompile(`{{(.*?)}}`)
	re2       = regexp.MustCompile(`[{}]`)
	re3       = regexp.MustCompile(`{{VEGETA}}`)
	seed      = rand.NewSource(time.Now().UnixNano())
	rando     = rand.New(seed)
)

// Ki contains the methods necessary for maniupulating a postman
// collection into a vegeta target
type Ki struct {
	logger *log.Logger
	coll   *postman.Collection
	env    *postman.Environment
	envMap map[string]string
}

// Randomizer defines a function that takes url as a parameter
// then returns a randomized value for the url.
// URL value can be used to generate endpoint specific randomizations
type Randomizer func(url string) string

// NewPostmanTargeter returns a vegeta.Targeter which round-robins over the passed
// Targets
// POSTMAN OPTIONS -
// for postman collections POST and PUT methods if the collection body contains
// a reference to var {{VEGETA}} regex ("{{VEGETA}}") the var will be
// replaced with a randomized value
func NewPostmanTargeter(tgts ...vegeta.Target) vegeta.Targeter {
	i := int64(-1)
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}
		*tgt = tgts[atomic.AddInt64(&i, 1)%int64(len(tgts))]

		if tgt.Method == "POST" || tgt.Method == "PUT" {
			unique := rando.Int()
			if len(tgt.Body) == 0 {
				return nil
			}
			tgt.Body = []byte(re3.ReplaceAllString(string(tgt.Body), strconv.Itoa(unique)))
		}
		return nil
	}
}

// Absorb loads the postman collection for later processing
func Absorb(collPath string, envPath string, logger *log.Logger) ([]vegeta.Target, error) {
	if logger == nil {
		logger = log.New(os.Stdout, "vegeta-postman: ", log.LstdFlags)
	}

	collFile, err := ioutil.ReadFile(collPath)
	if err != nil {
		return nil, err
	}

	coll := &postman.Collection{}
	err = json.Unmarshal(collFile, coll)
	if err != nil {
		return nil, err
	}

	env := &postman.Environment{}
	envMap := map[string]string{}
	if envPath != "" {
		ENVLoaded = true
		envFile, err := ioutil.ReadFile(envPath)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(envFile, env)
		if err != nil {
			return nil, err
		}
		envMap = env.Map()
	}

	ki := &Ki{
		logger: logger,
		coll:   coll,
		env:    env,
		envMap: envMap,
	}

	return ki.transform()
}

// Transform creates targets for vegeta based on the loaded
// postman swagger
func (ki *Ki) transform() ([]vegeta.Target, error) {
	startTime := time.Now()
	ki.logger.Println("beginning transformation...")

	targets := getTargets(ki.coll.Items)

	if ENVLoaded {
		targets = ki.setEnvironment(targets)
	}

	ki.logger.Printf("transformation completed in: %s\n", time.Since(startTime).String())
	return targets, nil
}

func getTargets(items []postman.CollectionItem) []vegeta.Target {
	targets := []vegeta.Target{}
	for _, v := range items {
		if v.Items != nil {
			targets = append(targets, getTargets(v.Items)...)
		} else {
			tgt := vegeta.Target{}

			tgt.Method = v.Request.Method
			tgt.URL = v.Request.URL.Raw
			tgt.Header = v.Request.WrapHeaders()
			tgt.Body = v.Request.Body.Bytes()

			targets = append(targets, tgt)
		}
	}
	return targets
}

func (ki *Ki) setEnvironment(targets []vegeta.Target) []vegeta.Target {
	newTargets := []vegeta.Target{}
	for _, target := range targets {
		target.URL = ki.replaceURL(target.URL)
		target.Header = ki.replaceHeader(target.Header)
		target.Body = ki.replaceBody(target.Body)
		newTargets = append(newTargets, target)
	}
	return newTargets
}

func (ki *Ki) replaceURL(url string) string {
	if re1.MatchString(url) {
		keysRaw := re1.FindAllString(url, -1)
		for _, key := range keysRaw {
			url = strings.Replace(url, key, ki.envMap[re2.ReplaceAllString(key, "")], -1)
		}
	}
	return url
}

func (ki *Ki) replaceHeader(h http.Header) http.Header {
	headers := http.Header{}
	for hk, hvs := range h {
		values := []string{}
		for _, hv := range hvs {
			keysRaw := re1.FindAllString(hv, -1)
			if len(keysRaw) == 0 {
				values = append(values, hv)
			} else {
				for _, key := range keysRaw {
					newValue := strings.Replace(hv, key, ki.envMap[re2.ReplaceAllString(key, "")], -1)
					values = append(values, newValue)
				}
			}
		}
		headers[hk] = values
	}
	return headers
}

func (ki *Ki) replaceBody(body []byte) []byte {
	if re1.Match(body) {
		keysRaw := re1.FindAll(body, -1)
		for _, key := range keysRaw {
			body = bytes.Replace(body, key, []byte(ki.envMap[string(re2.ReplaceAll(key, nil))]), -1)
		}
	}
	return body
}
