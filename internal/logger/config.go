package logger

type Config struct {
	Environment string
	ServiceName string
	Level       string
}

func DefaultConfig() Config {
	return Config{
		Environment: "development",
		ServiceName: "vyavsa-bill-book-api",
		Level:       "info",
	}
}
