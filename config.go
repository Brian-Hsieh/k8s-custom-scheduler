package main

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	latency_weight float64
	load_weight    float64
	omega          float64
	c1             float64
	c2             float64
}

var config Config

func loadConfig() error {
	config = Config{}

	parseEnvVar := func(name string, defaultValue float64) (float64, error) {
		envVal := os.Getenv(name)
		if envVal == "" {
			return defaultValue, nil
		}
		parsedVal, err := strconv.ParseFloat(envVal, 64)
		if err != nil {
			return defaultValue, fmt.Errorf("Error parsing %s: %v. Returning default value: %f\n", name, err, defaultValue)
		}
		return parsedVal, nil
	}

	var err error
	config.latency_weight, err = parseEnvVar("latency_weight", 1.0)
	if err != nil {
		return err
	}

	config.load_weight, err = parseEnvVar("load_weight", 5.0)
	if err != nil {
		return err
	}

	config.omega, err = parseEnvVar("omega", 1.0)
	if err != nil {
		return err
	}

	config.c1, err = parseEnvVar("c1", 0.5)
	if err != nil {
		return err
	}

	config.c2, err = parseEnvVar("c2", 0.5)
	if err != nil {
		return err
	}

	return nil
}
