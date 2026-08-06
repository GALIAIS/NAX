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
import { ImageIcon, SendIcon, SquareIcon, VideoIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import {
  PromptInputButton,
  usePromptInputAttachments,
} from '@/components/ai-elements/prompt-input'
import { ModelGroupSelector } from '@/components/model-group-selector'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import { getInputControlState } from '../../lib'
import type { GroupOption, ModelOption, PlaygroundMode } from '../../types'

type PlaygroundInputControlsProps = {
  disabled?: boolean
  groups: GroupOption[]
  groupValue: string
  isGenerating?: boolean
  isModelLoading?: boolean
  models: ModelOption[]
  modelValue: string
  onGroupChange: (value: string) => void
  onModelChange: (value: string) => void
  onStop?: () => void
  mode: PlaygroundMode
  hasReferenceURL?: boolean
  onModeChange: (mode: PlaygroundMode) => void
  text: string
  tools: ReactNode
}

export function PlaygroundInputControls({
  disabled,
  groups,
  groupValue,
  isGenerating,
  isModelLoading = false,
  models,
  modelValue,
  onGroupChange,
  onModelChange,
  onStop,
  mode,
  hasReferenceURL = false,
  onModeChange,
  text,
  tools,
}: PlaygroundInputControlsProps) {
  const { t } = useTranslation()
  const attachments = usePromptInputAttachments()
  const { canSubmit, isSelectorDisabled, shouldShowStop } =
    getInputControlState({
      allowEmptyText:
        mode === 'video' && (hasReferenceURL || attachments.files.length > 0),
      disabled,
      groups,
      hasStopHandler: Boolean(onStop),
      isGenerating,
      isModelLoading,
      models,
      text,
    })

  let modeLabel = t('Chat')
  let modeIcon: ReactNode = null
  if (mode === 'image') {
    modeLabel = t('Image generation')
    modeIcon = <ImageIcon size={14} />
  } else if (mode === 'video') {
    modeLabel = t('Video generation')
    modeIcon = <VideoIcon size={14} />
  }

  const renderSelector = () => (
    <div className='flex min-w-0 items-center gap-2'>
      <Select
        value={mode}
        onValueChange={(value) => {
          if (value === 'chat' || value === 'image' || value === 'video') {
            onModeChange(value)
          }
        }}
        disabled={isSelectorDisabled}
      >
        <SelectTrigger
          className='h-8 w-[7.25rem] shrink-0 text-xs'
          aria-label={t('Playground mode')}
        >
          <SelectValue>
            <span className='inline-flex items-center gap-1.5'>
              {modeIcon}
              {modeLabel}
            </span>
          </SelectValue>
        </SelectTrigger>
        <SelectContent alignItemWithTrigger={false}>
          <SelectItem value='chat'>{t('Chat')}</SelectItem>
          <SelectItem value='image'>{t('Image generation')}</SelectItem>
          <SelectItem value='video'>{t('Video generation')}</SelectItem>
        </SelectContent>
      </Select>
      <ModelGroupSelector
        selectedModel={modelValue}
        models={models}
        onModelChange={onModelChange}
        selectedGroup={groupValue}
        groups={groups}
        onGroupChange={onGroupChange}
        disabled={isSelectorDisabled}
      />
    </div>
  )

  const renderSubmitButton = () =>
    shouldShowStop ? (
      <PromptInputButton
        className='border-destructive/25 bg-destructive/10 text-destructive hover:bg-destructive/15 font-medium'
        onClick={onStop}
        variant='secondary'
      >
        <SquareIcon className='fill-current' size={16} />
        <span className='hidden sm:inline'>{t('Stop')}</span>
        <span className='sr-only sm:hidden'>{t('Stop')}</span>
      </PromptInputButton>
    ) : (
      <PromptInputButton
        className='bg-primary text-primary-foreground hover:bg-primary/90 disabled:bg-muted disabled:text-muted-foreground h-8 px-3 font-medium shadow-sm'
        disabled={!canSubmit}
        type='submit'
        variant='default'
      >
        <SendIcon size={16} />
        <span className='hidden sm:inline'>{t('Send')}</span>
        <span className='sr-only sm:hidden'>{t('Send')}</span>
      </PromptInputButton>
    )

  return (
    <div className='flex w-full flex-col gap-2.5 md:flex-row md:items-center md:justify-between'>
      <div className='flex min-w-0 items-center justify-end md:hidden'>
        {renderSelector()}
      </div>

      <div className='flex items-center justify-between gap-2 md:justify-start'>
        {tools}
        <div className='flex items-center gap-1.5 md:hidden'>
          {renderSubmitButton()}
        </div>
      </div>

      <div className='hidden min-w-0 items-center gap-2 md:flex'>
        {renderSelector()}
        {renderSubmitButton()}
      </div>
    </div>
  )
}
