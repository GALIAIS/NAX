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
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  DEFAULT_GLOBAL_THEME_SETTINGS,
  normalizeGlobalThemeSettings,
  THEME_PRESETS,
  type GlobalThemeSettings,
  type GlobalDirection,
  type GlobalLayoutCollapsible,
  type GlobalLayoutVariant,
  type ThemeFont,
  type ThemeMode,
  type ThemeRadius,
  type ThemeScale,
} from '@/lib/theme-customization'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

type GlobalThemeSectionProps = {
  defaultValue: string
}

function parseThemeSettings(value: string): GlobalThemeSettings {
  try {
    return normalizeGlobalThemeSettings(JSON.parse(value) as unknown)
  } catch {
    return DEFAULT_GLOBAL_THEME_SETTINGS
  }
}

function serializeThemeSettings(settings: GlobalThemeSettings): string {
  return JSON.stringify({
    theme: settings.theme,
    preset: settings.preset,
    font: settings.font,
    radius: settings.radius,
    scale: settings.scale,
    content_layout: settings.contentLayout,
    layout_variant: settings.layoutVariant,
    layout_collapsible: settings.layoutCollapsible,
    direction: settings.direction,
  })
}

function updateThemeSetting<K extends keyof GlobalThemeSettings>(
  settings: GlobalThemeSettings,
  key: K,
  value: GlobalThemeSettings[K]
): GlobalThemeSettings {
  return { ...settings, [key]: value }
}

const MODE_OPTIONS: Array<{ value: ThemeMode; label: string }> = [
  { value: 'system', label: 'System' },
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
]

const FONT_OPTIONS: Array<{ value: ThemeFont; label: string }> = [
  { value: 'default', label: 'Auto' },
  { value: 'sans', label: 'Sans' },
  { value: 'serif', label: 'Serif' },
]

const RADIUS_OPTIONS: Array<{ value: ThemeRadius; label: string }> = [
  { value: 'default', label: 'Auto' },
  { value: 'none', label: 'None' },
  { value: 'sm', label: 'Small' },
  { value: 'md', label: 'Medium' },
  { value: 'lg', label: 'Large' },
  { value: 'xl', label: 'Extra large' },
]

const SCALE_OPTIONS: Array<{ value: ThemeScale; label: string }> = [
  { value: 'default', label: 'Default' },
  { value: 'sm', label: 'Compact' },
  { value: 'lg', label: 'Comfortable' },
  { value: 'xl', label: 'Extra large' },
]

const LAYOUT_VARIANT_OPTIONS: Array<{
  value: GlobalLayoutVariant
  label: string
}> = [
  { value: 'inset', label: 'Inset' },
  { value: 'sidebar', label: 'Sidebar' },
  { value: 'floating', label: 'Floating' },
]

const LAYOUT_COLLAPSIBLE_OPTIONS: Array<{
  value: GlobalLayoutCollapsible
  label: string
}> = [
  { value: 'offcanvas', label: 'Full layout' },
  { value: 'icon', label: 'Compact' },
  { value: 'none', label: 'None' },
]

const DIRECTION_OPTIONS: Array<{ value: GlobalDirection; label: string }> = [
  { value: 'ltr', label: 'Left to Right' },
  { value: 'rtl', label: 'Right to Left' },
]

export function GlobalThemeSection(props: GlobalThemeSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const setSystemConfig = useSystemConfigStore((state) => state.setConfig)
  const [settings, setSettings] = useState(() =>
    parseThemeSettings(props.defaultValue)
  )
  const [savedValue, setSavedValue] = useState(() =>
    serializeThemeSettings(parseThemeSettings(props.defaultValue))
  )

  useEffect(() => {
    const next = parseThemeSettings(props.defaultValue)
    setSettings(next)
    setSavedValue(serializeThemeSettings(next))
  }, [props.defaultValue])

  const serialized = useMemo(() => serializeThemeSettings(settings), [settings])
  const isDirty = serialized !== savedValue

  const setValue = <K extends keyof GlobalThemeSettings>(
    key: K,
    value: GlobalThemeSettings[K]
  ) => {
    setSettings((current) => updateThemeSetting(current, key, value))
  }

  const save = async () => {
    const result = await updateOption.mutateAsync({
      key: 'theme.global',
      value: serialized,
    })
    if (result.success) {
      setSavedValue(serialized)
      setSystemConfig({ globalTheme: settings })
    }
  }

  return (
    <SettingsSection title={t('Global Theme')}>
      <SettingsForm
        onSubmit={(event) => {
          event.preventDefault()
          void save()
        }}
      >
        <SettingsPageFormActions
          onSave={() => void save()}
          isSaving={updateOption.isPending}
          isSaveDisabled={!isDirty}
        />

        <div className='bg-muted/20 space-y-2 rounded-lg border p-3 lg:col-span-2'>
          <p className='text-sm font-medium'>
            {t('Administrator-managed theme')}
          </p>
          <p className='text-muted-foreground text-xs'>
            {t(
              'The preset, typography, spacing, and layout are applied globally. Users can still choose light, dark, or system mode.'
            )}
          </p>
        </div>

        <div className='grid gap-4 sm:grid-cols-2 lg:col-span-2'>
          <ThemeSelect
            label={t('Default color mode')}
            value={settings.theme}
            options={MODE_OPTIONS.map((option) => ({
              ...option,
              label: t(option.label),
            }))}
            onChange={(value) => setValue('theme', value as ThemeMode)}
          />
          <ThemeSelect
            label={t('Font')}
            value={settings.font}
            options={FONT_OPTIONS.map((option) => ({
              ...option,
              label: t(option.label),
            }))}
            onChange={(value) => setValue('font', value as ThemeFont)}
          />
          <ThemeSelect
            label={t('Border radius')}
            value={settings.radius}
            options={RADIUS_OPTIONS.map((option) => ({
              ...option,
              label: t(option.label),
            }))}
            onChange={(value) => setValue('radius', value as ThemeRadius)}
          />
          <ThemeSelect
            label={t('Density')}
            value={settings.scale}
            options={SCALE_OPTIONS.map((option) => ({
              ...option,
              label: t(option.label),
            }))}
            onChange={(value) => setValue('scale', value as ThemeScale)}
          />
          <ThemeSelect
            label={t('Content width')}
            value={settings.contentLayout}
            options={[
              { value: 'full', label: t('Full width') },
              { value: 'centered', label: t('Centered') },
            ]}
            onChange={(value) =>
              setValue(
                'contentLayout',
                value as GlobalThemeSettings['contentLayout']
              )
            }
          />
          <ThemeSelect
            label={t('Sidebar')}
            value={settings.layoutVariant}
            options={LAYOUT_VARIANT_OPTIONS.map((option) => ({
              ...option,
              label: t(option.label),
            }))}
            onChange={(value) =>
              setValue('layoutVariant', value as GlobalLayoutVariant)
            }
          />
          <ThemeSelect
            label={t('Layout')}
            value={settings.layoutCollapsible}
            options={LAYOUT_COLLAPSIBLE_OPTIONS.map((option) => ({
              ...option,
              label: t(option.label),
            }))}
            onChange={(value) =>
              setValue('layoutCollapsible', value as GlobalLayoutCollapsible)
            }
          />
          <ThemeSelect
            label={t('Direction')}
            value={settings.direction}
            options={DIRECTION_OPTIONS.map((option) => ({
              ...option,
              label: t(option.label),
            }))}
            onChange={(value) =>
              setValue('direction', value as GlobalDirection)
            }
          />
        </div>

        <div className='space-y-2 lg:col-span-2'>
          <Label>{t('Color preset')}</Label>
          <div className='grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-5'>
            {THEME_PRESETS.map((preset) => (
              <Button
                key={preset.value}
                type='button'
                variant={
                  settings.preset === preset.value ? 'default' : 'outline'
                }
                className='h-auto justify-start gap-2 px-2.5 py-2 text-left'
                onClick={() => setValue('preset', preset.value)}
              >
                <span
                  className='ring-border size-5 shrink-0 rounded-full ring-1'
                  style={{
                    background: `linear-gradient(135deg, ${preset.swatches[0]}, ${preset.swatches[1] ?? preset.swatches[0]})`,
                  }}
                  aria-hidden='true'
                />
                <span className='truncate text-xs'>
                  {t(`preset.${preset.value}`)}
                </span>
              </Button>
            ))}
          </div>
        </div>
      </SettingsForm>
    </SettingsSection>
  )
}

function ThemeSelect(props: {
  label: string
  value: string
  options: Array<{ value: string; label: string }>
  onChange: (value: string) => void
}) {
  return (
    <div className='space-y-2'>
      <Label>{props.label}</Label>
      <Select
        value={props.value}
        onValueChange={(value) => value && props.onChange(value)}
      >
        <SelectTrigger>
          <SelectValue />
        </SelectTrigger>
        <SelectContent alignItemWithTrigger={false}>
          <SelectGroup>
            {props.options.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    </div>
  )
}
