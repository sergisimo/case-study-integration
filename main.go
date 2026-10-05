// discharge-bridge pulls discharge records from a hospital's legacy HIS
// export, maps them into the shape Recare expects, and forwards them on.
//
// NOTE: this is a first cut that's been running in prod for a while.
// It works, mostly. Take a look and see what you'd change before we
// scale it to more hospitals.
package main

import (
	"context"
	"discharge-bridge/discharge"
	"discharge-bridge/his"
	"log"
	"os"
	"strings"
)

// ---- config -----------------------------------------------------------

type Config struct {
	HISEndpoint    string
	RecareEndpoint string
	HISAPIKey      string // TODO: use OAUTH2.0 client credentials flow instead of static API key
	RecareAPIKey   string
}

// TODO: secret is never decrypted so it won't work when doing the call
func loadConfig(path string) Config {
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("could not read config: %v", err)
	}

	// TODO: load this from env vars instead of a config file, easier for devops implementations using Docker and K8s.
	values := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		values[key] = val
	}

	c := Config{
		HISEndpoint:    values["his_endpoint"],
		RecareEndpoint: values["recare_endpoint"],
		HISAPIKey:      values["his_api_key"],
		RecareAPIKey:   values["recare_api_key"],
	}

	// easier to debug config problems if we can see what got loaded
	log.Printf("loaded config: %+v", c) // TODO: do not print secrets

	return c
}

// ---- main ------------------------------------------------------------------

func main() {
	ctx := context.Background()
	cfg := loadConfig("config.yaml")
	recordFetcher := his.NewFetcher(cfg.HISEndpoint, cfg.HISAPIKey)
	dischargeClient := discharge.NewClient(cfg.RecareEndpoint, cfg.RecareAPIKey)

	summaries, err := recordFetcher.Fetch(ctx)
	if err != nil {
		log.Fatalf("failed to fetch records: %v", err)
	}
	log.Printf("fetched %d records from HIS", len(summaries))

	recordsProcessed := 0
	recordsFailed := 0
	resultCh := make(chan error)

	for _, sum := range summaries {
		go func(sum discharge.Summary) {
			err := dischargeClient.Send(ctx, sum)
			resultCh <- err
		}(sum)
	}

	for range summaries {
		err := <-resultCh
		if err != nil {
			recordsFailed++
		} else {
			recordsProcessed++
		}
	}

	log.Printf("done: %d sent, %d failed", recordsProcessed, recordsFailed)
}
