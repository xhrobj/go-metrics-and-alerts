package config

import (
	"fmt"
	"strconv"
)

type lookupEnvFunc func(string) (string, bool)

func getEnvInt(lookupEnv lookupEnvFunc, name string) (int, bool, error) {
	if value, ok := lookupEnv(name); ok {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return 0, false, fmt.Errorf(
				"parse environment variable %s=%q: %w",
				name,
				value,
				err,
			)
		}

		return parsed, true, nil
	}

	return 0, false, nil
}

func getEnvBool(lookupEnv lookupEnvFunc, name string) (bool, bool, error) {
	if value, ok := lookupEnv(name); ok {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return false, false, fmt.Errorf(
				"parse environment variable %s=%q: %w",
				name,
				value,
				err,
			)
		}

		return parsed, true, nil
	}

	return false, false, nil
}
