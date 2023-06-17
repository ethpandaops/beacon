package beacon

// Config is the configuration for a beacon node.
type Config struct {
	// Name is the human-readable name of the node.
	Name string `yaml:"name"`
	// Address is the address of the node.
	Addr string `yaml:"addr"`
	// Headers is the list of HTTP headers sent to the node.
	Headers map[string]string `yaml:"headers"`
}
