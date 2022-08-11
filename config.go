package beacon

type Config struct {
	// Name is the human-readable name of the node.
	Name string `yaml:"name"`
	// Address is the address of the node.
	Addr string `yaml:"addr"`
}
