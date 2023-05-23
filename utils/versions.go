package utils

import (
	"fmt"
	"regexp"

	"github.com/Masterminds/semver/v3"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/charrapp/charrapp/chart"
)

var (
	refSemverRegex     = regexp.MustCompile(`^refs/tags/(.*?\D)?(\d+\.\d+\.\d+)(\.?-?)(.*)$`)
	refHalfSemverRegex = regexp.MustCompile(`^refs/tags/(.*?\D)?(\d+\.\d+)(\.?-?)(.*)$`)
	dateSemverRegex    = regexp.MustCompile(`^refs/tags/(.*?\D)?(\d+-\d+-\d+)(\.?-?)(.*)$`)
	rawLSIORegex       = regexp.MustCompile(`^refs/tags/(.*)-ls(\d+)$`)
	rawNumberRegex     = regexp.MustCompile(`^refs/tags/(\d+)$`)
)

func ExtractVersions(refs []*plumbing.Reference) (chart.VersionMap, error) {
	versionMap := make(chart.VersionMap)
	for _, ref := range refs {
		match := refSemverRegex.FindStringSubmatch(ref.Name().String())
		if match != nil {
			fullVersion := match[2]
			if len(match[4]) > 0 {
				fullVersion += "-" + match[4]
			}

			v, err := semver.NewVersion(fullVersion)
			if err != nil {
				fullVersion = match[2]
				v, err = semver.NewVersion(fullVersion)
				if err != nil {
					return nil, fmt.Errorf("failed parsing version: %s: %w", fullVersion, err)
				}
			}

			versionMap[match[1]] = append(versionMap[match[1]], &chart.Version{
				Semver: v,
				Raw:    match[1] + match[2] + match[3] + match[4],
			})
		}
	}

	if len(versionMap) == 0 {
		for _, ref := range refs {
			match := dateSemverRegex.FindStringSubmatch(ref.Name().String())
			if match != nil {
				fullVersion := match[2]
				if len(match[4]) > 0 {
					fullVersion += "-" + match[4]
				}

				v, err := semver.NewVersion(fullVersion)
				if err != nil {
					fullVersion = match[2]
					v, err = semver.NewVersion(fullVersion)
					if err != nil {
						return nil, fmt.Errorf("failed parsing version: %s: %w", fullVersion, err)
					}
				}

				versionMap[match[2]] = append(versionMap[match[2]], &chart.Version{
					Semver: v,
					Raw:    match[1] + match[2] + match[3] + match[4],
				})
			}
		}
	}

	if len(versionMap) == 0 {
		for _, ref := range refs {
			match := refHalfSemverRegex.FindStringSubmatch(ref.Name().String())
			if match != nil {
				fullVersion := match[2] + ".0"
				if len(match[4]) > 0 {
					fullVersion += "-" + match[4]
				}

				v, err := semver.NewVersion(fullVersion)
				if err != nil {
					fullVersion = match[2] + ".0"
					v, err = semver.NewVersion(fullVersion)
					if err != nil {
						return nil, fmt.Errorf("failed parsing version: %s: %w", fullVersion, err)
					}
				}

				versionMap[match[2]+".0"] = append(versionMap[match[2]+".0"], &chart.Version{
					Semver: v,
					Raw:    match[1] + match[2] + match[3] + match[4],
				})
			}
		}
	}

	if len(versionMap) == 0 {
		for _, ref := range refs {
			match := rawLSIORegex.FindStringSubmatch(ref.Name().String())
			if match != nil {
				fullVersion := match[2] + ".0.0"
				if len(match[1]) > 0 {
					fullVersion += "-" + match[1]
				}

				v, err := semver.NewVersion(fullVersion)
				if err != nil {
					fullVersion = match[2] + ".0.0"
					v, err = semver.NewVersion(fullVersion)
					if err != nil {
						return nil, fmt.Errorf("failed parsing version: %s: %w", fullVersion, err)
					}
				}

				versionMap[match[2]+".0.0"] = append(versionMap[match[2]+".0.0"], &chart.Version{
					Semver: v,
					Raw:    match[1] + "-ls" + match[2],
				})
			}
		}
	}

	if len(versionMap) == 0 {
		for _, ref := range refs {
			match := rawNumberRegex.FindStringSubmatch(ref.Name().String())
			if match != nil {
				fullVersion := match[1] + ".0.0"

				v, err := semver.NewVersion(fullVersion)
				if err != nil {
					return nil, fmt.Errorf("failed parsing version: %s: %w", fullVersion, err)
				}

				versionMap[match[1]+".0.0"] = append(versionMap[match[1]+".0.0"], &chart.Version{
					Semver: v,
					Raw:    match[1],
				})
			}
		}
	}

	if len(versionMap) == 0 {
		println("Could not parse any of these as a version:")
		for _, ref := range refs {
			if ref.Name().IsTag() {
				println("- " + ref.Name().String())
			}
		}
	}

	return versionMap, nil
}
