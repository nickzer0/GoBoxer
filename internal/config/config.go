package config

import (
	"github.com/alexedwards/scs/v2"
	"github.com/nickzer0/GoBoxer/internal/driver"
)

// AppConfig holds the application config
type AppConfig struct {
	DataFile      string
	InProduction  bool
	AnsibleDebug  string
	DB            *driver.DB
	Session       *scs.SessionManager
	Domain        string
	PreferenceMap map[string]string
	Version       string
}
