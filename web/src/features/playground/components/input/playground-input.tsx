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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  PromptInput,
  PromptInputAttachment,
  PromptInputAttachments,
  PromptInputFooter,
  PromptInputHeader,
  PromptInputTextarea,
  type PromptInputMessage,
} from '@/components/ai-elements/prompt-input'

import { getSubmittableInputText, isValidReferenceImageURL } from '../../lib'
import type {
  ModelOption,
  GroupOption,
  ParameterEnabled,
  PlaygroundConfig,
  PlaygroundMedia,
} from '../../types'
import { PlaygroundInputControls } from './playground-input-controls'
import { PlaygroundInputTools } from './playground-input-tools'

interface PlaygroundInputProps {
  config: PlaygroundConfig
  onSubmit: (text: string, inputReferences?: PlaygroundMedia[]) => void
  onStop?: () => void
  onModeChange: (mode: PlaygroundConfig['mode']) => void
  disabled?: boolean
  isGenerating?: boolean
  models: ModelOption[]
  modelValue: string
  onModelChange: (value: string) => void
  isModelLoading?: boolean
  groups: GroupOption[]
  groupValue: string
  onGroupChange: (value: string) => void
  hasMessages?: boolean
  onConfigChange: <K extends keyof PlaygroundConfig>(
    key: K,
    value: PlaygroundConfig[K]
  ) => void
  onClearMessages?: () => void
  onParameterEnabledChange: (
    key: keyof ParameterEnabled,
    value: boolean
  ) => void
  parameterEnabled: ParameterEnabled
}

export function PlaygroundInput({
  config,
  onSubmit,
  onStop,
  onModeChange,
  disabled,
  isGenerating,
  models,
  modelValue,
  onModelChange,
  isModelLoading = false,
  groups,
  groupValue,
  onGroupChange,
  hasMessages = false,
  onConfigChange,
  onClearMessages,
  onParameterEnabledChange,
  parameterEnabled,
}: PlaygroundInputProps) {
  const { t } = useTranslation()
  const [text, setText] = useState('')
  let placeholder = t('Ask anything')
  if (config.mode === 'image') {
    placeholder = t('Describe the image you want to create')
  } else if (config.mode === 'video') {
    placeholder = t('Describe the video you want to create')
  }

  const handleSubmit = (message: PromptInputMessage) => {
    if (disabled) return
    const submittableText = getSubmittableInputText(message, disabled)
    const uploadedReferences =
      config.mode === 'video'
        ? (message.files ?? [])
            .filter(
              (file) => file.mediaType.startsWith('image/') && file.url.trim()
            )
            .slice(0, 1)
            .map((file) => ({
              kind: 'image' as const,
              url: file.url.trim(),
              alt: file.filename || t('Reference image'),
              mimeType: file.mediaType,
            }))
        : []
    const configuredReference = config.video_reference_url.trim()
    if (
      uploadedReferences.length === 0 &&
      configuredReference &&
      !isValidReferenceImageURL(configuredReference)
    ) {
      toast.error(t('Reference image URL must use HTTP or HTTPS'))
      return
    }
    let inputReferences: PlaygroundMedia[] = uploadedReferences
    if (
      inputReferences.length === 0 &&
      config.mode === 'video' &&
      configuredReference
    ) {
      inputReferences = [
        {
          kind: 'image',
          url: configuredReference,
          alt: t('Reference image'),
        },
      ]
    }

    if (!submittableText && inputReferences.length === 0) return
    onSubmit(submittableText ?? '', inputReferences)
    setText('')
  }

  return (
    <div className='grid shrink-0 gap-4 px-1 md:pb-4'>
      <PromptInput
        accept={config.mode === 'video' ? 'image/*' : undefined}
        className='relative'
        maxFiles={1}
        maxFileSize={10 * 1024 * 1024}
        multiple={false}
        replaceOnMaxFiles={config.mode === 'video'}
        groupClassName='bg-background/95 dark:bg-background/80 border-border/70 shadow-[0_18px_60px_-32px_rgba(0,0,0,0.65)] ring-1 ring-foreground/5 rounded-xl overflow-hidden transition-all duration-200 focus-within:border-primary/45 focus-within:ring-primary/15 focus-within:shadow-[0_22px_70px_-34px_rgba(0,0,0,0.75)]'
        onError={(error) => toast.error(error.message)}
        onSubmit={handleSubmit}
      >
        {config.mode === 'video' ? (
          <PromptInputHeader className='border-border/50 bg-muted/15 border-b px-3 py-2'>
            <PromptInputAttachments>
              {(attachment) => <PromptInputAttachment data={attachment} />}
            </PromptInputAttachments>
          </PromptInputHeader>
        ) : null}
        <PromptInputTextarea
          autoComplete='off'
          autoCorrect='off'
          autoCapitalize='off'
          spellCheck={false}
          className='min-h-20 px-5 pt-4 pb-3 leading-7 md:min-h-24 md:text-base'
          disabled={disabled}
          onChange={(event) => setText(event.target.value)}
          placeholder={placeholder}
          value={text}
        />

        <PromptInputFooter className='border-border/60 bg-muted/20 dark:bg-muted/10 border-t px-3 py-2.5 backdrop-blur'>
          <PlaygroundInputControls
            disabled={disabled}
            groups={groups}
            groupValue={groupValue}
            isGenerating={isGenerating}
            isModelLoading={isModelLoading}
            models={models}
            modelValue={modelValue}
            onGroupChange={onGroupChange}
            onModelChange={onModelChange}
            onStop={onStop}
            mode={config.mode}
            hasReferenceURL={Boolean(config.video_reference_url.trim())}
            onModeChange={onModeChange}
            text={text}
            tools={
              <PlaygroundInputTools
                config={config}
                disabled={disabled}
                hasMessages={hasMessages}
                onConfigChange={onConfigChange}
                onClearMessages={onClearMessages}
                onParameterEnabledChange={onParameterEnabledChange}
                parameterEnabled={parameterEnabled}
              />
            }
          />
        </PromptInputFooter>
      </PromptInput>
    </div>
  )
}
