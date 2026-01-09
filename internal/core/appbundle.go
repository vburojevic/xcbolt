package core

import (
	"fmt"
	"os"
	"path/filepath"

	"howett.net/plist"
)

type AppBundleInfo struct {
	BundleID     string
	DisplayName  string
	BundleName   string
	Executable   string
	Version      string
	BuildVersion string
}

func ReadAppBundleInfo(appPath string) (AppBundleInfo, error) {
	infoPlist := filepath.Join(appPath, "Info.plist")
	b, err := os.ReadFile(infoPlist)
	if err != nil {
		return AppBundleInfo{}, fmt.Errorf("read Info.plist: %w", err)
	}
	var m map[string]any
	_, err = plist.Unmarshal(b, &m)
	if err != nil {
		return AppBundleInfo{}, fmt.Errorf("parse Info.plist: %w", err)
	}
	get := func(key string) string {
		if v, ok := m[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	return AppBundleInfo{
		BundleID:     get("CFBundleIdentifier"),
		DisplayName:  get("CFBundleDisplayName"),
		BundleName:   get("CFBundleName"),
		Executable:   get("CFBundleExecutable"),
		Version:      get("CFBundleShortVersionString"),
		BuildVersion: get("CFBundleVersion"),
	}, nil
}
