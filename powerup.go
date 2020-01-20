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

// Randomizer - implementations should expect the {{VEGETA_...}} env reference
// as the arguement to Random(). Random will only be called on attacks the contain a reference
// of {{VEGETA_...}} you can use this value to determine which random values to generate.
type Randomizer interface {
	Random(string) string
}

// NewPostmanTargeter returns a vegeta.Targeter which round-robins over the passed
// Targets
// for postman - if the collection target contains a reference to var {{VEGETA_}} regex ("{{VEGETA_(.*?)}}")
// the var will be replaced with a randomized value using the consumers Randomizer implementation
// implmentations that don't require the use of a randomizer should implement a Randomizer that returns
// the original string argument.
func NewPostmanTargeter(randomizer Randomizer, tgts ...vegeta.Target) vegeta.Targeter {
	i := int64(-1)
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}
		*tgt = tgts[atomic.AddInt64(&i, 1)%int64(len(tgts))]

		// check for randmozier in url
		tgt.URL = randomizeString(tgt.URL, randomizer)
		// check for randomizer in body
		tgt.Body = randomizeBytes(tgt.Body, randomizer)
		// check for randomizer in headers
		tgt.Header = randomizeHeaders(tgt.Header, randomizer)

		return nil
	}
}

func randomizeString(in string, randomizer Randomizer) string {
	if !re3.MatchString(in) {
		return in
	}

	var out string
	keysRaw := re3.FindAllString(in, -1)
	for _, key := range keysRaw {
		out = strings.Replace(in, key, randomizer.Random(key), -1)
	}

	return out
}

func randomizeBytes(in []byte, randomizer Randomizer) []byte {
	log.Println(string(in))
	if !re3.Match(in) {
		return in
	}
	var out []byte
	keysRaw := re3.FindAll(in, -1)
	for _, key := range keysRaw {
		out = bytes.Replace(in, key, []byte(randomizer.Random(string(key))), -1)
	}

	return out
}

func randomizeHeaders(in http.Header, randomizer Randomizer) http.Header {
	if len(in) == 0 {
		return in
	}

	out := http.Header{}
	for k, v := range in {
		var headerKey, headerValue string
		if re3.MatchString(k) {
			keysRaw := re3.FindAllString(k, -1)
			for _, key := range keysRaw {
				headerKey = strings.Replace(k, key, randomizer.Random(key), -1)
			}
		}

		if len(v) > 0 && re3.MatchString(v[0]) {
			keysRaw := re3.FindAllString(v[0], -1)
			for _, key := range keysRaw {
				headerValue = strings.Replace(v[0], key, randomizer.Random(key), -1)
			}
		}
		out.Add(headerKey, headerValue)
	}

	return out
}

// Absorb parses the postman collection and environment and returns a slice of vegeta Targets.
// A collection path is required, an environment path is optional. logger is generally for debugging purposes only
// and passing nil acceptable.
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
			result := ki.envMap[re2.ReplaceAllString(key, "")]
			if result == "" {
				continue
			}
			value = strings.Replace(value, key, result, -1)
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
			result := ki.envMap[string(re2.ReplaceAll(key, nil))]
			if result == "" {
				continue
			}
			body = bytes.Replace(body, key, []byte(result), -1)
		}
	}
	return body
}
