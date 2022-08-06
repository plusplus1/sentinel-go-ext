package base

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/plusplus1/sentinel-go-ext/dashboard/config"
	"github.com/plusplus1/sentinel-go-ext/source"
	"github.com/plusplus1/sentinel-go-ext/source/reg"
)

var (
	_buildOnce = sync.Once{}
	_build     []reg.AppInfo
)

func EnsureBuild() {
	_buildOnce.Do(func() {

		var infoList []reg.AppInfo

		set := map[string]struct{}{}

		for _, app := range config.AppSettings().AppList {
			if app.App == `` || app.Env == `` || app.Url == `` {
				continue
			}

			if u, e := url.ParseRequestURI(app.Url); e == nil {
				if !source.IsSupported(u.Scheme) {
					continue
				}

				if endpoints := strings.Split(u.Host, `,`); len(endpoints) > 0 {

					rawId := app.App + app.Env + app.Url

					m := md5.New()
					m.Write([]byte(rawId))
					hex := m.Sum(nil)
					uniqId := fmt.Sprintf("%x", hex)

					if _, ok := set[uniqId]; !ok {

						args := map[string]string{}
						for k, v := range u.Query() {
							if len(v) > 0 {
								args[k] = v[0]
							}
						}

						infoList = append(infoList, reg.AppInfo{
							Id:        uniqId,
							Name:      app.App,
							Env:       app.Env,
							Desc:      app.Desc,
							Endpoints: endpoints,
							Type:      u.Scheme,
							Args:      args,
						})
					}

				}
			}
		}

		_build = infoList
	})
}

func ListAll() []reg.AppInfo {
	EnsureBuild()
	data := make([]reg.AppInfo, 0, len(_build))
	for _, i := range _build {
		data = append(data, i)
	}
	return data
}

func FindApp(id string) (reg.AppInfo, bool) {

	for _, x := range ListAll() {
		if x.Id == id {
			return x, true
		}
	}
	return reg.AppInfo{}, false
}
