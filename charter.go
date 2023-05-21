package parser

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"
)

const (
	templateDir       = "template"
	templateExtension = ".gotmpl"
)

type ChartData struct {
	Config  *Config
	Version string
	Ports   []*ContainerPort
}

func (data *ChartData) Chart() (map[string][]byte, error) {
	return data.recursiveChart(templateDir)
}

func (data *ChartData) recursiveChart(dirName string) (map[string][]byte, error) {
	outFiles := make(map[string][]byte)

	dir, err := os.ReadDir(dirName)
	if err != nil {
		return nil, fmt.Errorf("failed reading directory %s: %w", dirName, err)
	}

	for _, entry := range dir {
		newPath := filepath.Join(dirName, entry.Name())
		if entry.IsDir() {
			newFiles, err := data.recursiveChart(newPath)
			if err != nil {
				return nil, err
			}

			for p, data := range newFiles {
				outPath := filepath.Join(newPath, p)
				if strings.HasPrefix(outPath, templateDir) {
					outPath = strings.TrimLeft(outPath, templateDir)
				}
				outFiles[outPath] = data
			}

			continue
		}

		file, err := os.ReadFile(newPath)
		if err != nil {
			return nil, fmt.Errorf("failed reading file %s: %w", newPath, err)
		}

		if filepath.Ext(entry.Name()) == templateExtension {
			funcMap := sprig.HermeticTxtFuncMap()
			funcMap["toYaml"] = func(v interface{}) string {
				data, err := yaml.Marshal(v)
				if err != nil {
					return ""
				}
				return strings.TrimSuffix(string(data), "\n")
			}

			tmpl, err := template.New("chart").Funcs(funcMap).Parse(string(file))

			if err != nil {
				return nil, fmt.Errorf("failed parsing template %s: %w", newPath, err)
			}

			var out bytes.Buffer
			if err := tmpl.Execute(&out, data); err != nil {
				return nil, fmt.Errorf("failed executing template %s: %w", newPath, err)
			}

			outFiles[entry.Name()[:len(entry.Name())-len(templateExtension)]] = out.Bytes()
		} else {
			outFiles[entry.Name()] = file
		}
	}

	return outFiles, nil
}
