package main

import (
	"encoding/json"
	"flag"
	"io"
	"log/slog"
	"os"
	"slices"

	"github.com/goccy/go-yaml"
)

var (
	singleQuote = flag.Bool("single-quote", false, "Single or double quotes for strings.")

	dataKeys = []string{
		"STADIUM_DEVICE_ENDPOINT",
		"STADIUM_DEVICE_API_TOKEN",
		"STADIUM_DEVICE_USERNAME",
		"STADIUM_DEVICE_PASSWORD",
		"STADIUM_DEVICE_API_KEY",
		"STADIUM_DEVICE_ACCOUNT_ID",
		"STADIUM_DEVICE_ID",
		"STADIUM_DEVICE_TYPE",
	}
)

type secret struct {
	Name    string `json:"name"`
	KVPairs string `json:"kvpairs"`
}

func main() {
	flag.Parse()

	var output struct {
		Secrets []secret `json:"secrets"`
	}

	for _, path := range flag.Args() {
		log := slog.With("file", path)

		file, err := os.Open(path)
		if err != nil {
			log.Error("Cannot open file", "err", err.Error())
			return
		}

		targetYAML, err := io.ReadAll(file)
		if err != nil {
			log.Error("Cannot read file", "err", err.Error())
			return
		}

		var target map[string]any
		if err := yaml.Unmarshal(targetYAML, &target); err != nil {
			log.Error("Cannot YAML decode file", "err", err.Error())
			return
		}

		fullnameOverride, ok := target["fullnameOverride"].(string)
		if !ok {
			continue
		}

		kvpairs := map[string]any{}

		for k, v := range target {
			if k != "configmap" && k != "secret" {
				continue
			}

			var data map[string]any
			if m, _ := v.(map[string]any); m != nil {
				if data, _ = m["data"].(map[string]any); data == nil {
					continue
				}
			}

			for dataKey, dataVal := range data {
				if slices.Contains(dataKeys, dataKey) {
					kvpairs[dataKey] = dataVal
				}
			}
		}

		kvpairsJSON, err := json.Marshal(kvpairs)
		if err != nil {
			log.Error("Cannot JSON encode kvpairs", "err", err.Error())
			return
		}

		output.Secrets = append(output.Secrets, secret{
			Name:    fullnameOverride,
			KVPairs: string(kvpairsJSON),
		})
	}

	if err := yaml.NewEncoder(os.Stdout, yaml.UseSingleQuote(*singleQuote)).Encode(output); err != nil {
		slog.Error("Cannot YAML encode output", "err", err.Error())
	}
}
