package theme_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateAcceptsCompleteGlobalTheme(t *testing.T) {
	require.NoError(t, Validate(DefaultJSON()))
}

func TestValidateRejectsUnknownPreset(t *testing.T) {
	require.ErrorContains(t, Validate(`{"theme":"dark","preset":"unknown"}`), "invalid global theme preset")
}

func TestValidateRejectsNonObjectJSON(t *testing.T) {
	require.Error(t, Validate(`null`))
	require.Error(t, Validate(`[]`))
}

func TestParseFallsBackForInvalidPersistedValue(t *testing.T) {
	require.Equal(t, Default, Parse(`{"theme":"dark","preset":"unknown"}`))
}

func TestParseBackfillsNewLayoutFieldsForLegacyOption(t *testing.T) {
	settings := Parse(`{"theme":"dark","preset":"default"}`)
	require.Equal(t, "dark", settings.Theme)
	require.Equal(t, Default.LayoutVariant, settings.LayoutVariant)
	require.Equal(t, Default.LayoutCollapsible, settings.LayoutCollapsible)
	require.Equal(t, Default.Direction, settings.Direction)
}
