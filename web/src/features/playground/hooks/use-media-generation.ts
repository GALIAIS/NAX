/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  cancelVideo,
  createVideo,
  generateImages,
  getVideoContent,
  getVideoStatus,
} from '../api'
import {
  completeAssistantTiming,
  parseRequestErrorDetails,
  updateAssistantMessageWithError,
  updateLastAssistantMessage,
  updateCurrentVersionContent,
} from '../lib'
import type { Message, PlaygroundConfig, PlaygroundMedia } from '../types'

type UseMediaGenerationOptions = {
  config: PlaygroundConfig
  onMessageUpdate: (updater: (prev: Message[]) => Message[]) => void
}

const VIDEO_POLL_INTERVAL_MS = 1500
const VIDEO_POLL_LIMIT = 240

function getPrompt(messages: Message[]): string {
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const message = messages[index]
    if (message.from === 'user') {
      return message.versions[0]?.content.trim() || ''
    }
  }
  return ''
}

function getImageURL(item: {
  url?: string
  b64_json?: string
  mime_type?: string
  mimeType?: string
}): string {
  if (item.url?.trim()) return item.url.trim()
  if (item.b64_json?.trim()) {
    const mimeType =
      item.mime_type?.trim() || item.mimeType?.trim() || 'image/png'
    return `data:${mimeType};base64,${item.b64_json}`
  }
  return ''
}

function getTaskID(response: { id?: string; task_id?: string }): string {
  return response.id?.trim() || response.task_id?.trim() || ''
}

function isExternalMediaURL(value: string): boolean {
  return /^(?:https?:|data:|blob:)/i.test(value)
}

function getVideoState(
  status: string | undefined
): 'pending' | 'complete' | 'failed' {
  const normalized = status?.toLowerCase().trim()
  if (
    normalized === 'completed' ||
    normalized === 'succeeded' ||
    normalized === 'success' ||
    normalized === 'done'
  ) {
    return 'complete'
  }
  if (
    normalized === 'failed' ||
    normalized === 'error' ||
    normalized === 'cancelled' ||
    normalized === 'canceled'
  ) {
    return 'failed'
  }
  return 'pending'
}

function waitForNextPoll(signal: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    let timeout = 0
    let settled = false
    function cleanup() {
      signal.removeEventListener('abort', abort)
    }
    function abort() {
      if (settled) return
      settled = true
      window.clearTimeout(timeout)
      cleanup()
      reject(new DOMException('Generation aborted', 'AbortError'))
    }
    timeout = window.setTimeout(() => {
      if (settled) return
      settled = true
      cleanup()
      resolve()
    }, VIDEO_POLL_INTERVAL_MS)
    signal.addEventListener('abort', abort, { once: true })
    if (signal.aborted) abort()
  })
}

export function useMediaGeneration({
  config,
  onMessageUpdate,
}: UseMediaGenerationOptions) {
  const { t } = useTranslation()
  const [isGenerating, setIsGenerating] = useState(false)
  const abortControllerRef = useRef<AbortController | null>(null)
  const requestGenerationRef = useRef(0)
  const activeVideoTaskRef = useRef<string | null>(null)

  useEffect(
    () => () => {
      requestGenerationRef.current += 1
      abortControllerRef.current?.abort()
      abortControllerRef.current = null
      activeVideoTaskRef.current = null
    },
    []
  )

  const setMediaMessage = useCallback(
    (generation: number, media: PlaygroundMedia[]) => {
      onMessageUpdate((previousMessages) => {
        if (requestGenerationRef.current !== generation) return previousMessages
        return updateLastAssistantMessage(previousMessages, (message) =>
          completeAssistantTiming({
            ...updateCurrentVersionContent(message, ''),
            media,
            status: 'complete',
            isContentComplete: true,
            isReasoningComplete: true,
            isReasoningStreaming: false,
          })
        )
      })
    },
    [onMessageUpdate]
  )

  const setError = useCallback(
    (generation: number, error: unknown) => {
      if (requestGenerationRef.current !== generation) return
      const { errorCode, errorMessage } = parseRequestErrorDetails(error)
      toast.error(errorMessage)
      onMessageUpdate((previousMessages) => {
        if (requestGenerationRef.current !== generation) return previousMessages
        return updateAssistantMessageWithError(
          previousMessages,
          errorMessage,
          errorCode,
          t('Request error occurred')
        )
      })
    },
    [onMessageUpdate, t]
  )

  const pollVideo = useCallback(
    async (taskId: string, signal: AbortSignal): Promise<string> => {
      for (let attempt = 0; attempt < VIDEO_POLL_LIMIT; attempt += 1) {
        const status = await getVideoStatus(taskId, signal)
        const state = getVideoState(status.status)
        if (state === 'complete') {
          const returnedURL = [
            status.url,
            status.video_url,
            status.content_url,
            status.output_url,
            status.metadata?.url,
            status.metadata?.video_url,
            status.metadata?.content_url,
          ]
            .map((value) => value?.trim())
            .find(Boolean)
          if (returnedURL && isExternalMediaURL(returnedURL)) {
            return returnedURL
          }
          return getVideoContent(taskId, signal)
        }
        if (state === 'failed') {
          throw new Error(status.error?.message || t('Video generation failed'))
        }
        await waitForNextPoll(signal)
      }
      throw new Error(t('Video generation timed out'))
    },
    [t]
  )

  const sendMedia = useCallback(
    async (messages: Message[]) => {
      const prompt = getPrompt(messages)
      if (!prompt) {
        toast.error(t('Enter a prompt before generating media'))
        return
      }

      const generation = requestGenerationRef.current + 1
      requestGenerationRef.current = generation
      abortControllerRef.current?.abort()
      const abortController = new AbortController()
      abortControllerRef.current = abortController
      activeVideoTaskRef.current = null
      setIsGenerating(true)

      try {
        if (config.mode === 'image') {
          const response = await generateImages(
            {
              model: config.model,
              group: config.group,
              prompt,
              n: Math.min(10, Math.max(1, Math.round(config.image_n))),
              size: config.image_size,
              quality: config.image_quality,
              response_format: config.image_response_format,
            },
            abortController.signal
          )
          const media = (response.data || [])
            .map((item) => getImageURL(item))
            .filter(Boolean)
            .map((url) => ({ kind: 'image' as const, url }))
          if (media.length === 0) {
            throw new Error(t('The image endpoint returned no media'))
          }
          setMediaMessage(generation, media)
        } else {
          const response = await createVideo(
            {
              model: config.model,
              group: config.group,
              prompt,
              size: config.video_size,
              seconds: String(
                Math.min(60, Math.max(1, Math.round(config.video_seconds)))
              ),
              quality: config.video_quality,
              n: 1,
            },
            abortController.signal
          )
          const taskId = getTaskID(response)
          if (!taskId) throw new Error(t('Video endpoint returned no task id'))
          activeVideoTaskRef.current = taskId
          const url = await pollVideo(taskId, abortController.signal)
          setMediaMessage(generation, [{ kind: 'video', url }])
        }
      } catch (error: unknown) {
        if (!abortController.signal.aborted) {
          setError(generation, error)
        }
      } finally {
        if (requestGenerationRef.current === generation) {
          setIsGenerating(false)
          abortControllerRef.current = null
          activeVideoTaskRef.current = null
        }
      }
    },
    [config, pollVideo, setError, setMediaMessage, t]
  )

  const stopGeneration = useCallback(() => {
    const taskId = activeVideoTaskRef.current
    const nextGeneration = requestGenerationRef.current + 1
    requestGenerationRef.current = nextGeneration
    abortControllerRef.current?.abort()
    abortControllerRef.current = null
    activeVideoTaskRef.current = null
    setIsGenerating(false)
    if (taskId) {
      void cancelVideo(taskId).catch(() => undefined)
    }
    onMessageUpdate((previousMessages) =>
      updateLastAssistantMessage(previousMessages, (message) =>
        completeAssistantTiming({
          ...message,
          status: 'complete',
          isReasoningStreaming: false,
        })
      )
    )
  }, [onMessageUpdate])

  return { sendMedia, stopGeneration, isGenerating }
}
