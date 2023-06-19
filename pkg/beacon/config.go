package beacon

// Config is the configuration for a beacon node.
type Config struct {
	// Name is the human-readable name of the node.
	Name string `yaml:"name"`
	// Address is the address of the node.
	Addr string `yaml:"addr"`
	// Headers are the headers to send with every request.
	Headers map[string]string `yaml:"headers"`
}
