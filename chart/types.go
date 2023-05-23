package chart

import (
	"sort"

	"github.com/Masterminds/semver/v3"
)

type ContainerPort struct {
	Number uint16
	TCP    bool
}

type Version struct {
	Semver *semver.Version
	Raw    string
}

type VersionList []*Version

func (v VersionList) Sort() {
	sort.Slice(v, func(i, j int) bool {
		return v[i].Semver.LessThan(v[j].Semver)
	})
}

type VersionMap map[string]VersionList

func (m VersionMap) Reduce() VersionList {
	out := make([]*Version, len(m))
	j := 0
	for _, values := range m {
		values.Sort()
		out[j] = values[len(values)-1]
		j++
	}
	return out
}
