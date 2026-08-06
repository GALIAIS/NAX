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
import assert from 'node:assert/strict'
import { after, describe, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLButtonElement',
  'HTMLInputElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'KeyboardEvent',
  'PointerEvent',
  'CustomEvent',
  'MutationObserver',
  'ResizeObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act, useState } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { AdvancedCustomModelSelector } =
  await import('../advanced-custom-editor-dialog')

const i18n = createInstance()
await i18n.use(initReactI18next).init({ lng: 'en' })

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const options = [
  { value: 'deepseek-v4-flash', label: 'deepseek-v4-flash' },
  { value: 'glm-5.2', label: 'glm-5.2' },
  { value: 'kimi-k3', label: 'kimi-k3' },
]

function Harness() {
  const [models, setModels] = useState(['deepseek-v4-flash'])

  return (
    <I18nextProvider i18n={i18n}>
      <AdvancedCustomModelSelector
        models={models}
        options={options}
        ariaInvalid
        onChange={setModels}
      />
      <output data-testid='models'>{models.join(',')}</output>
    </I18nextProvider>
  )
}

describe('advanced custom model selector', () => {
  after(() => {
    domWindow.close()
  })

  test('fills one route with all channel models and can clear it to fallback', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => root.render(<Harness />))

    const chips = container.querySelector('[data-slot="combobox-chips"]')
    assert.ok(chips)
    assert.equal(chips.getAttribute('aria-invalid'), 'true')

    const buttons = [...container.querySelectorAll('button')]
    const fillButton = buttons.find(
      (button) => button.textContent === 'Fill All Models'
    )
    assert.ok(fillButton)
    await act(async () => fillButton.click())
    assert.equal(
      container.querySelector('[data-testid="models"]')?.textContent,
      'deepseek-v4-flash,glm-5.2,kimi-k3'
    )

    const clearButton = [...container.querySelectorAll('button')].find(
      (button) => button.textContent === 'Clear All'
    )
    assert.ok(clearButton)
    await act(async () => clearButton.click())
    assert.equal(
      container.querySelector('[data-testid="models"]')?.textContent,
      ''
    )

    await act(async () => root.unmount())
    container.remove()
  })
})
