package config

import "strconv"

type lookupEnvFunc func(string) (string, bool)

func getEnvInt(lookupEnv lookupEnvFunc, name string) (int, bool, error) {
	if v, ok := lookupEnv(name); ok {
		i, err := strconv.Atoi(v)
		if err != nil {
			return 0, false, err
		}
		return i, true, nil
	}
	return 0, false, nil
}

func getEnvBool(lookupEnv lookupEnvFunc, name string) (bool, bool, error) {
	if v, ok := lookupEnv(name); ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return false, false, err
		}
		return b, true, nil
	}
	return false, false, nil
}
