package powerup

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dgparker/vegeta-powerup/postman"
	vegeta "github.com/tsenart/vegeta/lib"
)

var (
	re1   = regexp.MustCompile(`{{(.*?)}}`)
	re2   = regexp.MustCompile(`[{}]`)
	re3   = regexp.MustCompile(`{{VEGETA_(.*?)}}`)
	seed  = rand.NewSource(time.Now().UnixNano())
	rando = rand.New(seed)
)

// Ki contains the methods necessary for maniupulating a postman
// collection into a vegeta target
type Ki struct {
	logger *log.Logger
	coll   *postman.Collection
	env    *postman.Environment
	envMap map[string]string
}

// Randomizer defines an interface that takes a string as a parameter
// then returns a randomized value for the string.
// URL value can be used to generate endpoint specific randomizations
type Randomizer interface {
	Random(string) string
}

// NewPostmanTargeter returns a vegeta.Targeter which round-robins over the passed
// Targets
// POSTMAN OPTIONS -
// for postman collections POST and PUT methods if the collection body contains
// a reference to var {{VEGETA_}} regex ("{{VEGETA_}}") the var will be
// replaced with a randomized value using the consumers Randomizer implementation
func NewPostmanTargeter(randomizer Randomizer, tgts ...vegeta.Target) vegeta.Targeter {
	i := int64(-1)
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}
		*tgt = tgts[atomic.AddInt64(&i, 1)%int64(len(tgts))]

		if re3.MatchString(tgt.URL) {
			keysRaw := re1.FindAllString(tgt.URL, -1)
			for _, key := range keysRaw {
				tgt.URL = strings.Replace(tgt.URL, key, randomizer.Random(key), -1)
			}
		}

		// if tgt.Method == "POST" || tgt.Method == "PUT" {
		// 	unique := rando.Int()
		// 	if len(tgt.Body) == 0 {
		// 		return nil
		// 	}

		// 	tgt.Body = []byte(re3.ReplaceAllString(string(tgt.Body), strconv.Itoa(unique)))
		// }

		// probably needs to go here
		// if re3.MatchString(string(key)) {
		// 	body = bytes.Replace(body, key, []byte(ki.Randomizer.Random(string(key))), -1)
		// 	continue
		// }
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

	targets := ki.getTargets(ki.coll.Items)

	ki.logger.Printf("transformation completed in: %s\n", time.Since(startTime).String())
	return targets, nil
}

func (ki *Ki) getTargets(items []postman.Item) []vegeta.Target {
	targets := []vegeta.Target{}
	for _, v := range items {
		if v.Items != nil || len(v.Items) != 0 {
			targets = append(targets, ki.getTargets(v.Items)...)
			continue
		}

		tgt := vegeta.Target{}

		tgt.Method = v.Request.Method
		tgt.URL = ki.parseSegment(v.Request.URL.Raw)
		tgt.Header = ki.parseHeader(v.Request.WrapHeaders())
		tgt.Body = ki.parseBody(v.Request.Body.Bytes())
		tgt = ki.parseAuth(tgt, v.Request.Auth)

		targets = append(targets, tgt)

	}
	return targets
}

func (ki *Ki) parseSegment(value string) string {
	if re1.MatchString(value) {
		keysRaw := re1.FindAllString(value, -1)
		for _, key := range keysRaw {
			value = strings.Replace(value, key, ki.envMap[re2.ReplaceAllString(key, "")], -1)
		}
	}
	return value
}

func (ki *Ki) parseHeader(h http.Header) http.Header {
	headers := http.Header{}
	for hk, hvs := range h {
		values := []string{}
		for _, hv := range hvs {
			values = append(values, ki.parseSegment(hv))
		}
		headers[hk] = values
	}
	return headers
}

func (ki *Ki) parseAuth(tgt vegeta.Target, auth postman.Auth) vegeta.Target {
	switch strings.ToLower(auth.Type) {
	case "apikey":
		apiKeyMap := map[string]string{}

		for _, v := range auth.APIKey {
			apiKeyMap[v.Key] = v.Value
		}

		switch apiKeyMap["in"] {
		case "header":
			tgt.Header.Add(apiKeyMap["key"], apiKeyMap["value"])
		case "query":
			tURL, err := url.Parse(tgt.URL)
			if err != nil {
				ki.logger.Println(err)
			}
			query := tURL.Query()
			query.Add(apiKeyMap["key"], apiKeyMap["value"])
			tURL.RawQuery = query.Encode()
			tgt.URL = tURL.String()
		default:
			ki.logger.Fatal("invalid api key format")
		}

	case "basic":
		basicAuthMap := map[string]string{}

		for _, v := range auth.Basic {
			basicAuthMap[v.Key] = v.Value
		}

		username := ki.parseSegment(basicAuthMap["username"])
		password := ki.parseSegment(basicAuthMap["password"])
		basicAuth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))

		tgt.Header.Add("Authorization", fmt.Sprintf("Basic %s", basicAuth))
	case "bearer":
		for _, v := range auth.Bearer {
			if v.Key == "token" {
				tgt.Header.Add("Authorization", fmt.Sprintf("Bearer %s", v.Value))
			}
		}
	}

	return tgt
}

func (ki *Ki) parseBody(body []byte) []byte {
	if re1.Match(body) {
		keysRaw := re1.FindAll(body, -1)
		for _, key := range keysRaw {
			body = bytes.Replace(body, key, []byte(ki.envMap[string(re2.ReplaceAll(key, nil))]), -1)
		}
	}
	return body
}
