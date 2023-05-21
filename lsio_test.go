package parser

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestImages(t *testing.T) {
	images, err := GetAllImages()
	testza.AssertNoError(t, err)

	plex := images[130]

	versions, err := plex.Versions()
	testza.AssertNoError(t, err)

	version := versions[0]

	ports, err := plex.Ports(version.Raw)
	testza.AssertNoError(t, err)
	testza.AssertNotNil(t, ports)

	config, err := plex.Config(version.Raw)
	testza.AssertNoError(t, err)
	testza.AssertNotNil(t, config)
}
