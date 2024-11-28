package acceptance

import (
	"embed"
	_ "embed"
	"log"
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

//go:embed features/*
var features embed.FS

var opts = godog.Options{
	Format:      "pretty",
	Paths:       []string{"features"},
	FS:          features,
	Output:      colors.Colored(os.Stdout),
	Concurrency: 1,
}

func init() {
	logOpts := zap.Options{
		Development: true,
	}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&logOpts)))

	godog.BindCommandLineFlags("godog.", &opts)
}

func TestMain(m *testing.M) {
	// parse CLI arguments
	pflag.Parse()
	opts.Paths = pflag.Args()

	// run tests
	sc := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options:             &opts,
	}.Run()
	switch sc {
	//	0 - success
	case 0:
		return

	//	1 - failed
	//	2 - command line usage error
	// 128 - or higher, os signal related error exit codes
	default:
		log.Fatalf("non-zero status returned (%d), failed to run feature tests", sc)
	}
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	InjectSteps(ctx)
	InjectHooks(ctx)
}
