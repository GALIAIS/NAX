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
  parseAdvancedCustomConfig,
  parseAdvancedCustomRouteModels,
  stringifyAdvancedCustomConfig,
  validateAdvancedCustomConfig,
} from '../advanced-custom'

describe('advanced custom visual configuration', () => {
  test('parses pasted model rules from English commas, Chinese commas, and new lines', () => {
    const models = parseAdvancedCustomRouteModels(
      'deepseek-v4-flash，glm-5.2\nkimi-k3, deepseek-v4-flash'
    )

    assert.deepEqual(models, ['deepseek-v4-flash', 'glm-5.2', 'kimi-k3'])
  })

  test('accepts multiple model rules and a custom incoming path', () => {
    const error = validateAdvancedCustomConfig({
      advanced_routes: [
        {
          incoming_path: '/vendor/v2/generate',
          upstream_path: '/custom/generate',
          converter: 'none',
          models: ['deepseek-v4-flash', 'glm-5.2', 're:^kimi-'],
        },
      ],
    })

    assert.equal(error, null)
  })

  test('keeps forward-compatible fields when visual mode normalizes JSON', () => {
    const parsed = parseAdvancedCustomConfig(
      JSON.stringify({
        vendor_metadata: { region: 'global' },
        advanced_routes: [
          {
            incoming_path: '/v1/chat/completions',
            upstream_path: '/v1/chat/completions',
            converter: 'none',
            retry_policy: { attempts: 3 },
            auth: {
              type: 'header',
              name: 'Authorization',
              value: 'Bearer {api_key}',
              rotate: true,
            },
          },
        ],
      })
    )

    assert.ok(parsed)
    const serialized = JSON.parse(stringifyAdvancedCustomConfig(parsed))
    assert.deepEqual(serialized.vendor_metadata, { region: 'global' })
    assert.deepEqual(serialized.advanced_routes[0].retry_policy, {
      attempts: 3,
    })
    assert.equal(serialized.advanced_routes[0].auth.rotate, true)
  })
})
