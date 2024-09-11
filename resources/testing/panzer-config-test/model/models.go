package model

type SpacePolicies struct {
	Targets []struct {
		Apps     []string `yaml:"apps"`
		Port     int      `yaml:"port"`
		Protocol string   `yaml:"protocol"`
		From     []struct {
			Org   string   `yaml:"org"`
			Space string   `yaml:"space"`
			Apps  []string `yaml:"apps"`
		} `yaml:"from"`
	} `yaml:"targets"`
}
