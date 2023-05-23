package lsio

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/pkg/errors"

	"github.com/charrapp/charrapp/chart"
	"github.com/charrapp/charrapp/parser"
	"github.com/charrapp/charrapp/utils"
)

const (
	lsioURL     = "https://fleet.linuxserver.io/?key=10:linuxserver"
	lsioCR      = "lscr.io/linuxserver/"
	rawTemplate = "https://raw.githubusercontent.com/linuxserver/docker-%s/%s/%s"
	udpSuffix   = "/udp"
)

var (
	urlRegex    = regexp.MustCompile(`href="/image\?name=linuxserver/(.+?)"`)
	exposeRegex = regexp.MustCompile(`EXPOSE (.+)`)
	portRegex   = regexp.MustCompile(`(\d+)(\/tcp|\/udp)?`)
)

type Image struct {
	Name     string
	versions chart.VersionList
}

func GetAllImages() ([]*Image, error) {
	resp, err := http.Get(lsioURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed fetching lsio")
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed reading response body")
	}

	matches := urlRegex.FindAllSubmatch(body, -1)

	images := make([]*Image, len(matches))
	for i, match := range matches {
		images[i] = &Image{
			Name: string(match[1]),
		}
	}

	return images, nil
}

func (i *Image) Versions() ([]*chart.Version, error) {
	if i.versions != nil {
		return i.versions, nil
	}

	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://github.com/linuxserver/docker-" + i.Name},
	})

	refs, err := remote.List(&git.ListOptions{
		Auth:            nil,
		InsecureSkipTLS: false,
		CABundle:        nil,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed listing refs")
	}

	versionMap, err := utils.ExtractVersions(refs)
	if err != nil {
		return nil, err
	}

	i.versions = versionMap.Reduce()
	i.versions.Sort()

	return i.versions, nil
}

func (i *Image) URL() string {
	return lsioCR + i.Name
}

func (i *Image) fetch(tag string, file string) ([]byte, error) {
	url := fmt.Sprintf(rawTemplate, i.Name, tag, file)
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "failed fetching url: "+url)
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed reading body")
	}

	return body, nil
}

func (i *Image) Ports(version string) ([]*chart.ContainerPort, error) {
	cfg, err := i.Config(version)
	if err != nil {
		return nil, err
	}

	ports := make([]*chart.ContainerPort, 0)
	if cfg.ParamPorts != nil || cfg.OptParamPorts != nil {
		for _, portList := range [][]parser.Port{cfg.ParamPorts, cfg.OptParamPorts} {
			for _, port := range portList {
				port, err := parsePort(port.InternalPort)
				if err != nil {
					return nil, err
				}

				if port != nil {
					ports = append(ports, port)
				}
			}
		}
		return ports, nil
	}

	body, err := i.fetch(version, "Dockerfile")
	if err != nil {
		return nil, err
	}

	matches := exposeRegex.FindAllSubmatch(body, -1)

	for _, match := range matches {
		port, err := parsePort(string(match[1]))
		if err != nil {
			return nil, err
		}

		if port != nil {
			ports = append(ports, port)
		}
	}

	return ports, nil
}

func (i *Image) Config(version string) (*parser.Config, error) {
	body, err := i.fetch(version, "readme-vars.yml")
	if err != nil {
		return nil, err
	}
	return parser.Parse(bytes.NewReader(body))
}

func parsePort(s string) (*chart.ContainerPort, error) {
	portMatch := portRegex.FindStringSubmatch(s)
	if portMatch == nil {
		return nil, nil
	}

	tcp := true
	if len(portMatch) > 2 && string(portMatch[2]) == udpSuffix {
		tcp = false
	}

	n, err := strconv.ParseUint(portMatch[1], 10, 16)
	if err != nil {
		return nil, errors.Wrap(err, "failed parsing port as uint16")
	}

	return &chart.ContainerPort{
		Number: uint16(n),
		TCP:    tcp,
	}, nil
}
