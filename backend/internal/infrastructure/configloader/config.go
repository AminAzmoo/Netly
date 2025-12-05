package configloader

type ConfigLoader struct{}

func New() *ConfigLoader {
	return &ConfigLoader{}
}

func (l *ConfigLoader) Load() error {
	return nil
}
