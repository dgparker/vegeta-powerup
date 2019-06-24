# vegeta-powerup
Charge up Vegeta load tests using postman

## Vegeta load testing
- **[Vegeta](https://github.com/tsenart/vegeta)**

## Example usage
```
func main() {
	targets, err := powerup.Absorb("collPath", "envPath", nil)
	if err != nil {
		log.Fatal(err)
	}

	rate := vegeta.Rate{
		Freq: 10,
		Per:  time.Second,
	}
	duration := 4 * time.Second
	targeter := powerup.NewPostmanTargeter(targets...)
	attacker := vegeta.NewAttacker()

	var metrics vegeta.Metrics
	for res := range attacker.Attack(targeter, rate, duration, "test") {
		metrics.Add(res)
	}
	metrics.Close()

	fmt.Printf("99th percentile: %s\n", metrics.Latencies.P99)
}
```
