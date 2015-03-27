package main

type Config struct {
	entries map[string]string
}

func NewConfig() *Config {
	return &Config{entries: make(map[string]string, 10)}
}

func (c *Config) SetMountPoint(dir string) {
	c.entries["mount"] = dir
}

func (c *Config) GetMountPoint() string {
	return c.entries["mount"]
}

func (c *Config) SetShadowDir(dir string) {
	c.entries["shadow"] = dir
}

func (c *Config) GetShadowDir() string {
	return c.entries["shadow"]
}

func (c *Config) SetOutputFormat(format string) {
	c.entries["format"] = format
}

func (c *Config) GetOutputFormat() string {
	return c.entries["format"]
}

func (c *Config) SetTraceDestination(fileName string) {
	c.entries["destination"] = fileName
}

func (c *Config) GetTraceDestination() string {
	return c.entries["destination"]
}

func (c *Config) SetReadOnly(readonly bool) {
	s := "false"
	if readonly {
		s = "true"
	}
	c.entries["readonly"] = s
}

func (c *Config) GetReadOnly() bool {
	if c.entries["readonly"] == "true" {
		return true
	}
	return false
}
