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
import { describe, test } from 'node:test'

import {
  getInputControlState,
  isValidReferenceImageURL,
} from './input-control-utils.ts'

const baseOptions = {
  groups: [{ label: 'default', value: 'default', ratio: 1 }],
  hasStopHandler: true,
  models: [{ label: 'video-model', value: 'video-model' }],
  text: '',
}

describe('getInputControlState', () => {
  test('allows an image-only video request when a reference is present', () => {
    const state = getInputControlState({
      ...baseOptions,
      allowEmptyText: true,
    })

    assert.equal(state.canSubmit, true)
  })

  test('still requires text when no media reference is available', () => {
    const state = getInputControlState(baseOptions)

    assert.equal(state.canSubmit, false)
  })
})

describe('isValidReferenceImageURL', () => {
  test('accepts HTTP image locations and rejects unsafe schemes', () => {
    assert.equal(
      isValidReferenceImageURL('https://example.test/reference.png'),
      true
    )
    assert.equal(isValidReferenceImageURL('javascript:alert(1)'), false)
    assert.equal(isValidReferenceImageURL('not a URL'), false)
  })
})
