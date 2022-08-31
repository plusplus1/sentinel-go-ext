package config

import (
	"log"
	"os"
	"sync"

	"gopkg.in/yaml.v2"
)

var (
	appInst = appSettings{}
)

type appSettings struct {
	sync.Once
	Settings
}

type Settings struct {

	// MySQL configuration
	MySQL struct {
		Host     string `json:"host,omitempty" yaml:"host"`
		Port     int    `json:"port,omitempty" yaml:"port"`
		User     string `json:"user,omitempty" yaml:"user"`
		Password string `json:"password,omitempty" yaml:"password"`
		Database string `json:"database,omitempty" yaml:"database"`
	} `json:"mysql,omitempty" yaml:"mysql"`

	// Feishu SSO configuration
	Feishu struct {
		AppID       string `json:"app_id,omitempty" yaml:"app_id"`
		AppSecret   string `json:"app_secret,omitempty" yaml:"app_secret"`
		RedirectURI string `json:"redirect_uri,omitempty" yaml:"redirect_uri"`
	} `json:"feishu,omitempty" yaml:"feishu"`
}

func InitAppSettings(file string) {
	appInst.Do(func() {

		var (
			bs []byte
			e  error
		)

		if bs, e = os.ReadFile(file); e == nil {
			tmp := Settings{}
			if e = yaml.Unmarshal(bs, &tmp); e == nil {
				appInst.Settings = tmp
			}
		}

		if e != nil {
			log.Printf("[ERROR]\t---> init settings[%s] failed, %v", file, e)
			return
		}
		log.Printf("[INFO ]\t---> init settings[%s] success", file)
	})
}

func AppSettings() Settings {
	return appInst.Settings
}
