package source

import (
	_ "github.com/alibaba/sentinel-golang/api"
	_ "github.com/alibaba/sentinel-golang/core/base"
	_ "github.com/alibaba/sentinel-golang/core/circuitbreaker"
	_ "github.com/alibaba/sentinel-golang/core/config"
	_ "github.com/alibaba/sentinel-golang/core/flow"
	_ "github.com/alibaba/sentinel-golang/core/hotspot"
	_ "github.com/alibaba/sentinel-golang/core/isolation"
	_ "github.com/alibaba/sentinel-golang/core/stat"
	_ "github.com/alibaba/sentinel-golang/core/system_metric"
	_ "github.com/alibaba/sentinel-golang/ext/datasource"
	_ "github.com/alibaba/sentinel-golang/logging"
	_ "github.com/alibaba/sentinel-golang/util"

	_ "github.com/plusplus1/sentinel-go-ext/source/etcd"

	"github.com/plusplus1/sentinel-go-ext/source/reg"
)

func IsSupported(s string) bool {
	return reg.SourceBuilder(s) != nil
}
