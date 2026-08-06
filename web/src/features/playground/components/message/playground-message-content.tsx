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
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import {
  CodeBlock,
  CodeBlockCopyButton,
} from '@/components/ai-elements/code-block'
import { Loader } from '@/components/ai-elements/loader'
import { MessageContent } from '@/components/ai-elements/message'
import {
  Reasoning,
  ReasoningContent,
  ReasoningTrigger,
} from '@/components/ai-elements/reasoning'
import { Response } from '@/components/ai-elements/response'
import { Shimmer } from '@/components/ai-elements/shimmer'
import {
  Source,
  Sources,
  SourcesContent,
  SourcesTrigger,
} from '@/components/ai-elements/sources'
import { cn } from '@/lib/utils'

import { MESSAGE_STATUS } from '../../constants'
import {
  getMessageAlignmentClass,
  getMessageContentState,
  isErrorMessage,
  type MessageAlignment,
} from '../../lib'
import { getMessageContentStyles } from '../../lib/message/message-styles'
import type { Message } from '../../types'
import { MessageError } from './message-error'
import { MessageMetadata } from './message-metadata'

type PlaygroundMessageContentProps = {
  actions: ReactNode
  alignment: MessageAlignment
  errorActions?: ReactNode
  isSourceVisible?: boolean
  message: Message
  versionContent: string
}

export function PlaygroundMessageContent({
  actions,
  alignment,
  errorActions,
  isSourceVisible = false,
  message,
  versionContent,
}: PlaygroundMessageContentProps) {
  const { t } = useTranslation()
  const {
    displayContent,
    hasReasoning,
    hasSources,
    reasoningContent,
    showLoader,
    showMessageContent,
    sources,
  } = getMessageContentState(message, versionContent)
  const isError = isErrorMessage(message)
  const isMessageFinal =
    message.status !== MESSAGE_STATUS.LOADING &&
    message.status !== MESSAGE_STATUS.STREAMING
  const hasMedia = (message.media?.length ?? 0) > 0
  const hasInputReferences = (message.inputReferences?.length ?? 0) > 0

  let renderedContent: ReactNode = null
  if (isSourceVisible && showMessageContent) {
    renderedContent = (
      <CodeBlock
        code={versionContent}
        className='my-0 group-[.is-assistant]:w-full group-[.is-assistant]:max-w-[78ch]'
        collapsedLines={24}
        defaultCollapsed={false}
        language='markdown'
        maxExpandedLines={48}
        showLineNumbers
        showToolbar
        title={t('Raw response')}
      >
        <CodeBlockCopyButton />
      </CodeBlock>
    )
  } else if (!isSourceVisible && showMessageContent) {
    renderedContent = (
      <MessageContent variant='flat' className={cn(getMessageContentStyles())}>
        <Response final={isMessageFinal}>{displayContent}</Response>
      </MessageContent>
    )
  }

  return (
    <div
      className={cn(
        'flex w-full min-w-0 flex-col',
        getMessageAlignmentClass(alignment)
      )}
    >
      {hasSources && (
        <Sources>
          <SourcesTrigger count={sources.length} />
          <SourcesContent>
            {sources.map((source) => (
              <Source
                href={source.href}
                key={`${source.href}-${source.title}`}
                title={source.title}
              />
            ))}
          </SourcesContent>
        </Sources>
      )}

      {hasReasoning && (
        <Reasoning
          defaultOpen
          duration={message.reasoning?.duration}
          isStreaming={message.isReasoningStreaming}
        >
          <ReasoningTrigger />
          <ReasoningContent>{reasoningContent}</ReasoningContent>
        </Reasoning>
      )}

      {showLoader && (
        <div className='flex items-center gap-2 py-2'>
          <Loader />
          <Shimmer className='text-sm' duration={1}>
            {t('Responding...')}
          </Shimmer>
        </div>
      )}

      {isError && (
        <>
          <MessageError message={message} className='mb-2' />
          <MessageMetadata alignment={alignment} message={message} />
          {errorActions}
        </>
      )}

      {!isError && (showMessageContent || hasMedia || hasInputReferences) && (
        <>
          {hasInputReferences ? (
            <div className='mb-3 flex max-w-full flex-wrap gap-2'>
              {message.inputReferences?.map((media) => (
                <a
                  className='bg-muted/20 border-border/70 group relative flex max-w-64 items-center gap-2 overflow-hidden rounded-lg border p-1.5'
                  href={media.url}
                  key={media.url}
                  rel='noreferrer'
                  target='_blank'
                >
                  <img
                    alt={media.alt || t('Reference image')}
                    className='size-11 shrink-0 rounded-md object-cover'
                    src={media.url}
                  />
                  <span className='min-w-0 pr-2'>
                    <span className='text-foreground block truncate text-xs font-medium'>
                      {t('Reference image')}
                    </span>
                    <span className='text-muted-foreground block text-[10px]'>
                      {t('Video input')}
                    </span>
                  </span>
                </a>
              ))}
            </div>
          ) : null}
          {hasMedia ? (
            <div className='mb-3 grid gap-3 sm:grid-cols-2'>
              {message.media?.map((media) =>
                media.kind === 'image' ? (
                  <a
                    className='bg-muted/20 group relative overflow-hidden rounded-xl border'
                    href={media.url}
                    key={media.url}
                    rel='noreferrer'
                    target='_blank'
                  >
                    <img
                      alt={media.alt || t('Generated image')}
                      className='max-h-[min(60vh,32rem)] w-full object-contain transition-transform group-hover:scale-[1.01]'
                      loading='lazy'
                      src={media.url}
                    />
                    <span className='bg-background/80 text-muted-foreground absolute right-2 bottom-2 rounded px-2 py-1 text-[11px] opacity-0 backdrop-blur transition-opacity group-hover:opacity-100'>
                      {t('Open media')}
                    </span>
                  </a>
                ) : (
                  <video
                    className='bg-muted/20 max-h-[min(60vh,32rem)] w-full rounded-xl border object-contain'
                    controls
                    key={media.url}
                    preload='metadata'
                    src={media.url}
                  >
                    {t('Your browser does not support video playback.')}
                  </video>
                )
              )}
            </div>
          ) : null}
          {renderedContent}
          <MessageMetadata alignment={alignment} message={message} />
          {actions}
        </>
      )}
    </div>
  )
}
