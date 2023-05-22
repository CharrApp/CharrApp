// Package parser is responsible for parsing linuxserver.io readme-vars.yaml files.

package parser

import (
	"fmt"
	"io"
	"reflect"

	"github.com/noirbizarre/gonja"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Config struct {
	ProjectName               string `yaml:"project_name"`
	ProjectURL                string `yaml:"project_url"`
	ProjectLogo               string `yaml:"project_logo"`
	ProjectBlurb              string `yaml:"project_blurb" lsp:"true"`
	ProjectRepoName           string `yaml:"project_repo_name" lsp:"true"`
	ProjectDeprecationStatus  bool   `yaml:"project_deprecation_status"`
	ProjectDeprecationMessage string `yaml:"project_deprecation_message"`

	ProjectBlurbOptionalExtrasEnabled bool                 `yaml:"project_blurb_optional_extras_enabled"`
	ProjectBlurbOptionalExtras        []string             `yaml:"project_blurb_optional_extras"`
	AvailableArchitectures            []Architecture       `yaml:"available_architectures"`
	DevelopmentVersions               bool                 `yaml:"development_versions"`
	DevelopmentVersionsItems          []DevelopmentVersion `yaml:"development_versions_items"`

	CommonParamEnvVarsEnabled   bool             `yaml:"common_param_env_vars_enabled"`
	ParamContainerName          string           `yaml:"param_container_name" lsp:"true"`
	ParamUsageIncludeHostname   bool             `yaml:"param_usage_include_hostname"`
	ParamHostname               string           `yaml:"param_hostname"`
	ParamHostnameDesc           string           `yaml:"param_hostname_desc"`
	ParamUsageIncludeMacAddress bool             `yaml:"param_usage_include_mac_address"`
	ParamMacAddress             string           `yaml:"param_mac_address"`
	ParamMacAddressDesc         string           `yaml:"param_mac_address_desc"`
	ParamUsageIncludeNet        bool             `yaml:"param_usage_include_net"`
	ParamNet                    string           `yaml:"param_net"`
	ParamNetDesc                string           `yaml:"param_net_desc"`
	ParamUsageIncludeEnv        bool             `yaml:"param_usage_include_env"`
	ParamEnvVars                []EnvVar         `yaml:"param_env_vars"`
	ParamUsageIncludeVols       bool             `yaml:"param_usage_include_vols"`
	ParamVolumes                []Volume         `yaml:"param_volumes"`
	ParamUsageIncludePorts      bool             `yaml:"param_usage_include_ports"`
	ParamPorts                  []Port           `yaml:"param_ports"`
	ParamDeviceMap              bool             `yaml:"param_device_map"`
	ParamDevices                []Device         `yaml:"param_devices"`
	CapAddParam                 bool             `yaml:"cap_add_param"`
	CapAddParamVars             []CapAddVar      `yaml:"cap_add_param_vars"`
	SecurityOptParam            bool             `yaml:"security_opt_param"`
	SecurityOptParamVars        []SecurityOptVar `yaml:"security_opt_param_vars"`

	OptParamUsageIncludeEnv           bool             `yaml:"opt_param_usage_include_env"`
	OptParamEnvVars                   []EnvVar         `yaml:"opt_param_env_vars"`
	OptParamUsageIncludeVols          bool             `yaml:"opt_param_usage_include_vols"`
	OptParamVolumes                   []Volume         `yaml:"opt_param_volumes"`
	OptParamUsageIncludePorts         bool             `yaml:"opt_param_usage_include_ports"`
	OptParamPorts                     []Port           `yaml:"opt_param_ports"`
	OptParamDeviceMap                 bool             `yaml:"opt_param_device_map"`
	OptParamDevices                   []Device         `yaml:"opt_param_devices"`
	OptCapAddParam                    bool             `yaml:"opt_cap_add_param"`
	OptCapAddParamVars                []CapAddVar      `yaml:"opt_cap_add_param_vars"`
	OptSecurityOptParam               bool             `yaml:"opt_security_opt_param"`
	OptSecurityOptParamVars           []SecurityOptVar `yaml:"opt_security_opt_param_vars"`
	UnraidTemplateSync                bool             `yaml:"unraid_template_sync"`
	UnraidTemplate                    bool             `yaml:"unraid_template"`
	UnraidRequirement                 bool             `yaml:"unraid_requirement"`
	OptionalBlock1                    bool             `yaml:"optional_block_1"`
	OptionalBlock1Items               []string         `yaml:"optional_block_1_items"`
	AppSetupBlockEnabled              bool             `yaml:"app_setup_block_enabled"`
	AppSetupBlock                     string           `yaml:"app_setup_block"`
	ReadmeHwaccel                     bool             `yaml:"readme_hwaccel"`
	ReadmeKeyboard                    bool             `yaml:"readme_keyboard"`
	ReadmeMedia                       bool             `yaml:"readme_media"`
	ReadmeSeccomp                     bool             `yaml:"readme_seccomp"`
	ExternalApplicationSnippetEnabled bool             `yaml:"external_application_snippet_enabled"`
	ExternalApplicationCliBlock       string           `yaml:"external_application_cli_block"`
	ExternalApplicationComposeBlock   string           `yaml:"external_application_compose_block"`
	ExternalApplicationUnraidBlock    string           `yaml:"external_application_unraid_block"`
	Changelogs                        []Changelog      `yaml:"changelogs"`
}

type Architecture struct {
	Arch string `yaml:"arch" lsp:"true"`
	Tag  string `yaml:"tag"`
}

type DevelopmentVersion struct {
	Tag   string `yaml:"tag"`
	Desc  string `yaml:"desc"`
	Extra string `yaml:"extra,omitempty"`
}

type EnvVar struct {
	EnvVar     string   `yaml:"env_var"`
	EnvValue   string   `yaml:"env_value"`
	Desc       string   `yaml:"desc"`
	EnvOptions []string `yaml:"env_options,omitempty"`
}

type Volume struct {
	VolPath     string `yaml:"vol_path"`
	VolHostPath string `yaml:"vol_host_path"`
	Desc        string `yaml:"desc"`
	Name        string `yaml:"name"`
	Default     bool   `yaml:"default,omitempty"`
}

type Port struct {
	ExternalPort string `yaml:"external_port"`
	InternalPort string `yaml:"internal_port"`
	PortDesc     string `yaml:"port_desc"`
	Name         string `yaml:"name"`
}

type Device struct {
	DevicePath     string `yaml:"device_path"`
	DeviceHostPath string `yaml:"device_host_path"`
	Desc           string `yaml:"desc"`
	Name           string `yaml:"name"`
}

type CapAddVar struct {
	CapAddVar string `yaml:"cap_add_var"`
}

type SecurityOptVar struct {
	RunVar     string `yaml:"run_var"`
	ComposeVar string `yaml:"compose_var"`
	Desc       string `yaml:"desc"`
}

type Changelog struct {
	Date string `yaml:"date"`
	Desc string `yaml:"desc"`
}

func Parse(reader io.Reader) (*Config, error) {
	config := Config{}
	if err := yaml.NewDecoder(reader).Decode(&config); err != nil {
		return nil, errors.Wrap(err, "failed decoding yaml")
	}
	if err := interpolate(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

func interpolate(config *Config) error {
	return interpolateRecursive(reflect.ValueOf(config), nil, config)
}

func interpolateRecursive(val reflect.Value, typ *reflect.StructField, config *Config) error {
	switch val.Kind() {
	case reflect.Ptr:
		return interpolateRecursive(val.Elem(), nil, config)
	case reflect.Slice:
		for i := 0; i < val.Len(); i += 1 {
			if err := interpolateRecursive(val.Index(i), nil, config); err != nil {
				return err
			}
		}
	case reflect.Struct:
		for i := 0; i < val.NumField(); i += 1 {
			s := val.Type().Field(i)
			if err := interpolateRecursive(val.Field(i), &s, config); err != nil {
				return err
			}
		}
	case reflect.String:
		if typ != nil && typ.Tag.Get("lsp") != "true" {
			return nil
		}
		newVal, err := executeTemplate(val.String(), config)
		if err != nil {
			return err
		}
		val.SetString(newVal)
	}
	return nil
}

func executeTemplate(s string, c *Config) (string, error) {
	tmpl, err := gonja.FromString(s)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	result, err := tmpl.Execute(gonja.Context{
		"project_name": c.ProjectName,
		"project_url":  c.ProjectURL,
		"arch_x86_64":  "x86-64",
		"arch_arm64":   "arm64",
		"arch_armhf":   "armhf",
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return result, nil
}
