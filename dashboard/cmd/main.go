package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"syscall"

	"github.com/fvbock/endless"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"

	"github.com/plusplus1/sentinel-go-ext/dashboard"
	"github.com/plusplus1/sentinel-go-ext/dashboard/config"
)

const (
	appVer  = "1.0.0"
	appName = "Sentinel-Dashboard-Go"

	defaultPort  = 6111
	defaultUsage = `A golang implement of sentinel dashboard `
)

var (
	logger     = zap.S()
	httpEngine *gin.Engine
)

func main() {

	app := cli.NewApp()
	{
		app.Name, app.Version, app.Usage = appName, appVer, defaultUsage
		app.Before, app.After, app.Action = before, after, startUp

		app.Flags = []cli.Flag{
			&cli.Int64Flag{Name: `port`, Aliases: []string{`p`}, Usage: `http serve port`, Value: defaultPort},
			&cli.StringFlag{Name: `conf`, Aliases: []string{`c`}, Usage: `settings file`, Value: `conf/dashboard-settings.yaml`},
		}
	}

	if e := app.Run(os.Args); e != nil {
		logger.Infow(`server exit!`, zap.Error(e))
	}

	fmt.Println("exit!")
}

func startUp(ctx *cli.Context) error {

	port := ctx.Int64(`port`)
	if httpEngine == nil {
		httpEngine = gin.Default()
	}

	addr := ":" + strconv.FormatInt(port, 10)
	srv := endless.NewServer(addr, httpEngine)
	srv.BeforeBegin = func(add string) {
		logger.Infof("Start: %s , actual pid is %d", "http://0.0.0.0"+add, syscall.Getpid())
	}
	err := srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func before(ctx *cli.Context) error {
	// init logger
	logConfig := zap.NewProductionConfig()
	logConfig.OutputPaths = []string{`stdout`}
	if lg, e := logConfig.Build(); e == nil {
		zap.ReplaceGlobals(lg)
		logger = zap.S()
	}

	// init settings
	config.InitAppSettings(ctx.String(`conf`))

	// init http engine
	gin.ForceConsoleColor()
	httpEngine = gin.Default()
	httpEngine.Use(gzip.Gzip(gzip.BestSpeed))

	// init routes
	dashboard.InstallApi(httpEngine)

	return nil
}

func after(_ *cli.Context) error {
	_ = logger.Sync()
	return nil
}
