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
import { ImageIcon, VideoIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { PromptInputButton } from '@/components/ai-elements/prompt-input'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import type { PlaygroundConfig } from '../../types'

type PlaygroundMediaPanelProps = {
  config: PlaygroundConfig
  disabled?: boolean
  onConfigChange: <K extends keyof PlaygroundConfig>(
    key: K,
    value: PlaygroundConfig[K]
  ) => void
}

export function PlaygroundMediaPanel(props: PlaygroundMediaPanelProps) {
  const { t } = useTranslation()
  if (props.config.mode === 'chat') return null

  const isImage = props.config.mode === 'image'
  const label = isImage ? t('Image options') : t('Video options')
  const Icon = isImage ? ImageIcon : VideoIcon

  return (
    <Popover>
      <Tooltip>
        <TooltipTrigger
          render={
            <PopoverTrigger
              render={
                <PromptInputButton
                  aria-label={label}
                  className='text-muted-foreground hover:text-foreground hover:bg-muted/70 relative font-medium'
                  disabled={props.disabled}
                  variant='ghost'
                >
                  <Icon size={16} />
                </PromptInputButton>
              }
            />
          }
        />
        <TooltipContent>
          <p>{label}</p>
        </TooltipContent>
      </Tooltip>
      <PopoverContent
        align='start'
        className='w-[22rem] max-w-[calc(100vw-2rem)] space-y-3 p-3'
        collisionPadding={8}
        side='top'
        sideOffset={8}
      >
        <div className='flex items-center gap-2'>
          <Icon size={16} />
          <div>
            <div className='text-sm font-semibold'>{label}</div>
            <div className='text-muted-foreground text-xs'>
              {t(
                'These values are sent only for the selected generation mode.'
              )}
            </div>
          </div>
        </div>

        {isImage ? (
          <div className='grid gap-3'>
            <MediaSelect
              label={t('Image size')}
              value={props.config.image_size}
              options={['1024x1024', '1536x1024', '1024x1536']}
              onChange={(value) => props.onConfigChange('image_size', value)}
            />
            <MediaSelect
              label={t('Image quality')}
              value={props.config.image_quality}
              options={['auto', 'standard', 'hd']}
              onChange={(value) =>
                props.onConfigChange(
                  'image_quality',
                  value as PlaygroundConfig['image_quality']
                )
              }
            />
            <MediaSelect
              label={t('Image response')}
              value={props.config.image_response_format}
              options={['url', 'b64_json']}
              onChange={(value) =>
                props.onConfigChange(
                  'image_response_format',
                  value as PlaygroundConfig['image_response_format']
                )
              }
            />
            <label className='grid gap-1 text-xs'>
              <span className='text-muted-foreground'>{t('Image count')}</span>
              <Input
                type='number'
                min={1}
                max={10}
                value={props.config.image_n}
                onChange={(event) =>
                  props.onConfigChange(
                    'image_n',
                    Math.min(10, Math.max(1, Number(event.target.value) || 1))
                  )
                }
              />
            </label>
          </div>
        ) : (
          <div className='grid gap-3'>
            <MediaSelect
              label={t('Video size')}
              value={props.config.video_size}
              options={['1280x720', '720x1280', '1024x1024']}
              onChange={(value) => props.onConfigChange('video_size', value)}
            />
            <MediaSelect
              label={t('Video quality')}
              value={props.config.video_quality}
              options={['standard', 'high']}
              onChange={(value) => props.onConfigChange('video_quality', value)}
            />
            <label className='grid gap-1 text-xs'>
              <span className='text-muted-foreground'>
                {t('Video seconds')}
              </span>
              <Input
                type='number'
                min={1}
                max={60}
                value={props.config.video_seconds}
                onChange={(event) =>
                  props.onConfigChange(
                    'video_seconds',
                    Math.min(60, Math.max(1, Number(event.target.value) || 1))
                  )
                }
              />
            </label>
          </div>
        )}
      </PopoverContent>
    </Popover>
  )
}

function MediaSelect(props: {
  label: string
  value: string
  options: string[]
  onChange: (value: string) => void
}) {
  return (
    <label className='grid gap-1 text-xs'>
      <span className='text-muted-foreground'>{props.label}</span>
      <Select
        value={props.value}
        onValueChange={(value) => value && props.onChange(value)}
      >
        <SelectTrigger>
          <SelectValue />
        </SelectTrigger>
        <SelectContent alignItemWithTrigger={false}>
          <SelectGroup>
            {props.options.map((option) => (
              <SelectItem key={option} value={option}>
                {option}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    </label>
  )
}
