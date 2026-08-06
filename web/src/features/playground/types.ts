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
// Message types
export type MessageRole = 'user' | 'assistant' | 'system'

export type MessageStatus = 'loading' | 'streaming' | 'complete' | 'error'

export type PlaygroundMessageLayoutMode = 'alternating' | 'left'

export type PlaygroundMode = 'chat' | 'image' | 'video'

export type PlaygroundMedia = {
  kind: 'image' | 'video'
  url: string
  taskId?: string
  alt?: string
  mimeType?: string
}

export interface MessageVersion {
  id: string
  content: string
}

export interface Message {
  key: string
  from: MessageRole
  versions: MessageVersion[]
  createdAt?: number
  startedAt?: number
  completedAt?: number
  durationMs?: number
  sources?: { href: string; title: string }[]
  reasoning?: {
    content: string
    duration: number
    startedAt?: number
    completedAt?: number
    durationMs?: number
  }
  isReasoningStreaming?: boolean
  isReasoningComplete?: boolean
  isContentComplete?: boolean
  status?: MessageStatus
  errorCode?: string | null
  media?: PlaygroundMedia[]
  /**
   * Ephemeral source media used by image-to-video requests. The storage
   * schema intentionally omits this field so local data URLs are never
   * persisted to localStorage.
   */
  inputReferences?: PlaygroundMedia[]
}

// API payload types
export interface ChatCompletionMessage {
  role: MessageRole
  content: string | ContentPart[]
}

export interface ContentPart {
  type: 'text' | 'image_url'
  text?: string
  image_url?: {
    url: string
  }
}

export interface ChatCompletionRequest {
  model: string
  group?: string
  messages: ChatCompletionMessage[]
  stream: boolean
  temperature?: number
  top_p?: number
  max_tokens?: number
  frequency_penalty?: number
  presence_penalty?: number
  seed?: number
}

export interface ChatCompletionChunk {
  id: string
  object: string
  created: number
  model: string
  choices: Array<{
    index: number
    delta: {
      role?: MessageRole
      content?: string
      reasoning_content?: string
    }
    finish_reason: string | null
  }>
}

export interface ChatCompletionResponse {
  id: string
  object: string
  created: number
  model: string
  choices: Array<{
    index: number
    message: {
      role: MessageRole
      content: string
      reasoning_content?: string
    }
    finish_reason: string
  }>
  usage?: {
    prompt_tokens: number
    completion_tokens: number
    total_tokens: number
  }
}

// Configuration types
export interface PlaygroundConfig {
  mode: PlaygroundMode
  model: string
  group: string
  temperature: number
  top_p: number
  max_tokens: number
  frequency_penalty: number
  presence_penalty: number
  seed: number | null
  stream: boolean
  image_size: string
  image_quality: 'auto' | 'standard' | 'hd'
  image_n: number
  image_response_format: 'url' | 'b64_json'
  image_aspect_ratio: string
  image_resolution: string
  video_size: string
  video_seconds: number
  video_quality: string
  video_duration: number
  video_aspect_ratio: string
  video_resolution: string
  video_reference_url: string
}

export interface ParameterEnabled {
  temperature: boolean
  top_p: boolean
  max_tokens: boolean
  frequency_penalty: boolean
  presence_penalty: boolean
  seed: boolean
}

// Model and group options
export interface ModelOption {
  label: string
  value: string
}

export interface GroupOption {
  label: string
  value: string
  ratio: number
  desc?: string
}

export interface ImageGenerationRequest {
  model: string
  prompt: string
  group?: string
  n?: number
  size?: string
  quality?: string
  response_format?: 'url' | 'b64_json'
  aspect_ratio?: string
  resolution?: string
  stream?: boolean
}

export interface ImageGenerationResponse {
  data?: Array<{
    url?: string
    b64_json?: string
    mime_type?: string
    mimeType?: string
    revised_prompt?: string
  }>
}

export interface VideoGenerationRequest {
  model: string
  prompt: string
  group?: string
  size?: string
  seconds?: string
  quality?: string
  n?: number
  duration?: number
  aspect_ratio?: string
  resolution?: string
  image?: string | { url: string }
  reference_images?: Array<string | { url: string }>
}

export interface VideoGenerationResponse {
  id?: string
  request_id?: string
  task_id?: string
  status?: string
  metadata?: {
    url?: string
    video_url?: string
    content_url?: string
  }
  url?: string
  video_url?: string
  content_url?: string
  output_url?: string
}

export interface VideoStatusResponse extends VideoGenerationResponse {
  error?: {
    message?: string
  }
}
