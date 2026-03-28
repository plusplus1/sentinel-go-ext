package base

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/plusplus1/sentinel-go-ext/dashboard/dao"
	"github.com/plusplus1/sentinel-go-ext/dashboard/source/reg"
)

var (
	_buildOnce = sync.Once{}
	_build     []reg.AppInfo
)

// appSettings represents app settings stored in MySQL
type appSettings struct {
	Env  string `json:"env"`
	URL  string `json:"url"`
	Desc string `json:"desc"`
}

func EnsureBuild() {
	_buildOnce.Do(func() {
		var infoList []reg.AppInfo
		set := map[string]struct{}{}

		// Only load from MySQL, allow AppInfo to be empty if no apps found
		loadFromMySQL(&infoList, set)

		_build = infoList
	})
}

func loadFromMySQL(infoList *[]reg.AppInfo, set map[string]struct{}) bool {
	cfg := dao.DefaultMySQLConfig()
	db, err := dao.NewMySQLDB(cfg)
	if err != nil {
		return false
	}
	defer db.Close()

	// Read from business_line_apps table instead of old apps table
	rows, err := db.Query("SELECT app_key, COALESCE(description, ''), COALESCE(settings, '') FROM business_line_apps WHERE status = 'active'")
	if err != nil {
		return false
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var appKey, description string
		var settingsJSON string
		if err := rows.Scan(&appKey, &description, &settingsJSON); err != nil {
			continue
		}

		if settingsJSON == "" {
			// Use default etcd endpoint if no settings
			args := map[string]string{}
			uniqId := appKey
			if _, ok := set[uniqId]; !ok {
				*infoList = append(*infoList, reg.AppInfo{
					Id:        uniqId,
					Name:      appKey,
					Env:       "default",
					Desc:      description,
					Endpoints: []string{"http://127.0.0.1:2379"},
					Type:      "etcd",
					Args:      args,
				})
				set[uniqId] = struct{}{}
				count++
			}
			continue
		}

		var settings appSettings
		if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
			continue
		}

		if settings.URL == "" {
			continue
		}

		u, err := url.ParseRequestURI(settings.URL)
		if err != nil {
			continue
		}

		endpoints := strings.Split(u.Host, ",")
		if len(endpoints) == 0 {
			continue
		}

		env := settings.Env
		if env == "" {
			env = "default"
		}

		rawId := appKey + env + settings.URL
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

			*infoList = append(*infoList, reg.AppInfo{
				Id:        uniqId,
				Name:      appKey,
				Env:       env,
				Desc:      description,
				Endpoints: endpoints,
				Type:      u.Scheme,
				Args:      args,
			})
			set[uniqId] = struct{}{}
			count++
		}
	}

	return count > 0
}

func ListAll() []reg.AppInfo {
	EnsureBuild()
	data := make([]reg.AppInfo, 0, len(_build))
	return append(data, _build...)
}

func FindApp(id string) (reg.AppInfo, bool) {

	for _, x := range ListAll() {
		if x.Id == id {
			return x, true
		}
	}
	return reg.AppInfo{}, false
}
