/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
/**
 * Theme customization constants and types.
 *
 * Lives in `lib/` (not `context/`) so it can be imported alongside the
 * provider without breaking React Fast Refresh boundaries.
 */

export const THEME_PRESETS = [
  {
    value: 'default',
    name: 'Default',
    swatches: ['oklch(0.72 0.18 250)', 'oklch(0.7 0.12 280)'],
  },
  {
    // Inspired by Anthropic's official brand language: warm cream canvas
    // (#faf9f5) paired with clay/coral (#d97757) as the single accent.
    // Swatches preview the canvas → accent gradient that defines the system.
    value: 'anthropic',
    name: 'Anthropic',
    swatches: ['oklch(0.984 0.005 95)', 'oklch(0.685 0.142 38)'],
  },
  {
    value: 'simple-large',
    name: 'Simple Large-font',
    swatches: ['oklch(0.15 0 0)', 'oklch(0.99 0 0)'],
  },
  {
    value: 'underground',
    name: 'Underground',
    swatches: ['oklch(0.5315 0.0694 156.19)', 'oklch(0.5748 0.0862 336.52)'],
  },
  {
    value: 'rose-garden',
    name: 'Rose Garden',
    swatches: ['oklch(0.5827 0.2418 12.23)', 'oklch(0.8131 0.1129 5.67)'],
  },
  {
    value: 'lake-view',
    name: 'Lake View',
    swatches: ['oklch(0.765 0.177 163.22)', 'oklch(0.551 0.0899 200.52)'],
  },
  {
    value: 'sunset-glow',
    name: 'Sunset Glow',
    swatches: ['oklch(0.5591 0.1882 25.33)', 'oklch(0.7938 0.1248 42.42)'],
  },
  {
    value: 'forest-whisper',
    name: 'Forest Whisper',
    swatches: ['oklch(0.5276 0.1072 182.22)', 'oklch(0.5236 0.0505 250.18)'],
  },
  {
    value: 'ocean-breeze',
    name: 'Ocean Breeze',
    swatches: ['oklch(0.5461 0.2152 262.88)', 'oklch(0.5854 0.2041 277.12)'],
  },
  {
    value: 'lavender-dream',
    name: 'Lavender Dream',
    swatches: ['oklch(0.5709 0.1808 306.89)', 'oklch(0.811 0.0589 201.14)'],
  },
] as const

export type ThemePreset = (typeof THEME_PRESETS)[number]['value']
export type ThemeMode = 'dark' | 'light' | 'system'
export type ThemeRadius = 'default' | 'none' | 'sm' | 'md' | 'lg' | 'xl'
export type ThemeScale = 'default' | 'sm' | 'lg' | 'xl'
export type ContentLayout = 'full' | 'centered'
export type GlobalLayoutVariant = 'inset' | 'sidebar' | 'floating'
export type GlobalLayoutCollapsible = 'offcanvas' | 'icon' | 'none'
export type GlobalDirection = 'ltr' | 'rtl'

/**
 * Font axis for the theme.
 *
 * - `default` — resolve at runtime from the active preset
 *   (see `PRESET_DEFAULT_FONT`). The shipped `default` and `anthropic`
 *   presets resolve to serif; other named color presets fall back to
 *   sans unless they list a different choice. Mirrors how
 *   `radius: 'default'` defers to a per-preset hint.
 * - `sans` — humanist sans (Public Sans), the project's UI fallback.
 * - `serif` — editorial serif (Lora + CJK fallbacks), the project's
 *   "soul" typography. Inherits across the whole UI; monospace contexts
 *   keep their own family via Tailwind preflight and `.font-mono`.
 */
export type ThemeFont = 'default' | 'sans' | 'serif'

/**
 * The resolved (non-`default`) font value applied to the DOM. The provider
 * always sets `data-theme-font` to one of these concrete values so CSS only
 * needs simple attribute selectors (no `:not()` gymnastics, no per-preset
 * font branches).
 */
export type ResolvedThemeFont = Exclude<ThemeFont, 'default'>

export type ThemeCustomization = {
  preset: ThemePreset
  font: ThemeFont
  radius: ThemeRadius
  scale: ThemeScale
  contentLayout: ContentLayout
}

export const DEFAULT_THEME_CUSTOMIZATION: ThemeCustomization = {
  preset: 'default',
  font: 'default',
  radius: 'default',
  scale: 'default',
  contentLayout: 'full',
}

/** Theme values delivered by `/api/status` and controlled by administrators. */
export type GlobalThemeSettings = ThemeCustomization & {
  theme: ThemeMode
  layoutVariant: GlobalLayoutVariant
  layoutCollapsible: GlobalLayoutCollapsible
  direction: GlobalDirection
}

export const DEFAULT_GLOBAL_THEME_SETTINGS: GlobalThemeSettings = {
  theme: 'system',
  ...DEFAULT_THEME_CUSTOMIZATION,
  layoutVariant: 'inset',
  layoutCollapsible: 'icon',
  direction: 'ltr',
}

export const GLOBAL_THEME_MODE_VALUES: ReadonlySet<ThemeMode> = new Set([
  'dark',
  'light',
  'system',
])

export const GLOBAL_LAYOUT_VARIANT_VALUES: ReadonlySet<GlobalLayoutVariant> =
  new Set(['inset', 'sidebar', 'floating'])

export const GLOBAL_LAYOUT_COLLAPSIBLE_VALUES: ReadonlySet<GlobalLayoutCollapsible> =
  new Set(['offcanvas', 'icon', 'none'])

export const GLOBAL_DIRECTION_VALUES: ReadonlySet<GlobalDirection> = new Set([
  'ltr',
  'rtl',
])

export const THEME_PRESET_VALUES = new Set(
  THEME_PRESETS.map((p) => p.value)
) as ReadonlySet<ThemePreset>

export const THEME_FONT_VALUES: ReadonlySet<ThemeFont> = new Set([
  'default',
  'sans',
  'serif',
])

export const THEME_RADIUS_VALUES: ReadonlySet<ThemeRadius> = new Set([
  'default',
  'none',
  'sm',
  'md',
  'lg',
  'xl',
])

export const THEME_SCALE_VALUES: ReadonlySet<ThemeScale> = new Set([
  'default',
  'sm',
  'lg',
  'xl',
])

export const CONTENT_LAYOUT_VALUES: ReadonlySet<ContentLayout> = new Set([
  'full',
  'centered',
])

export const THEME_COOKIE_KEYS = {
  preset: 'theme_preset',
  font: 'theme_font',
  radius: 'theme_radius',
  scale: 'theme_scale',
  contentLayout: 'theme_content_layout',
} as const

/**
 * Preset → default font mapping. Used by the provider to resolve the user's
 * `font: 'default'` preference against the active preset.
 *
 * Co-located with the preset registry so a preset's signature typography
 * is declared in one place. Presets not listed here fall back to the
 * `resolveThemeFont` default of `sans`. The shipped `default` preset
 * opts into serif so the editorial Lora voice is the out-of-the-box
 * experience; vivid color presets stay on the humanist sans so their
 * accents read clearly without competing with the body type.
 */
export const PRESET_DEFAULT_FONT: Partial<
  Record<ThemePreset, ResolvedThemeFont>
> = {
  default: 'sans',
  anthropic: 'serif',
}

/**
 * Resolve a user font preference + active preset into the concrete font that
 * should drive the DOM. Pure function so it's safe to call inside both the
 * effect that applies the attribute and the UI preview that hints at what
 * `default` will render as.
 */
export function resolveThemeFont(
  font: ThemeFont,
  preset: ThemePreset
): ResolvedThemeFont {
  if (font === 'default') {
    return PRESET_DEFAULT_FONT[preset] ?? 'sans'
  }
  return font
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function readAllowedValue<T extends string>(
  value: unknown,
  allowed: ReadonlySet<T>,
  fallback: T
): T {
  return typeof value === 'string' && allowed.has(value as T)
    ? (value as T)
    : fallback
}

/**
 * Normalize the backend payload. This deliberately accepts unknown so a
 * stale/corrupt local status cache cannot let a user override the global
 * appearance or crash the provider.
 */
export function normalizeGlobalThemeSettings(
  value: unknown
): GlobalThemeSettings {
  if (!isRecord(value)) {
    return DEFAULT_GLOBAL_THEME_SETTINGS
  }

  return {
    theme: readAllowedValue(
      value.theme,
      GLOBAL_THEME_MODE_VALUES,
      DEFAULT_GLOBAL_THEME_SETTINGS.theme
    ),
    preset: readAllowedValue(
      value.preset,
      THEME_PRESET_VALUES,
      DEFAULT_GLOBAL_THEME_SETTINGS.preset
    ),
    font: readAllowedValue(
      value.font,
      THEME_FONT_VALUES,
      DEFAULT_GLOBAL_THEME_SETTINGS.font
    ),
    radius: readAllowedValue(
      value.radius,
      THEME_RADIUS_VALUES,
      DEFAULT_GLOBAL_THEME_SETTINGS.radius
    ),
    scale: readAllowedValue(
      value.scale,
      THEME_SCALE_VALUES,
      DEFAULT_GLOBAL_THEME_SETTINGS.scale
    ),
    contentLayout: readAllowedValue(
      value.content_layout ?? value.contentLayout,
      CONTENT_LAYOUT_VALUES,
      DEFAULT_GLOBAL_THEME_SETTINGS.contentLayout
    ),
    layoutVariant: readAllowedValue(
      value.layout_variant ?? value.layoutVariant,
      GLOBAL_LAYOUT_VARIANT_VALUES,
      DEFAULT_GLOBAL_THEME_SETTINGS.layoutVariant
    ),
    layoutCollapsible: readAllowedValue(
      value.layout_collapsible ?? value.layoutCollapsible,
      GLOBAL_LAYOUT_COLLAPSIBLE_VALUES,
      DEFAULT_GLOBAL_THEME_SETTINGS.layoutCollapsible
    ),
    direction: readAllowedValue(
      value.direction,
      GLOBAL_DIRECTION_VALUES,
      DEFAULT_GLOBAL_THEME_SETTINGS.direction
    ),
  }
}
