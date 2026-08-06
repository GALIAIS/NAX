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
import {
  Clock3Icon,
  ImageIcon,
  ImagePlusIcon,
  LinkIcon,
  MonitorUpIcon,
  RatioIcon,
  VideoIcon,
  XIcon,
} from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import {
  PromptInputButton,
  usePromptInputAttachments,
} from '@/components/ai-elements/prompt-input'
import { Button } from '@/components/ui/button'
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

import { isGrokImageModel } from '../../lib/media/image-request-utils'
import type { PlaygroundConfig } from '../../types'

const GROK_IMAGE_COUNTS = ['1', '2', '3', '4'] as const
const GROK_IMAGE_ASPECT_RATIOS = [
  '1:1',
  '16:9',
  '9:16',
  '4:3',
  '3:4',
  '3:2',
  '2:3',
] as const
const GROK_IMAGE_RESOLUTIONS = ['1k', '2k'] as const
const GROK_VIDEO_DURATIONS = ['6', '10', '15'] as const
const GROK_VIDEO_ASPECT_RATIOS = [
  '1:1',
  '16:9',
  '9:16',
  '4:3',
  '3:4',
  '3:2',
  '2:3',
] as const
const GROK_VIDEO_RESOLUTIONS = ['480p', '720p', '1080p'] as const

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
  const attachments = usePromptInputAttachments()
  if (props.config.mode === 'chat') return null

  const isImage = props.config.mode === 'image'
  const isGrokImage = isImage && isGrokImageModel(props.config.model)
  const hasVideoReference =
    attachments.files.length > 0 ||
    props.config.video_reference_url.trim().length > 0
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
        className='w-[25rem] max-w-[calc(100vw-2rem)] space-y-4 p-4'
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
            {isGrokImage ? (
              <>
                <div className='border-border/70 bg-muted/20 rounded-lg border px-3 py-2 text-[11px] leading-4'>
                  <span className='font-medium'>
                    {t('Grok image controls')}
                  </span>
                  <p className='text-muted-foreground mt-0.5'>
                    {t(
                      'Synchronized with the Grok2API creative console. Images use URL responses and non-streaming generation.'
                    )}
                  </p>
                </div>
                <MediaSelect
                  icon={<ImagePlusIcon size={12} />}
                  label={t('Image count')}
                  value={String(
                    Math.min(4, Math.max(1, Math.round(props.config.image_n)))
                  )}
                  options={[...GROK_IMAGE_COUNTS]}
                  optionLabel={(value) => `${value}×`}
                  onChange={(value) =>
                    props.onConfigChange('image_n', Number(value))
                  }
                />
                <MediaSelect
                  icon={<RatioIcon size={12} />}
                  label={t('Image aspect ratio')}
                  value={props.config.image_aspect_ratio}
                  options={[...GROK_IMAGE_ASPECT_RATIOS]}
                  onChange={(value) =>
                    props.onConfigChange('image_aspect_ratio', value)
                  }
                />
                <MediaSelect
                  icon={<MonitorUpIcon size={12} />}
                  label={t('Image resolution')}
                  value={props.config.image_resolution}
                  options={[...GROK_IMAGE_RESOLUTIONS]}
                  onChange={(value) =>
                    props.onConfigChange('image_resolution', value)
                  }
                />
              </>
            ) : (
              <>
                <MediaSelect
                  label={t('Image size')}
                  value={props.config.image_size}
                  options={['1024x1024', '1536x1024', '1024x1536']}
                  onChange={(value) =>
                    props.onConfigChange('image_size', value)
                  }
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
                  <span className='text-muted-foreground'>
                    {t('Image count')}
                  </span>
                  <Input
                    type='number'
                    min={1}
                    max={10}
                    value={props.config.image_n}
                    onChange={(event) =>
                      props.onConfigChange(
                        'image_n',
                        Math.min(
                          10,
                          Math.max(1, Number(event.target.value) || 1)
                        )
                      )
                    }
                  />
                </label>
              </>
            )}
          </div>
        ) : (
          <div className='grid gap-4'>
            <div className='border-border/70 bg-muted/20 grid gap-3 rounded-xl border p-3'>
              <div className='flex items-start justify-between gap-3'>
                <div className='flex min-w-0 items-start gap-2.5'>
                  <div className='bg-primary/10 text-primary mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-lg'>
                    <ImagePlusIcon size={16} />
                  </div>
                  <div className='min-w-0'>
                    <div className='text-xs font-semibold'>
                      {t('Reference image')}
                    </div>
                    <p className='text-muted-foreground mt-0.5 text-[11px] leading-4'>
                      {t(
                        'Upload, paste, drop, or provide a URL. A local image overrides the URL.'
                      )}
                    </p>
                  </div>
                </div>
                <span className='bg-background text-muted-foreground shrink-0 rounded-md border px-1.5 py-0.5 text-[10px] font-medium'>
                  {hasVideoReference ? t('Ready') : t('Optional')}
                </span>
              </div>

              <Button
                className='w-full justify-center gap-2'
                disabled={props.disabled}
                onClick={attachments.openFileDialog}
                size='sm'
                type='button'
                variant='outline'
              >
                <MonitorUpIcon size={14} />
                {attachments.files.length > 0
                  ? t('Replace reference image')
                  : t('Upload reference image')}
              </Button>

              <label className='grid gap-1.5 text-xs'>
                <span className='text-muted-foreground inline-flex items-center gap-1.5'>
                  <LinkIcon size={12} />
                  {t('Reference image URL')}
                </span>
                <div className='flex items-center gap-2'>
                  <Input
                    inputMode='url'
                    onChange={(event) =>
                      props.onConfigChange(
                        'video_reference_url',
                        event.target.value
                      )
                    }
                    placeholder='https://example.com/reference.png'
                    type='url'
                    value={props.config.video_reference_url}
                  />
                  {props.config.video_reference_url ? (
                    <Button
                      aria-label={t('Clear')}
                      className='shrink-0'
                      onClick={() =>
                        props.onConfigChange('video_reference_url', '')
                      }
                      size='icon-sm'
                      type='button'
                      variant='ghost'
                    >
                      <XIcon size={14} />
                    </Button>
                  ) : null}
                </div>
              </label>
            </div>

            <MediaSelect
              icon={<Clock3Icon size={12} />}
              label={t('Video duration')}
              value={String(props.config.video_duration)}
              options={[...GROK_VIDEO_DURATIONS]}
              optionLabel={(value) => t('{{value}} seconds', { value })}
              onChange={(value) =>
                props.onConfigChange('video_duration', Number(value))
              }
            />
            <MediaSelect
              icon={<RatioIcon size={12} />}
              label={t('Video aspect ratio')}
              value={props.config.video_aspect_ratio}
              options={[...GROK_VIDEO_ASPECT_RATIOS]}
              onChange={(value) =>
                props.onConfigChange('video_aspect_ratio', value)
              }
            />
            <MediaSelect
              icon={<MonitorUpIcon size={12} />}
              label={t('Video resolution')}
              value={props.config.video_resolution}
              options={[...GROK_VIDEO_RESOLUTIONS]}
              onChange={(value) =>
                props.onConfigChange('video_resolution', value)
              }
            />
          </div>
        )}
      </PopoverContent>
    </Popover>
  )
}

function MediaSelect(props: {
  icon?: ReactNode
  label: string
  value: string
  options: string[]
  optionLabel?: (value: string) => ReactNode
  onChange: (value: string) => void
}) {
  return (
    <label className='grid gap-1.5 text-xs'>
      <span className='text-muted-foreground inline-flex items-center gap-1.5'>
        {props.icon}
        {props.label}
      </span>
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
                {props.optionLabel?.(option) ?? option}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    </label>
  )
}
