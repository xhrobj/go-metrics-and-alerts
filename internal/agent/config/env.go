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
