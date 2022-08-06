package dashboard

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/plusplus1/sentinel-go-ext/dashboard/api/app"
	"github.com/plusplus1/sentinel-go-ext/dashboard/config"
	"github.com/plusplus1/sentinel-go-ext/dashboard/webui"
)

func InstallApi(router gin.IRouter) {

	// set auth
	auth := gin.BasicAuth(config.AppSettings().Accounts)
	// set
	staticGroup := router.Group("/web", auth)
	{
		staticGroup.StaticFS("/", http.FS(webui.DistFiles))
		if eg := router.(*gin.Engine); eg != nil {
			eg.NoRoute(func(c *gin.Context) {
				c.Redirect(302, "/web/dist/page.html")
			})
		}
	}

	apiGroup := router.Group("/api/", auth)
	{
		apiGroup.GET("/app/list", app.ListApps)
		apiGroup.GET("/app/rule/flow/list", app.ListFlowRules)
		apiGroup.GET("/app/rule/circuitbreaker/list", app.ListCircuitbreakerRules)
	}

	//debug
	router.GET("/fs/", scanFs)

}

func scanFs(c *gin.Context) {

	var (
		ret     = gin.H{}
		items   []gin.H
		path, _ = c.GetQuery("p")
	)

	if path == `` {
		path = "/"
	}
	dirs, e := webui.DistFiles.ReadDir(path)
	for _, d := range dirs {
		items = append(items, gin.H{
			`name`:   d.Name(),
			`is_dir`: d.IsDir(),
			`type`:   d.Type(),
		})
	}
	ret[`items`] = items
	ret[`path`] = path
	if e != nil {
		ret[`error`] = e.Error()
	}

	c.JSONP(200, ret)
}
