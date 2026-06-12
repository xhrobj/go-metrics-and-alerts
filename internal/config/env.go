package config

import (
	"os"
	"strconv"
)

func getEnvInt(name string) (int, bool, error) {
	if v, ok := os.LookupEnv(name); ok {
		i, err := strconv.Atoi(v)
		if err != nil {
			return 0, false, err
		}
		return i, true, nil
	}
	return 0, false, nil
}

func getEnvBool(name string) (bool, bool, error) {
	if v, ok := os.LookupEnv(name); ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return false, false, err
		}
		return b, true, nil
	}
	return false, false, nil
}
