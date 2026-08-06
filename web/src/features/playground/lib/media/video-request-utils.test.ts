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
import { buildVideoGenerationRequest } from './video-request-utils.ts'

describe('buildVideoGenerationRequest', () => {
  test('uses Grok creative-console controls and keeps references in order', () => {
    const request = buildVideoGenerationRequest(
      {
        ...DEFAULT_CONFIG,
        model: 'grok-imagine-video',
        group: 'default',
        video_duration: 10,
        video_aspect_ratio: '9:16',
        video_resolution: '1080p',
      },
      'A camera rises above the city.',
      [' data:image/png;base64,AAAA ', 'https://example.test/second.png']
    )

    assert.deepEqual(request, {
      model: 'grok-imagine-video',
      group: 'default',
      prompt: 'A camera rises above the city.',
      duration: 10,
      aspect_ratio: '9:16',
      resolution: '1080p',
      image: { url: 'data:image/png;base64,AAAA' },
      reference_images: [{ url: 'https://example.test/second.png' }],
    })
    assert.equal('size' in request, false)
    assert.equal('seconds' in request, false)
    assert.equal('quality' in request, false)
    assert.equal('n' in request, false)
  })
})
