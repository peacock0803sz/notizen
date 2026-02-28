package tmpl

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates
var embeddedTemplates embed.FS

// Load returns a parsed template by name (e.g. "diary/entry.md.tmpl").
// Filesystem override at ~/.config/notizen/templates/ takes precedence over embedded defaults.
func Load(name string) (*template.Template, error) {
	overridePath := filepath.Join(configTemplateDir(), name)
	if _, err := os.Stat(overridePath); err == nil {
		return template.ParseFiles(overridePath)
	}

	embeddedPath := filepath.Join("templates", name)
	t, err := template.ParseFS(embeddedTemplates, embeddedPath)
	if err != nil {
		return nil, fmt.Errorf("template %q not found: %w", name, err)
	}
	return t, nil
}

func configTemplateDir() string {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, _ := os.UserHomeDir()
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "notizen", "templates")
}
