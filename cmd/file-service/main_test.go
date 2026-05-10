package main

import (
	"testing"

	appfx "file-service/internal/fx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
)

func TestMainPackage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main Suite")
}

var _ = Describe("main", func() {
	It("wires the expected FX modules", func() {
		originalNewApp := newApp
		originalRunApp := runApp
		defer func() {
			newApp = originalNewApp
			runApp = originalRunApp
		}()

		var modules []fx.Option
		newApp = func(opts ...fx.Option) *fx.App {
			modules = append(modules, opts...)
			return &fx.App{}
		}
		runApp = func(*fx.App) {}

		main()

		Expect(modules).To(HaveLen(7))
		Expect(modules[0]).To(Equal(appfx.ConfigModule))
		Expect(modules[1]).To(Equal(appfx.LoggerModule))
		Expect(modules[2]).To(Equal(appfx.AppModule))
		Expect(modules[3]).To(Equal(appfx.StorageModule))
		Expect(modules[4]).To(Equal(appfx.RepoModule))
		Expect(modules[5]).To(Equal(appfx.ServiceModule))
		Expect(modules[6]).To(Equal(appfx.PresentationModule))
	})
})
