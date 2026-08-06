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
import type { PlaygroundConfig, VideoGenerationRequest } from '../../types'

export function buildVideoGenerationRequest(
  config: PlaygroundConfig,
  prompt: string,
  referenceURLs: string[]
): VideoGenerationRequest {
  const normalizedReferences = referenceURLs
    .map((value) => value.trim())
    .filter(Boolean)

  return {
    model: config.model,
    group: config.group,
    prompt,
    duration: Math.min(15, Math.max(1, Math.round(config.video_duration))),
    aspect_ratio: config.video_aspect_ratio,
    resolution: config.video_resolution,
    image: normalizedReferences[0]
      ? { url: normalizedReferences[0] }
      : undefined,
    reference_images:
      normalizedReferences.length > 1
        ? normalizedReferences.slice(1).map((url) => ({ url }))
        : undefined,
    n: 1,
  }
}
