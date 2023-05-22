package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/charrapp/charrapp/chart"
	"github.com/charrapp/charrapp/lsio"
)

const baseOut = "out"

func TestE2E(t *testing.T) {
	images, err := lsio.GetAllImages()
	testza.AssertNoError(t, err)

	for _, image := range images {
		println("processing", image.Name)
		writeOut(t, image)
	}
}

func writeOut(t *testing.T, img *lsio.Image) {
	versions, err := img.Versions()
	testza.AssertNoError(t, err)

	if len(versions) == 0 {
		println("Found 0 versions!")
		return
	}

	version := versions[len(versions)-1]

	config, err := img.Config(version.Raw)
	if err != nil {
		println("FAILED:", err.Error())
		return
	}

	ports, err := img.Ports(version.Raw)
	testza.AssertNoError(t, err)

	chartData := chart.Data{
		Config:  config,
		Version: fmt.Sprintf("%d.%d.%d", version.Semver.Major(), version.Semver.Minor(), version.Semver.Patch()),
		Ports:   ports,
	}

	files, err := chartData.GenerateChart()
	testza.AssertNoError(t, err)

	for name, b := range files {
		realPath := filepath.Join(baseOut, img.Name, name)
		testza.AssertNoError(t, os.MkdirAll(filepath.Dir(realPath), 0o777))
		testza.AssertNoError(t, os.WriteFile(realPath, b, 0o777))
	}
}
