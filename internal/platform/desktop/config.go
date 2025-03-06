package desktop

type Config struct {
	MaxThreads  int
	UseGPU      bool
	WorkingDir  string
	SaveResults bool
	ResultsPath string
}

func NewDefaultConfig() *Config {
	return &Config{
		MaxThreads:  4,
		UseGPU:      true,
		WorkingDir:  "./work",
		SaveResults: true,
		ResultsPath: "./results",
	}
}
