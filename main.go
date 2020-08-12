package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/OpenPeeDeeP/xdg"
	"github.com/hoisie/mustache"
	"github.com/inconshreveable/log15"
	"gopkg.in/yaml.v2"
)

type Configuration struct {
	Scheme              string `yaml:"scheme"`
	SchemeRepositoryURL string `yaml:"scheme_repository_url"`
	SchemesListURL      string `yaml:"schemes_list_url"`
	TemplatesListURL    string `yaml:"templates_list_url"`
	Applications        map[string]struct {
		Hook                  string `yaml:"hook"`
		TemplateRepositoryURL string `yaml:"template_repository_url"`
		Files                 map[string]struct {
			Mode        string `yaml:"mode"`
			StartMarker string `yaml:"start_marker"`
			EndMarker   string `yaml:"end_marker"`
			Destination string `yaml:"destination"`
		} `yaml:"files"`
	} `yaml:"applications"`
}

type TemplateConfig map[string]struct {
	Extension string `yaml:"extension"`
	Output    string `yaml:"output"`
}

type ColorScheme struct {
	Name   string `yaml:"scheme"`
	Author string `yaml:"author"`
	Base00 string `yaml:"base00"`
	Base01 string `yaml:"base01"`
	Base02 string `yaml:"base02"`
	Base03 string `yaml:"base03"`
	Base04 string `yaml:"base04"`
	Base05 string `yaml:"base05"`
	Base06 string `yaml:"base06"`
	Base07 string `yaml:"base07"`
	Base08 string `yaml:"base08"`
	Base09 string `yaml:"base09"`
	Base0A string `yaml:"base0A"`
	Base0B string `yaml:"base0B"`
	Base0C string `yaml:"base0C"`
	Base0D string `yaml:"base0D"`
	Base0E string `yaml:"base0E"`
	Base0F string `yaml:"base0F"`
}

func (s ColorScheme) Vars() map[string]interface{} {
	var vars = map[string]interface{}{
		"scheme-name":   s.Name,
		"scheme-author": s.Author,
	}

	for base, color := range map[string]string{
		"00": s.Base00,
		"01": s.Base01,
		"02": s.Base02,
		"03": s.Base03,
		"04": s.Base04,
		"05": s.Base05,
		"06": s.Base06,
		"07": s.Base07,
		"08": s.Base08,
		"09": s.Base09,
		"0A": s.Base0A,
		"0B": s.Base0B,
		"0C": s.Base0C,
		"0D": s.Base0D,
		"0E": s.Base0E,
		"0F": s.Base0F,
	} {
		vars[fmt.Sprintf("base%s-hex", base)] = color

		vars[fmt.Sprintf("base%s-hex-r", base)] = color[0:2]
		vars[fmt.Sprintf("base%s-rgb-r", base)] = toRGB(color[0:2])
		vars[fmt.Sprintf("base%s-dec-r", base)] = toDec(color[0:2])

		vars[fmt.Sprintf("base%s-hex-g", base)] = color[2:4]
		vars[fmt.Sprintf("base%s-rgb-g", base)] = toRGB(color[2:4])
		vars[fmt.Sprintf("base%s-dec-g", base)] = toDec(color[2:4])

		vars[fmt.Sprintf("base%s-hex-r", base)] = color[4:6]
		vars[fmt.Sprintf("base%s-rgb-r", base)] = toRGB(color[4:6])
		vars[fmt.Sprintf("base%s-dec-r", base)] = toDec(color[4:6])
	}

	return vars
}

func toRGB(c string) uint64 {
	v, _ := strconv.ParseUint(c, 16, 32)
	return v
}

func toDec(c string) float64 {
	v := toRGB(c)
	return float64(v) / 255
}

func main() {
	log := log15.New()

	log.Debug("retrieving configuration")
	config, err := loadConfiguration()
	if err != nil {
		log.Error("retrieving configuration", "err", err)
		return
	}

	switch len(os.Args) {
	case 3:
		config.Scheme = os.Args[1]
		config.SchemeRepositoryURL = os.Args[2]
	case 2:
		config.Scheme = os.Args[1]
		config.SchemeRepositoryURL = ""
	case 1:
		// Nothing to do
	default:
		log.Error("too many arguments")
		return
	}

	scheme, err := loadScheme(log, config)
	if err != nil {
		log.Error("retrieving color scheme", "err", err)
		return
	}

	log.Debug("retrieving templates list", "url", config.TemplatesListURL)
	var templates map[string]string
	err = loadYAMLFile(config.TemplatesListURL, &templates)
	if err != nil {
		log.Error("retrieving templates list", "err", err)
		return
	}

	for template, app := range config.Applications {
		log := log.New("template", template)

		if len(app.TemplateRepositoryURL) == 0 {
			if _, ok := templates[template]; !ok {
				log.Error("finding template", "err", "can't find template in list")
				continue
			}
			app.TemplateRepositoryURL = templates[template]
		}

		log.Info("building template", "template_repository_url", app.TemplateRepositoryURL)

		parts := strings.Split(app.TemplateRepositoryURL, "/")
		if len(parts) != 5 {
			log.Error("building template", "err", "unhandled template repository url format", "template_repository_url", app.TemplateRepositoryURL)
			continue
		}

		user, repository := parts[3], parts[4]

		var templateConfig TemplateConfig
		err = loadYAMLFile(githubFileURL(user, repository, "templates/config.yaml"), &templateConfig)
		if err != nil {
			log.Error("retrieving template configuration", "err", err)
			continue
		}

		for file, _ := range templateConfig {
			log := log.New("file", file)

			body, err := loadFile(githubFileURL(user, repository, fmt.Sprintf("templates/%s.mustache", file)))
			if err != nil {
				log.Error("retrieving file")
				continue
			}

			tpl, err := mustache.ParseString(string(body))
			if err != nil {
				log.Error("parsing template", "err", err)
				continue
			}

			destination := expandPath(app.Files[file].Destination)
			result := tpl.Render(scheme.Vars())

			// If the mode is replace, we want to replace the
			// content of the destination file with the result from
			// the start marker to the end marker. We just load the
			// current destination file, replace in-memory and
			// continue as if the result was the complete file from
			// start.
			if app.Files[file].Mode == "replace" {
				if len(app.Files[file].StartMarker) == 0 {
					log.Error("empty start marker")
					continue
				}

				if len(app.Files[file].EndMarker) == 0 {
					log.Error("empty start marker")
					continue
				}

				raw, err := ioutil.ReadFile(destination)
				if err != nil {
					log.Error("loading destination file", "err", err)
					continue
				}

				var buf bytes.Buffer
				scanner := bufio.NewScanner(bytes.NewReader(raw))
				for scanner.Scan() {
					line := scanner.Text()
					buf.WriteString(line)
					buf.WriteRune('\n')

					// While we don't find the start
					// marker, write the line in the
					// buffer.
					if line != app.Files[file].StartMarker {
						continue
					}

					// If we find the start marker, write
					// the result to the buffer.
					buf.WriteString(result)
					buf.WriteRune('\n')

					// Then skip until the end marker.
					for scanner.Scan() {
						line = scanner.Text()

						if line != app.Files[file].EndMarker {
							continue
						}

						break
					}
					buf.WriteString(line)
					buf.WriteRune('\n')

					// And continue until the end of the
					// scanner.
				}

				if scanner.Err() != nil {
					log.Error("rewriting destination file", "err", err)
					continue
				}

				// At this point, we just replace the result
				// with the content of the buffer.
				result = buf.String()
			}

			log.Info("writing template file", "destination", destination)
			err = ioutil.WriteFile(destination, []byte(result), os.ModePerm)
			if err != nil {
				log.Error("writing destination file", "err", err)
				continue
			}
		}

		if len(app.Hook) == 0 {
			continue
		}

		log.Debug("running hook", "cmd", app.Hook)

		parts = strings.Fields(app.Hook)
		out, err := exec.Command(parts[0], parts[1:]...).Output()
		if err != nil {
			log.Error("running hook", "err", err, "out", string(out))
			continue
		}
		log.Info("running hook", "out", string(out))
	}
}

func wrap(err error, msg string, args ...interface{}) error {
	return fmt.Errorf(`%s: %w`, fmt.Sprintf(msg, args...), err)
}

func loadConfiguration() (Configuration, error) {
	var config Configuration

	// Set the defaults here so they can be omitted from the actual
	// configuration.
	config.SchemesListURL = githubFileURL("chriskempson", "base16-schemes-source", "list.yaml")
	config.TemplatesListURL = githubFileURL("chriskempson", "base16-templates-source", "list.yaml")

	raw, err := ioutil.ReadFile(xdg.New("b16m", "").QueryConfig("config.yaml"))
	if err != nil {
		return config, wrap(err, "finding configuration")
	}

	err = yaml.Unmarshal(raw, &config)
	if err != nil {
		return config, wrap(err, "parsing configuration")
	}

	return config, nil
}

func loadScheme(log log15.Logger, config Configuration) (ColorScheme, error) {
	var scheme ColorScheme

	if len(config.SchemeRepositoryURL) == 0 {
		log.Debug("retrieving schemes list", "url", config.SchemesListURL)
		var schemes map[string]string
		err := loadYAMLFile(config.SchemesListURL, &schemes)
		if err != nil {
			return scheme, wrap(err, "retrieving schemes list")
		}

		for name, url := range schemes {
			if !strings.HasPrefix(config.Scheme, name) {
				continue
			}
			config.SchemeRepositoryURL = url
		}

		if len(config.SchemeRepositoryURL) == 0 {
			return scheme, fmt.Errorf("scheme %s not found", config.Scheme)
		}
	}

	parts := strings.Split(config.SchemeRepositoryURL, "/")
	if len(parts) != 5 {
		return scheme, fmt.Errorf("unhandled scheme repository url format: %s", config.SchemeRepositoryURL)
	}

	user, repository := parts[3], parts[4]

	err := loadYAMLFile(githubFileURL(user, repository, fmt.Sprintf("%s.yaml", config.Scheme)), &scheme)
	if err != nil {
		return scheme, wrap(err, "loading file")
	}

	return scheme, nil
}

func loadFile(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, wrap(err, "retrieving list")
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, wrap(err, "reading response")
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response (status=%d body=%s)", res.StatusCode, string(body))
	}

	return body, nil
}

func loadYAMLFile(url string, dest interface{}) error {
	body, err := loadFile(url)
	if err != nil {
		return wrap(err, "loading file")
	}

	err = yaml.Unmarshal(body, dest)
	if err != nil {
		return wrap(err, "parsing file")
	}

	return nil
}

func githubFileURL(user, repository, file string) string {
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/%s", user, repository, file)
}

func expandPath(path string) string {
	if len(path) != 0 && path[0] == '~' {
		path = "$HOME" + path[1:]
	}
	return os.Expand(path, os.Getenv)
}
