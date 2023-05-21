package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/MarvinJWendt/testza"
)

const baseOut = "out"

func TestE2E(t *testing.T) {
	images, err := GetAllImages()
	testza.AssertNoError(t, err)

	for _, image := range images {
		println("processing", image.Name)
		writeOut(t, image)
	}
}

func writeOut(t *testing.T, img *Image) {
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

	chartData := ChartData{
		Config:  config,
		Version: fmt.Sprintf("%d.%d.%d", version.Semver.Major(), version.Semver.Minor(), version.Semver.Patch()),
		Ports:   ports,
	}

	chart, err := chartData.Chart()
	testza.AssertNoError(t, err)

	for name, b := range chart {
		realPath := filepath.Join(baseOut, img.Name, name)
		testza.AssertNoError(t, os.MkdirAll(filepath.Dir(realPath), 0777))
		testza.AssertNoError(t, os.WriteFile(realPath, b, 0777))
	}
}
