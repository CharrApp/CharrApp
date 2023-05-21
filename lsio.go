package parser

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"

	"github.com/Masterminds/semver"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/pkg/errors"
)

const lsioURL = "https://fleet.linuxserver.io/?key=10:linuxserver"
const lsioCR = "lscr.io/linuxserver/"
const rawTemplate = "https://raw.githubusercontent.com/linuxserver/docker-%s/%s/%s"

var urlRegex = regexp.MustCompile(`href="/image\?name=linuxserver/(.+?)"`)

type Image struct {
	Name     string
	versions []*Version
}

func GetAllImages() ([]*Image, error) {
	resp, err := http.Get(lsioURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed fetching lsio")
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
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

var refVersionRegex = regexp.MustCompile(`^refs/tags/v?(\d+\.\d+\.\d+)(\.?-?)(.*)$`)

type Version struct {
	Semver *semver.Version
	Raw    string
}

func (i *Image) Versions() ([]*Version, error) {
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
		return nil, err
	}

	versionMap := make(map[string][]*Version, 0)
	for _, ref := range refs {
		match := refVersionRegex.FindStringSubmatch(ref.Name().String())
		if match != nil {
			fullVersion := match[1]
			if len(match[3]) > 0 {
				fullVersion += "-" + match[3]
			}

			v, err := semver.NewVersion(fullVersion)
			if err != nil {
				fullVersion = match[1]
				v, err = semver.NewVersion(fullVersion)
				if err != nil {
					return nil, fmt.Errorf("failed parsing version: %s: %w", fullVersion, err)
				}
			}

			versionMap[match[1]] = append(versionMap[match[1]], &Version{
				Semver: v,
				Raw:    match[1] + match[2] + match[3],
			})
		}
	}

	i.versions = make([]*Version, len(versionMap))
	j := 0
	for _, values := range versionMap {
		sort.Slice(values, func(i, j int) bool {
			return values[i].Semver.LessThan(values[j].Semver)
		})
		i.versions[j] = values[len(values)-1]
		j++
	}

	sort.Slice(i.versions, func(a, b int) bool {
		return i.versions[a].Semver.LessThan(i.versions[b].Semver)
	})

	return i.versions, nil
}

func (i *Image) URL() string {
	return lsioCR + i.Name
}

type ContainerPort struct {
	Number uint16
	TCP    bool
}

var exposeRegex = regexp.MustCompile(`EXPOSE (.+)`)
var portRegex = regexp.MustCompile(`(\d+)(\/tcp|\/udp)?`)

func (i *Image) fetch(tag string, file string) ([]byte, error) {
	resp, err := http.Get(fmt.Sprintf(rawTemplate, i.Name, tag, file))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (i *Image) Ports(version string) ([]*ContainerPort, error) {
	cfg, err := i.Config(version)
	if err != nil {
		return nil, err
	}

	ports := make([]*ContainerPort, 0)
	if cfg.ParamPorts != nil || cfg.OptParamPorts != nil {
		for _, port := range cfg.ParamPorts {
			portMatch := portRegex.FindStringSubmatch(port.InternalPort)
			if portMatch != nil {
				tcp := true
				if len(portMatch) > 2 && string(portMatch[2]) == "/udp" {
					tcp = false
				}

				n, err := strconv.ParseUint(portMatch[1], 10, 16)
				if err != nil {
					return nil, err
				}

				ports = append(ports, &ContainerPort{
					Number: uint16(n),
					TCP:    tcp,
				})
			}
		}

		for _, port := range cfg.OptParamPorts {
			portMatch := portRegex.FindStringSubmatch(port.InternalPort)
			if portMatch != nil {
				tcp := true
				if len(portMatch) > 2 && string(portMatch[2]) == "/udp" {
					tcp = false
				}

				n, err := strconv.ParseUint(portMatch[1], 10, 16)
				if err != nil {
					return nil, err
				}

				ports = append(ports, &ContainerPort{
					Number: uint16(n),
					TCP:    tcp,
				})
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
		portMatch := portRegex.FindAllSubmatch(match[1], -1)
		for _, p := range portMatch {
			tcp := true
			if len(p) > 2 && string(p[2]) == "/udp" {
				tcp = false
			}

			n, err := strconv.ParseUint(string(p[1]), 10, 16)
			if err != nil {
				return nil, err
			}

			ports = append(ports, &ContainerPort{
				Number: uint16(n),
				TCP:    tcp,
			})
		}
	}

	return ports, nil
}

func (i *Image) Config(version string) (*Config, error) {
	body, err := i.fetch(version, "readme-vars.yml")
	if err != nil {
		return nil, err
	}
	return Parse(bytes.NewReader(body))
}
