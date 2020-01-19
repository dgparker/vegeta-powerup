package main

import (
	"fmt"
	"log"
	"os"
	"time"

	powerup "github.com/dgparker/vegeta-powerup"
	vegeta "github.com/tsenart/vegeta/lib"
)

func main() {
	targets, err := powerup.Absorb(os.Getenv("COLLECTION_PATH"), os.Getenv("ENV_PATH"), nil)
	if err != nil {
		log.Fatal(err)
	}

	rate := vegeta.Rate{
		Freq: 500,
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
	return v
}
