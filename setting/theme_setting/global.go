package theme_setting

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// OptionKey is the administrator-controlled option published to every client.
const OptionKey = "theme.global"

// GlobalThemeSettings contains the appearance axes that are shared by every
// frontend user. Keeping this as one JSON option makes a theme change atomic
// and lets older installations safely fall back to the defaults.
type GlobalThemeSettings struct {
	Theme             string `json:"theme"`
	Preset            string `json:"preset"`
	Font              string `json:"font"`
	Radius            string `json:"radius"`
	Scale             string `json:"scale"`
	ContentLayout     string `json:"content_layout"`
	LayoutVariant     string `json:"layout_variant"`
	LayoutCollapsible string `json:"layout_collapsible"`
	Direction         string `json:"direction"`
}

var Default = GlobalThemeSettings{
	Theme:             "system",
	Preset:            "default",
	Font:              "default",
	Radius:            "default",
	Scale:             "default",
	ContentLayout:     "full",
	LayoutVariant:     "inset",
	LayoutCollapsible: "icon",
	Direction:         "ltr",
}

var validThemes = map[string]struct{}{
	"light":  {},
	"dark":   {},
	"system": {},
}

var validPresets = map[string]struct{}{
	"default":        {},
	"anthropic":      {},
	"simple-large":   {},
	"underground":    {},
	"rose-garden":    {},
	"lake-view":      {},
	"sunset-glow":    {},
	"forest-whisper": {},
	"ocean-breeze":   {},
	"lavender-dream": {},
}

var validFonts = map[string]struct{}{
	"default": {},
	"sans":    {},
	"serif":   {},
}

var validRadii = map[string]struct{}{
	"default": {},
	"none":    {},
	"sm":      {},
	"md":      {},
	"lg":      {},
	"xl":      {},
}

var validScales = map[string]struct{}{
	"default": {},
	"sm":      {},
	"lg":      {},
	"xl":      {},
}

var validLayouts = map[string]struct{}{
	"full":     {},
	"centered": {},
}

var validLayoutVariants = map[string]struct{}{
	"inset":    {},
	"sidebar":  {},
	"floating": {},
}

var validLayoutCollapsibles = map[string]struct{}{
	"offcanvas": {},
	"icon":      {},
	"none":      {},
}

var validDirections = map[string]struct{}{
	"ltr": {},
	"rtl": {},
}

// Validate checks both the JSON shape and the finite set of values accepted by
// the frontend. Unknown fields are intentionally ignored for forward
// compatibility, while missing fields are filled from Default.
func Validate(value string) error {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return fmt.Errorf("global theme must be a JSON object")
	}
	var settings GlobalThemeSettings
	if err := common.UnmarshalJsonStr(trimmed, &settings); err != nil {
		return fmt.Errorf("global theme must be a JSON object: %w", err)
	}
	settings = normalize(settings)
	if _, ok := validThemes[settings.Theme]; !ok {
		return fmt.Errorf("invalid global theme mode %q", settings.Theme)
	}
	if _, ok := validPresets[settings.Preset]; !ok {
		return fmt.Errorf("invalid global theme preset %q", settings.Preset)
	}
	if _, ok := validFonts[settings.Font]; !ok {
		return fmt.Errorf("invalid global theme font %q", settings.Font)
	}
	if _, ok := validRadii[settings.Radius]; !ok {
		return fmt.Errorf("invalid global theme radius %q", settings.Radius)
	}
	if _, ok := validScales[settings.Scale]; !ok {
		return fmt.Errorf("invalid global theme scale %q", settings.Scale)
	}
	if _, ok := validLayouts[settings.ContentLayout]; !ok {
		return fmt.Errorf("invalid global theme content layout %q", settings.ContentLayout)
	}
	if _, ok := validLayoutVariants[settings.LayoutVariant]; !ok {
		return fmt.Errorf("invalid global theme layout variant %q", settings.LayoutVariant)
	}
	if _, ok := validLayoutCollapsibles[settings.LayoutCollapsible]; !ok {
		return fmt.Errorf("invalid global theme layout collapsible mode %q", settings.LayoutCollapsible)
	}
	if _, ok := validDirections[settings.Direction]; !ok {
		return fmt.Errorf("invalid global theme direction %q", settings.Direction)
	}
	return nil
}

// Parse returns a normalized settings object. Invalid persisted values are
// treated as an installation-wide default instead of breaking /api/status.
func Parse(value string) GlobalThemeSettings {
	var settings GlobalThemeSettings
	if err := common.UnmarshalJsonStr(strings.TrimSpace(value), &settings); err != nil {
		return Default
	}
	settings = normalize(settings)
	if ValidateNormalized(settings) != nil {
		return Default
	}
	return settings
}

// ValidateNormalized validates a struct without decoding JSON again.
func ValidateNormalized(settings GlobalThemeSettings) error {
	if _, ok := validThemes[settings.Theme]; !ok {
		return fmt.Errorf("invalid global theme mode %q", settings.Theme)
	}
	if _, ok := validPresets[settings.Preset]; !ok {
		return fmt.Errorf("invalid global theme preset %q", settings.Preset)
	}
	if _, ok := validFonts[settings.Font]; !ok {
		return fmt.Errorf("invalid global theme font %q", settings.Font)
	}
	if _, ok := validRadii[settings.Radius]; !ok {
		return fmt.Errorf("invalid global theme radius %q", settings.Radius)
	}
	if _, ok := validScales[settings.Scale]; !ok {
		return fmt.Errorf("invalid global theme scale %q", settings.Scale)
	}
	if _, ok := validLayouts[settings.ContentLayout]; !ok {
		return fmt.Errorf("invalid global theme content layout %q", settings.ContentLayout)
	}
	if _, ok := validLayoutVariants[settings.LayoutVariant]; !ok {
		return fmt.Errorf("invalid global theme layout variant %q", settings.LayoutVariant)
	}
	if _, ok := validLayoutCollapsibles[settings.LayoutCollapsible]; !ok {
		return fmt.Errorf("invalid global theme layout collapsible mode %q", settings.LayoutCollapsible)
	}
	if _, ok := validDirections[settings.Direction]; !ok {
		return fmt.Errorf("invalid global theme direction %q", settings.Direction)
	}
	return nil
}

func normalize(settings GlobalThemeSettings) GlobalThemeSettings {
	if settings.Theme == "" {
		settings.Theme = Default.Theme
	}
	if settings.Preset == "" {
		settings.Preset = Default.Preset
	}
	if settings.Font == "" {
		settings.Font = Default.Font
	}
	if settings.Radius == "" {
		settings.Radius = Default.Radius
	}
	if settings.Scale == "" {
		settings.Scale = Default.Scale
	}
	if settings.ContentLayout == "" {
		settings.ContentLayout = Default.ContentLayout
	}
	if settings.LayoutVariant == "" {
		settings.LayoutVariant = Default.LayoutVariant
	}
	if settings.LayoutCollapsible == "" {
		settings.LayoutCollapsible = Default.LayoutCollapsible
	}
	if settings.Direction == "" {
		settings.Direction = Default.Direction
	}
	return settings
}

// DefaultJSON is used to seed the option map on fresh installations.
func DefaultJSON() string {
	encoded, err := common.Marshal(Default)
	if err != nil {
		return `{"theme":"system","preset":"default","font":"default","radius":"default","scale":"default","content_layout":"full","layout_variant":"inset","layout_collapsible":"icon","direction":"ltr"}`
	}
	return string(encoded)
}
