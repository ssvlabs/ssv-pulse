package configs

var Values Config

type Config struct {
	Benchmark Benchmark `mapstructure:"benchmark"`
	Analyzer  Analyzer  `mapstructure:"analyzer"`
}
