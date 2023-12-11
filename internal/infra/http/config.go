package server

type Config struct {
	Port              string
	disableMiddleware bool
}

func NewConfig(
	port string,
	disableMiddleware bool,
) Config {
	return Config{
		Port:              port,
		disableMiddleware: false,
	}
}
