package core

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func listSchemesFromDir(dir string, out *[]string, seen map[string]struct{}) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".xcscheme") {
			continue
		}
		scheme := strings.TrimSuffix(name, ".xcscheme")
		if scheme == "" {
			continue
		}
		if _, ok := seen[scheme]; ok {
			continue
		}
		seen[scheme] = struct{}{}
		*out = append(*out, scheme)
	}
}

func workspaceProjectPaths(projectRoot, workspaceRel string) []string {
	workspacePath := absJoin(projectRoot, workspaceRel)
	dataPath := filepath.Join(workspacePath, "contents.xcworkspacedata")
	b, err := os.ReadFile(dataPath)
	if err != nil {
		return nil
	}

	// Match FileRef locations like: group:Index.xcodeproj
	locRE := regexp.MustCompile(`location="([^"]+)"`)
	matches := locRE.FindAllStringSubmatch(string(b), -1)
	if len(matches) == 0 {
		return nil
	}

	paths := make([]string, 0, len(matches))
	for _, m := range matches {
		loc := m[1]
		parts := strings.SplitN(loc, ":", 2)
		if len(parts) != 2 {
			continue
		}
		kind := parts[0]
		path := parts[1]
		if !strings.HasSuffix(path, ".xcodeproj") {
			continue
		}
		var fullPath string
		switch kind {
		case "absolute":
			fullPath = path
		case "group", "container":
			fullPath = filepath.Join(workspacePath, path)
		default:
			fullPath = filepath.Join(workspacePath, path)
		}
		fullPath = filepath.Clean(fullPath)
		paths = append(paths, fullPath)
	}
	return paths
}

func listSchemesFromFS(projectRoot string, cfg Config, projectsFromRoot []string) []string {
	seen := map[string]struct{}{}
	var schemes []string

	if cfg.Workspace != "" {
		workspacePath := absJoin(projectRoot, cfg.Workspace)
		listSchemesFromDir(filepath.Join(workspacePath, "xcshareddata", "xcschemes"), &schemes, seen)
		userDir := filepath.Join(workspacePath, "xcuserdata")
		if entries, err := os.ReadDir(userDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				listSchemesFromDir(filepath.Join(userDir, entry.Name(), "xcschemes"), &schemes, seen)
			}
		}

		if len(schemes) == 0 {
			for _, projPath := range workspaceProjectPaths(projectRoot, cfg.Workspace) {
				listSchemesFromDir(filepath.Join(projPath, "xcshareddata", "xcschemes"), &schemes, seen)
				userDir := filepath.Join(projPath, "xcuserdata")
				if entries, err := os.ReadDir(userDir); err == nil {
					for _, entry := range entries {
						if !entry.IsDir() {
							continue
						}
						listSchemesFromDir(filepath.Join(userDir, entry.Name(), "xcschemes"), &schemes, seen)
					}
				}
			}
		}
	}

	if cfg.Project != "" {
		projectPath := absJoin(projectRoot, cfg.Project)
		listSchemesFromDir(filepath.Join(projectPath, "xcshareddata", "xcschemes"), &schemes, seen)
		userDir := filepath.Join(projectPath, "xcuserdata")
		if entries, err := os.ReadDir(userDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				listSchemesFromDir(filepath.Join(userDir, entry.Name(), "xcschemes"), &schemes, seen)
			}
		}
	}

	if len(schemes) == 0 {
		for _, proj := range projectsFromRoot {
			projectPath := absJoin(projectRoot, proj)
			listSchemesFromDir(filepath.Join(projectPath, "xcshareddata", "xcschemes"), &schemes, seen)
		}
	}

	sort.Strings(schemes)
	return schemes
}

func parseConfigurationsFromPBXProj(path string, out *[]string, seen map[string]struct{}) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	inSection := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Begin XCBuildConfiguration section") {
			inSection = true
			continue
		}
		if strings.Contains(line, "End XCBuildConfiguration section") {
			inSection = false
			continue
		}
		if !inSection {
			continue
		}
		trim := strings.TrimSpace(line)
		if !strings.HasPrefix(trim, "name =") {
			continue
		}
		value := strings.TrimPrefix(trim, "name =")
		value = strings.TrimSpace(value)
		value = strings.TrimSuffix(value, ";")
		value = strings.Trim(value, "\"")
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		*out = append(*out, value)
	}
}

func listConfigurationsFromPBXProj(projectRoot string, cfg Config, projectsFromRoot []string) []string {
	seen := map[string]struct{}{}
	var configs []string

	addProject := func(projectPath string) {
		pbxproj := filepath.Join(projectPath, "project.pbxproj")
		parseConfigurationsFromPBXProj(pbxproj, &configs, seen)
	}

	if cfg.Project != "" {
		addProject(absJoin(projectRoot, cfg.Project))
	}

	if cfg.Workspace != "" {
		paths := workspaceProjectPaths(projectRoot, cfg.Workspace)
		for _, projPath := range paths {
			addProject(projPath)
		}
		if len(paths) == 0 && len(projectsFromRoot) == 1 {
			addProject(absJoin(projectRoot, projectsFromRoot[0]))
		}
	}

	if len(configs) == 0 {
		for _, proj := range projectsFromRoot {
			addProject(absJoin(projectRoot, proj))
		}
	}

	sort.Strings(configs)
	return configs
}

func scanProjectEntries(projectRoot string) ([]string, []string, error) {
	entries, err := os.ReadDir(projectRoot)
	if err != nil {
		return nil, nil, err
	}
	workspaces := []string{}
	projects := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() && strings.HasSuffix(name, ".xcworkspace") {
			workspaces = append(workspaces, name)
		}
		if entry.IsDir() && strings.HasSuffix(name, ".xcodeproj") {
			projects = append(projects, name)
		}
	}
	return workspaces, projects, nil
}

func ensureSchemeAndConfigFromFS(projectRoot string, cfg Config, emit Emitter) (Config, error) {
	workspaces, projects, err := scanProjectEntries(projectRoot)
	if err != nil {
		return cfg, err
	}

	// Auto-pick workspace/project if unset.
	if cfg.Workspace == "" && len(workspaces) == 1 {
		cfg.Workspace = workspaces[0]
	}
	if cfg.Project == "" && cfg.Workspace == "" && len(projects) == 1 {
		cfg.Project = projects[0]
	}

	schemes := listSchemesFromFS(projectRoot, cfg, projects)
	configurations := listConfigurationsFromPBXProj(projectRoot, cfg, projects)

	if len(schemes) > 0 {
		if cfg.Scheme == "" || !stringInSlice(cfg.Scheme, schemes) {
			cfg.Scheme = schemes[0]
			emitMaybe(emit, Status("context", "Auto-selected scheme", map[string]any{"scheme": cfg.Scheme}))
		}
	}
	if len(configurations) > 0 {
		if cfg.Configuration == "" || !stringInSlice(cfg.Configuration, configurations) {
			cfg.Configuration = configurations[0]
			emitMaybe(emit, Status("context", "Auto-selected configuration", map[string]any{"configuration": cfg.Configuration}))
		}
	}
	if cfg.Scheme == "" {
		return cfg, errors.New("no scheme configured (run `xcbolt init` or pass --scheme)")
	}
	return cfg, nil
}

func stringInSlice(needle string, haystack []string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}
