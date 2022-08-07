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
	Accounts map[string]string `json:"accounts,omitempty" yaml:"accounts"`

	AppList []struct {
		App  string `json:"app,omitempty" yaml:"app"`
		Desc string `json:"desc,omitempty" yaml:"desc"`
		Url  string `json:"url,omitempty" yaml:"url"`
		Env  string `json:"env,omitempty" yaml:"env"`
	} `json:"app_list,omitempty" yaml:"app_list"`
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
