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

import { DEFAULT_CONFIG } from '../../constants.ts'
import { buildImageGenerationRequest } from './image-request-utils.ts'

describe('buildImageGenerationRequest', () => {
  test('matches the Grok2API creative-console image payload', () => {
    const request = buildImageGenerationRequest(
      {
        ...DEFAULT_CONFIG,
        model: 'grok-imagine-image-quality',
        group: 'default',
        image_n: 9,
        image_aspect_ratio: '9:16',
        image_resolution: '2k',
      },
      'A neon-lit city in the rain.'
    )

    assert.deepEqual(request, {
      model: 'grok-imagine-image-quality',
      group: 'default',
      prompt: 'A neon-lit city in the rain.',
      n: 4,
      aspect_ratio: '9:16',
      resolution: '2k',
      response_format: 'url',
      stream: false,
    })
    assert.equal('size' in request, false)
    assert.equal('quality' in request, false)
  })

  test('keeps standard OpenAI image controls for other models', () => {
    const request = buildImageGenerationRequest(
      {
        ...DEFAULT_CONFIG,
        model: 'gpt-image-1',
        image_n: 2,
        image_size: '1536x1024',
        image_quality: 'hd',
        image_response_format: 'b64_json',
      },
      'A product photo.'
    )

    assert.equal(request.size, '1536x1024')
    assert.equal(request.quality, 'hd')
    assert.equal(request.response_format, 'b64_json')
    assert.equal('aspect_ratio' in request, false)
    assert.equal('resolution' in request, false)
  })
})
