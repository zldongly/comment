// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	"github.com/zldongly/comment/app/helloworld/internal/biz"
	"github.com/zldongly/comment/app/helloworld/internal/conf"
	"github.com/zldongly/comment/app/helloworld/internal/data"
	"github.com/zldongly/comment/app/helloworld/internal/server"
	"github.com/zldongly/comment/app/helloworld/internal/service"
)

// initApp init kratos application.
func initApp(*conf.Server, *conf.Data, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
