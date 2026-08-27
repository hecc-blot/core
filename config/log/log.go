package config

type LocalConfig struct {
	Enable     bool   `mapstructure:"enable"`
	RootDir    string `mapstructure:"root_dir"`
	ShowLine   bool   `mapstructure:"show_line"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

type SlsConfig struct {
	Enable      bool   `mapstructure:"enable"`
	Endpoint    string `mapstructure:"endpoint"`
	AccessKey   string `mapstructure:"access_key"`
	SecretKey   string `mapstructure:"secret_key"`
	SecretToken string `mapstructure:"secret_token"`
	Project     string `mapstructure:"project"`
	LogStore    string `mapstructure:"log_store"`
}
