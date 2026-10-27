package config

import (
	"strings"

	"golang.org/x/sys/windows"
)

func systemLanguage() string {
	langs, err := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME)
	if err == nil && len(langs) > 0 && strings.HasPrefix(strings.ToLower(langs[0]), "ru") {
		return "ru"
	}
	return "en"
}
