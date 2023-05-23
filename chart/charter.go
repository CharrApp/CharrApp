package chart

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"

	"github.com/charrapp/charrapp/parser"
)

const (
	templateDir       = "template"
	templateExtension = ".gotmpl"
)

type Data struct {
	Config  *parser.Config
	Version string
	Ports   []*ContainerPort
}

// GenerateChart constructs the chart and returns a map containing all generated files
func (data *Data) GenerateChart() (map[string][]byte, error) {
	return data.recursiveGenerateChart(templateDir)
}

func (data *Data) recursiveGenerateChart(dirName string) (map[string][]byte, error) {
	outFiles := make(map[string][]byte)

	dir, err := os.ReadDir(dirName)
	if err != nil {
		return nil, fmt.Errorf("failed reading directory %s: %w", dirName, err)
	}

	for _, entry := range dir {
		newPath := filepath.Join(dirName, entry.Name())
		if entry.IsDir() {
			newFiles, err := data.recursiveGenerateChart(newPath)
			if err != nil {
				return nil, err
			}

			for p, data := range newFiles {
				outPath := filepath.Join(newPath, p)
				outPath = strings.TrimPrefix(outPath, templateDir)
				outFiles[outPath] = data
			}

			continue
		}

		file, err := os.ReadFile(newPath)
		if err != nil {
			return nil, fmt.Errorf("failed reading file %s: %w", newPath, err)
		}

		if filepath.Ext(entry.Name()) == templateExtension {
			out, err := data.templateFile(string(file))
			if err != nil {
				return nil, fmt.Errorf("failed templating %s: %w", newPath, err)
			}

			strippedName := entry.Name()[:len(entry.Name())-len(templateExtension)]
			outFiles[strippedName] = out
		} else {
			outFiles[entry.Name()] = file
		}
	}

	return outFiles, nil
}

func (data *Data) templateFile(file string) ([]byte, error) {
	funcMap := sprig.HermeticTxtFuncMap()
	funcMap["toYaml"] = func(v interface{}) string {
		data, err := yaml.Marshal(v)
		if err != nil {
			return ""
		}
		return strings.TrimSuffix(string(data), "\n")
	}

	tmpl, err := template.New("chart").Funcs(funcMap).Parse(file)
	if err != nil {
		return nil, fmt.Errorf("failed parsing template: %w", err)
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return nil, fmt.Errorf("failed executing template: %w", err)
	}

	return out.Bytes(), nil
}
