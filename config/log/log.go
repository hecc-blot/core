package config

type LocalConfig struct {
	RootDir    string `mapstructure:"root_dir"`
	ShowLine   bool   `mapstructure:"show_line"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}
