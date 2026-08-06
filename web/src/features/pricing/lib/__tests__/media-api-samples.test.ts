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
  buildImageApiSample,
  buildVideoApiSample,
} from '../media-api-samples.ts'

const context = {
  baseUrl: 'https://api.example.com',
  apiKeyEnv: 'NEW_API_KEY',
  modelName: 'grok-imagine-video',
  endpointPath: '/v1/videos',
}

describe('media API samples', () => {
  test('video samples include Grok controls, reference image, and polling', () => {
    for (const language of [
      'curl',
      'python',
      'typescript',
      'javascript',
    ] as const) {
      const sample = buildVideoApiSample(language, context)
      assert.match(sample, /grok-imagine-video/)
      assert.match(sample, /duration/)
      assert.match(sample, /aspect_ratio/)
      assert.match(sample, /resolution/)
      assert.match(sample, /reference\.png/)
      assert.match(sample, /\/v1\/videos\//)
    }
  })

  test('image samples use Grok image controls and response format', () => {
    const sample = buildImageApiSample('typescript', {
      ...context,
      modelName: 'grok-imagine-image',
      endpointPath: '/v1/images/generations',
    })

    assert.match(sample, /aspect_ratio/)
    assert.match(sample, /resolution/)
    assert.match(sample, /response_format/)
    assert.doesNotMatch(sample, /quality: 'standard'/)
  })

  test('generic image samples retain OpenAI size and quality controls', () => {
    const sample = buildImageApiSample('curl', {
      ...context,
      modelName: 'dall-e-3',
      endpointPath: '/v1/images/generations',
    })

    assert.match(sample, /1024x1024/)
    assert.match(sample, /standard/)
  })
})
