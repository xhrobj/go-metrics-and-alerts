package config

import (
	"fmt"
	"strconv"
)

type lookupEnvFunc func(string) (string, bool)

func getEnvInt(lookupEnv lookupEnvFunc, name string) (int, bool, error) {
	if v, ok := lookupEnv(name); ok {
		i, err := strconv.Atoi(v)
		if err != nil {
			return 0, false, fmt.Errorf(
				"parse environment variable %s=%q: %w",
				name,
				v,
				err,
			)
		}
		return i, true, nil
	}
	return 0, false, nil
}

func getEnvBool(lookupEnv lookupEnvFunc, name string) (bool, bool, error) {
	if v, ok := lookupEnv(name); ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return false, false, fmt.Errorf(
				"parse environment variable %s=%q: %w",
				name,
				v,
				err,
			)
		}
		return b, true, nil
	}
	return false, false, nil
}
