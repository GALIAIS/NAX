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
import type { ImageGenerationRequest, PlaygroundConfig } from '../../types'

export function isGrokImageModel(model: string): boolean {
  return model.trim().toLowerCase().startsWith('grok-imagine-image')
}

export function buildImageGenerationRequest(
  config: PlaygroundConfig,
  prompt: string
): ImageGenerationRequest {
  const countLimit = isGrokImageModel(config.model) ? 4 : 10
  const request: ImageGenerationRequest = {
    model: config.model,
    group: config.group,
    prompt,
    n: Math.min(countLimit, Math.max(1, Math.round(config.image_n))),
  }

  if (isGrokImageModel(config.model)) {
    return {
      ...request,
      aspect_ratio: config.image_aspect_ratio,
      resolution: config.image_resolution,
      response_format: 'url',
      stream: false,
    }
  }

  return {
    ...request,
    size: config.image_size,
    quality: config.image_quality,
    response_format: config.image_response_format,
  }
}
