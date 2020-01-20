# vegeta-powerup

```go get github.com/dgparker/vegeta-powerup```

Charge up Vegeta load tests using postman

## About
vegeta-powerup enhances the [Vegeta](https://github.com/tsenart/vegeta) load testing tool and library by allowing you to import your Postman collections and environments for easy targeting.

vegeta-powerup parses your collections and environments (optional) and turns them in to vegeta targets. It also gives you the ability to generate random values at the time of attack, which is often times necessary for load testing. 

You can add ```{{VEGETA_...}}``` env references to your postman collection that will enable you to generate random values at the time of attack.

### Vegeta load testing
You can read and check the documentation for the vegeta load testing tool and library here
- **[Vegeta](https://github.com/tsenart/vegeta)**

## Example usage
```
func main() {
	targets, err := powerup.Absorb("collectionPath", "environmentPath", nil)
	if err != nil {
		log.Fatal(err)
	}

	rate := vegeta.Rate{
		Freq: 9001,
		Per:  time.Second,
	}

	duration := 30 * time.Second
	targeter := powerup.NewPostmanTargeter(&randomizer{}, targets...)
	attacker := vegeta.NewAttacker()

	var metrics vegeta.Metrics
	for res := range attacker.Attack(targeter, rate, duration, "test") {
		metrics.Add(res)
	}
	metrics.Close()

	fmt.Printf("99th percentile: %s\n", metrics.Latencies.P99)
}

type randomizer struct{}

func (r *randomizer) Random(v string) string {
	switch v {
	case "{{VEGETA_RANDOMEMAIL}}":
		// randomize logic...
	case "{{VEGETA_RANDOMPHONE}}":
		// randomize logic...
	case "{{VEGETA_RANDOMNAME}}":
		// randomize logic...
	}
	...
}
```
