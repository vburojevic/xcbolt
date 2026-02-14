package core

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type PlatformFamily string

const (
	PlatformUnknown  PlatformFamily = ""
	PlatformIOS      PlatformFamily = "ios"
	PlatformIPadOS   PlatformFamily = "ipados"
	PlatformTvOS     PlatformFamily = "tvos"
	PlatformVisionOS PlatformFamily = "visionos"
	PlatformWatchOS  PlatformFamily = "watchos"
	PlatformMacOS    PlatformFamily = "macos"
	PlatformCatalyst PlatformFamily = "catalyst"
)

type TargetType string

const (
	TargetAuto      TargetType = "auto"
	TargetSimulator TargetType = "simulator"
	TargetDevice    TargetType = "device"
	TargetLocal     TargetType = "local"
)

type DestinationCandidate struct {
	ID             string
	Name           string
	PlatformFamily PlatformFamily
	TargetType     TargetType
	Platform       string
	OSVersion      string
	RuntimeName    string
	RuntimeID      string
	State          string
	Available      bool
}

func NormalizePlatformFamily(v string) PlatformFamily {
	s := strings.ToLower(strings.TrimSpace(v))
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")
	switch s {
	case "ios", "iphoneos":
		return PlatformIOS
	case "ipados", "ipadair", "ipad":
		return PlatformIPadOS
	case "tvos", "appletv":
		return PlatformTvOS
	case "visionos", "xros", "xr":
		return PlatformVisionOS
	case "watchos", "watch":
		return PlatformWatchOS
	case "macos", "mac":
		return PlatformMacOS
	case "catalyst", "maccatalyst":
		return PlatformCatalyst
	default:
		return PlatformUnknown
	}
}

func NormalizeTargetType(v string) TargetType {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "", "auto":
		return TargetAuto
	case "sim", "simulator":
		return TargetSimulator
	case "dev", "device":
		return TargetDevice
	case "local", "mac", "host":
		return TargetLocal
	default:
		return TargetAuto
	}
}

func InferPlatformFamilyFromRuntime(runtimeID, runtimeName, deviceName string) PlatformFamily {
	all := strings.ToLower(runtimeID + " " + runtimeName)
	if strings.Contains(all, "watch") {
		return PlatformWatchOS
	}
	if strings.Contains(all, "tvos") || strings.Contains(all, "apple tv") {
		return PlatformTvOS
	}
	if strings.Contains(all, "xros") || strings.Contains(all, "vision") {
		return PlatformVisionOS
	}
	if strings.Contains(all, "ios") {
		if strings.Contains(strings.ToLower(deviceName), "ipad") {
			return PlatformIPadOS
		}
		return PlatformIOS
	}
	return PlatformUnknown
}

func InferPlatformFamilyFromDevice(platform, model, name string) PlatformFamily {
	all := strings.ToLower(strings.TrimSpace(platform + " " + model + " " + name))
	if strings.Contains(all, "watch") {
		return PlatformWatchOS
	}
	if strings.Contains(all, "tvos") || strings.Contains(all, "apple tv") {
		return PlatformTvOS
	}
	if strings.Contains(all, "vision") || strings.Contains(all, "xros") {
		return PlatformVisionOS
	}
	if strings.Contains(all, "ipad") {
		return PlatformIPadOS
	}
	if strings.Contains(all, "ios") || strings.Contains(all, "iphone") {
		return PlatformIOS
	}
	if strings.Contains(all, "catalyst") {
		return PlatformCatalyst
	}
	if strings.Contains(all, "mac") {
		return PlatformMacOS
	}
	return PlatformUnknown
}

func PlatformStringForDestination(family PlatformFamily, targetType TargetType) string {
	switch targetType {
	case TargetSimulator:
		switch family {
		case PlatformIOS, PlatformIPadOS:
			return "iOS Simulator"
		case PlatformTvOS:
			return "tvOS Simulator"
		case PlatformVisionOS:
			return "visionOS Simulator"
		case PlatformWatchOS:
			return "watchOS Simulator"
		}
	case TargetDevice:
		switch family {
		case PlatformIOS, PlatformIPadOS:
			return "iOS"
		case PlatformTvOS:
			return "tvOS"
		case PlatformVisionOS:
			return "visionOS"
		case PlatformWatchOS:
			return "watchOS"
		}
	case TargetLocal:
		switch family {
		case PlatformCatalyst:
			return "macOS"
		case PlatformMacOS:
			return "macOS"
		}
	}
	return ""
}

func destinationKindFromTargetType(tt TargetType, family PlatformFamily) DestinationKind {
	switch tt {
	case TargetSimulator:
		return DestSimulator
	case TargetDevice:
		return DestDevice
	case TargetLocal:
		if family == PlatformCatalyst {
			return DestCatalyst
		}
		return DestMacOS
	default:
		return DestAuto
	}
}

func syncDestinationLegacy(dst *Destination) {
	if dst.TargetType == "" {
		switch dst.Kind {
		case DestSimulator:
			dst.TargetType = TargetSimulator
		case DestDevice:
			dst.TargetType = TargetDevice
		case DestMacOS, DestCatalyst:
			dst.TargetType = TargetLocal
		default:
			dst.TargetType = TargetAuto
		}
	}
	if dst.PlatformFamily == "" {
		switch dst.Kind {
		case DestMacOS:
			dst.PlatformFamily = PlatformMacOS
		case DestCatalyst:
			dst.PlatformFamily = PlatformCatalyst
		}
	}
	if dst.PlatformFamily == "" {
		dst.PlatformFamily = NormalizePlatformFamily(dst.Platform)
	}
	if dst.ID == "" {
		dst.ID = strings.TrimSpace(dst.UDID)
	}
	if dst.UDID == "" {
		dst.UDID = strings.TrimSpace(dst.ID)
	}
	if dst.Kind == DestAuto {
		dst.Kind = destinationKindFromTargetType(dst.TargetType, dst.PlatformFamily)
	}
	if dst.TargetType != TargetAuto {
		dst.Kind = destinationKindFromTargetType(dst.TargetType, dst.PlatformFamily)
	}
	if dst.Platform == "" {
		dst.Platform = PlatformStringForDestination(dst.PlatformFamily, dst.TargetType)
	}
	if dst.TargetType == TargetLocal {
		dst.ID = ""
		dst.UDID = ""
	}
}

func normalizeDestination(dst Destination) Destination {
	syncDestinationLegacy(&dst)
	return dst
}

func ListDestinationCandidates(ctx context.Context, emit Emitter) ([]DestinationCandidate, error) {
	out := []DestinationCandidate{}

	list, err := SimctlList(ctx, emit)
	if err == nil {
		sims := FlattenSimulators(list)
		for _, s := range sims {
			family := s.PlatformFamily
			if family == "" {
				family = InferPlatformFamilyFromRuntime(s.RuntimeID, s.RuntimeName, s.Name)
			}
			plat := PlatformStringForDestination(family, TargetSimulator)
			if family == "" || plat == "" {
				continue
			}
			out = append(out, DestinationCandidate{
				ID:             s.UDID,
				Name:           s.Name,
				PlatformFamily: family,
				TargetType:     TargetSimulator,
				Platform:       plat,
				OSVersion:      s.OSVersion,
				RuntimeName:    s.RuntimeName,
				RuntimeID:      s.RuntimeID,
				State:          s.State,
				Available:      s.Available,
			})
		}
	}

	if DevicectlAvailable(ctx) {
		if devs, derr := DevicectlList(ctx, emit); derr == nil {
			for _, d := range devs {
				family := d.PlatformFamily
				if family == "" {
					family = InferPlatformFamilyFromDevice(d.Platform, d.Model, d.Name)
				}
				plat := PlatformStringForDestination(family, TargetDevice)
				if family == "" || plat == "" {
					continue
				}
				out = append(out, DestinationCandidate{
					ID:             d.Identifier,
					Name:           d.Name,
					PlatformFamily: family,
					TargetType:     TargetDevice,
					Platform:       plat,
					OSVersion:      d.OSVersion,
					Available:      true,
				})
			}
		}
	}

	// Always expose local Mac targets.
	out = append(out,
		DestinationCandidate{ID: "macos", Name: "My Mac", PlatformFamily: PlatformMacOS, TargetType: TargetLocal, Platform: "macOS", Available: true},
		DestinationCandidate{ID: "catalyst", Name: "My Mac (Catalyst)", PlatformFamily: PlatformCatalyst, TargetType: TargetLocal, Platform: "macOS", Available: true},
	)

	return out, nil
}

func familyPriority(f PlatformFamily) int {
	switch f {
	case PlatformIOS:
		return 0
	case PlatformIPadOS:
		return 1
	case PlatformTvOS:
		return 2
	case PlatformVisionOS:
		return 3
	case PlatformWatchOS:
		return 4
	case PlatformMacOS:
		return 5
	case PlatformCatalyst:
		return 6
	default:
		return 100
	}
}

func candidateScore(c DestinationCandidate) int {
	score := 0
	if c.TargetType == TargetSimulator {
		score += 100
		if strings.EqualFold(c.State, "booted") {
			score += 20
		}
		if c.Available {
			score += 10
		}
	}
	if c.TargetType == TargetDevice {
		score += 50
	}
	if c.TargetType == TargetLocal {
		score += 10
	}
	score -= familyPriority(c.PlatformFamily)
	return score
}

func resolveCandidateByTarget(candidates []DestinationCandidate, target string) ([]DestinationCandidate, []DestinationCandidate) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, nil
	}
	exactID := []DestinationCandidate{}
	exactName := []DestinationCandidate{}
	for _, c := range candidates {
		if strings.EqualFold(c.ID, target) {
			exactID = append(exactID, c)
		}
		if strings.EqualFold(c.Name, target) {
			exactName = append(exactName, c)
		}
	}
	return exactID, exactName
}

func destinationFromCandidate(dst *Destination, c DestinationCandidate) {
	dst.TargetType = c.TargetType
	dst.PlatformFamily = c.PlatformFamily
	dst.Kind = destinationKindFromTargetType(c.TargetType, c.PlatformFamily)
	dst.ID = c.ID
	dst.UDID = c.ID
	dst.Name = c.Name
	dst.Platform = c.Platform
	if c.OSVersion != "" {
		dst.OS = c.OSVersion
	}
	if c.RuntimeID != "" {
		dst.RuntimeID = c.RuntimeID
	}
}

func resolveDestination(cfg Config, candidates []DestinationCandidate) (Config, error) {
	dst := normalizeDestination(cfg.Destination)

	if dst.TargetType == TargetLocal {
		if dst.PlatformFamily == "" {
			dst.PlatformFamily = PlatformMacOS
		}
		dst.Kind = destinationKindFromTargetType(TargetLocal, dst.PlatformFamily)
		dst.Platform = PlatformStringForDestination(dst.PlatformFamily, TargetLocal)
		dst.Name = "My Mac"
		if dst.PlatformFamily == PlatformCatalyst {
			dst.Name = "My Mac (Catalyst)"
		}
		dst.ID = ""
		dst.UDID = ""
		cfg.Destination = dst
		return cfg, nil
	}

	filtered := make([]DestinationCandidate, 0, len(candidates))
	for _, c := range candidates {
		if c.TargetType != TargetLocal && !c.Available {
			continue
		}
		if dst.PlatformFamily != "" && c.PlatformFamily != dst.PlatformFamily {
			continue
		}
		if dst.TargetType != "" && dst.TargetType != TargetAuto && c.TargetType != dst.TargetType {
			continue
		}
		if c.TargetType == TargetLocal && dst.TargetType != TargetLocal && dst.TargetType != TargetAuto {
			continue
		}
		filtered = append(filtered, c)
	}

	if len(filtered) == 0 && (dst.PlatformFamily == PlatformMacOS || dst.PlatformFamily == PlatformCatalyst) {
		dst.TargetType = TargetLocal
		dst.Kind = destinationKindFromTargetType(TargetLocal, dst.PlatformFamily)
		dst.Platform = "macOS"
		dst.Name = "My Mac"
		if dst.PlatformFamily == PlatformCatalyst {
			dst.Name = "My Mac (Catalyst)"
		}
		cfg.Destination = dst
		return cfg, nil
	}

	if t := strings.TrimSpace(dst.ID); t != "" {
		ids, names := resolveCandidateByTarget(filtered, t)
		if len(ids) == 1 {
			destinationFromCandidate(&dst, ids[0])
			cfg.Destination = dst
			return cfg, nil
		}
		if len(ids) > 1 {
			return cfg, fmt.Errorf("destination id %q is ambiguous", t)
		}
		if len(names) == 1 {
			destinationFromCandidate(&dst, names[0])
			cfg.Destination = dst
			return cfg, nil
		}
		if len(names) > 1 {
			return cfg, fmt.Errorf("destination name %q is ambiguous", t)
		}
		return cfg, fmt.Errorf("destination target %q was not found", t)
	}
	if t := strings.TrimSpace(dst.Name); t != "" {
		ids, names := resolveCandidateByTarget(filtered, t)
		if len(ids) == 1 {
			destinationFromCandidate(&dst, ids[0])
			cfg.Destination = dst
			return cfg, nil
		}
		if len(names) == 1 {
			destinationFromCandidate(&dst, names[0])
			cfg.Destination = dst
			return cfg, nil
		}
		if len(ids)+len(names) > 1 {
			return cfg, fmt.Errorf("destination name %q is ambiguous", t)
		}
		return cfg, fmt.Errorf("destination target %q was not found", t)
	}

	// Auto-pick when destination isn't explicit.
	if dst.TargetType == "" || dst.TargetType == TargetAuto || dst.Kind == DestAuto || dst.ID == "" {
		if len(filtered) == 0 {
			return cfg, errors.New("no destinations available for the selected platform/target type")
		}
		sort.SliceStable(filtered, func(i, j int) bool {
			si := candidateScore(filtered[i])
			sj := candidateScore(filtered[j])
			if si == sj {
				if filtered[i].OSVersion == filtered[j].OSVersion {
					return filtered[i].Name < filtered[j].Name
				}
				return filtered[i].OSVersion > filtered[j].OSVersion
			}
			return si > sj
		})
		destinationFromCandidate(&dst, filtered[0])
		cfg.Destination = dst
		return cfg, nil
	}

	return cfg, errors.New("unable to resolve destination")
}

// ResolveDestinationIfNeeded resolves and normalizes config destination for build/test/run.
func ResolveDestinationIfNeeded(ctx context.Context, projectRoot string, cfg Config, emit Emitter) (Config, error) {
	_ = projectRoot
	candidates, err := ListDestinationCandidates(ctx, emit)
	if err != nil {
		return cfg, err
	}
	cfg2, err := resolveDestination(cfg, candidates)
	if err != nil {
		return cfg, err
	}
	return cfg2, nil
}

func destinationMetadata(dst Destination) map[string]any {
	dst = normalizeDestination(dst)
	return map[string]any{
		"platformFamily":      string(dst.PlatformFamily),
		"targetType":          string(dst.TargetType),
		"targetId":            dst.ID,
		"resolvedDestination": dst.Platform,
		"companionTargetId":   dst.CompanionTargetID,
		"companionBundleId":   dst.CompanionBundleID,
	}
}
