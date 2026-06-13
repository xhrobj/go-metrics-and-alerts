package config

func testLookupEnv(values map[string]string) lookupEnvFunc {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}
