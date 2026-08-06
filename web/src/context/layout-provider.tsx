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
import { createContext, useContext } from 'react'

import {
  DEFAULT_GLOBAL_THEME_SETTINGS,
  type GlobalLayoutCollapsible,
  type GlobalLayoutVariant,
} from '@/lib/theme-customization'
import { useSystemConfigStore } from '@/stores/system-config-store'

export type Collapsible = GlobalLayoutCollapsible
export type Variant = GlobalLayoutVariant

type LayoutContextType = {
  resetLayout: () => void

  defaultCollapsible: Collapsible
  collapsible: Collapsible
  setCollapsible: (collapsible: Collapsible) => void

  defaultVariant: Variant
  variant: Variant
  setVariant: (variant: Variant) => void
}

const LayoutContext = createContext<LayoutContextType | null>(null)

type LayoutProviderProps = {
  children: React.ReactNode
}

export function LayoutProvider({ children }: LayoutProviderProps) {
  const globalTheme =
    useSystemConfigStore((state) => state.config.globalTheme) ??
    DEFAULT_GLOBAL_THEME_SETTINGS
  const collapsible = globalTheme.layoutCollapsible
  const variant = globalTheme.layoutVariant

  // Layout is administrator-managed along with the color theme. Keep the
  // context methods for existing consumers, but prevent local cookie/state
  // mutations from diverging between users.
  const setCollapsible = (_newCollapsible: Collapsible) => {}
  const setVariant = (_newVariant: Variant) => {}
  const resetLayout = () => {}

  const contextValue: LayoutContextType = {
    resetLayout,
    defaultCollapsible: DEFAULT_GLOBAL_THEME_SETTINGS.layoutCollapsible,
    collapsible,
    setCollapsible,
    defaultVariant: DEFAULT_GLOBAL_THEME_SETTINGS.layoutVariant,
    variant,
    setVariant,
  }

  return <LayoutContext value={contextValue}>{children}</LayoutContext>
}

// Define the hook for the provider
// eslint-disable-next-line react-refresh/only-export-components
export function useLayout() {
  const context = useContext(LayoutContext)
  if (!context) {
    throw new Error('useLayout must be used within a LayoutProvider')
  }
  return context
}
