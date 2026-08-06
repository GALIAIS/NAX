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

export type MediaSampleLanguage =
  | 'curl'
  | 'python'
  | 'typescript'
  | 'javascript'

export type MediaSampleContext = {
  baseUrl: string
  apiKeyEnv: string
  modelName: string
  endpointPath: string
}

function imageRequestBody(ctx: MediaSampleContext): Record<string, unknown> {
  const grokImagine = /grok.*imagine/i.test(ctx.modelName)
  return {
    model: ctx.modelName,
    prompt: 'A serene koi pond at sunset, ukiyo-e style.',
    ...(grokImagine
      ? { aspect_ratio: '1:1', resolution: '1k' }
      : { size: '1024x1024', quality: 'standard' }),
    n: 1,
    response_format: 'url',
  }
}

function videoRequestBody(ctx: MediaSampleContext): Record<string, unknown> {
  return {
    model: ctx.modelName,
    prompt: 'Ocean waves roll onto a moonlit beach, cinematic camera motion.',
    duration: 6,
    aspect_ratio: '16:9',
    resolution: '720p',
    image: { url: 'https://example.com/reference.png' },
  }
}

function videoStatusPath(endpointPath: string): string {
  const normalized = endpointPath.replace(/\/+$/, '')
  return normalized.endsWith('/generations')
    ? normalized.slice(0, -'/generations'.length)
    : normalized
}

export function buildImageApiSample(
  lang: MediaSampleLanguage,
  ctx: MediaSampleContext
): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}`
  const body = imageRequestBody(ctx)
  const bodyJson = JSON.stringify(body, null, 2)

  if (lang === 'curl') {
    return [
      `curl ${url} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${bodyJson.replaceAll('\n', '\n     ')}'`,
    ].join('\n')
  }

  if (lang === 'python') {
    return [
      'import os',
      'import requests',
      '',
      `response = requests.post(`,
      `    "${url}",`,
      `    headers={"Authorization": f"Bearer {os.environ['${ctx.apiKeyEnv}']}"},`,
      `    json=${JSON.stringify(body, null, 4).replaceAll(/^/gm, '    ').trimStart()},`,
      `    timeout=120,`,
      ')',
      'response.raise_for_status()',
      'image = response.json()["data"][0]',
      'print(image.get("url") or image.get("b64_json"))',
    ].join('\n')
  }

  const typeAnnotation =
    lang === 'typescript'
      ? ` as { data: Array<{ url?: string; b64_json?: string }> }`
      : ''
  return [
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: {`,
    `    Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `    'Content-Type': 'application/json',`,
    `  },`,
    `  body: JSON.stringify(${bodyJson.replaceAll(/^/gm, '  ').trimStart()}),`,
    `})`,
    `if (!response.ok) throw new Error(await response.text())`,
    `const result = (await response.json())${typeAnnotation}`,
    `console.log(result.data[0].url ?? result.data[0].b64_json)`,
  ].join('\n')
}

export function buildVideoApiSample(
  lang: MediaSampleLanguage,
  ctx: MediaSampleContext
): string {
  const createUrl = `${ctx.baseUrl}${ctx.endpointPath}`
  const statusBaseUrl = `${ctx.baseUrl}${videoStatusPath(ctx.endpointPath)}`
  const body = videoRequestBody(ctx)
  const bodyJson = JSON.stringify(body, null, 2)

  if (lang === 'curl') {
    return [
      '# 1. Submit an asynchronous video job',
      `curl ${createUrl} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${bodyJson.replaceAll('\n', '\n     ')}'`,
      '',
      '# 2. Poll with the id / request_id / task_id returned above',
      `curl ${statusBaseUrl}/<REQUEST_ID> \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}"`,
    ].join('\n')
  }

  if (lang === 'python') {
    return [
      'import os',
      'import time',
      'import requests',
      '',
      `headers = {"Authorization": f"Bearer {os.environ['${ctx.apiKeyEnv}']}"}`,
      `job_response = requests.post(`,
      `    "${createUrl}",`,
      `    headers=headers,`,
      `    json=${JSON.stringify(body, null, 4).replaceAll(/^/gm, '    ').trimStart()},`,
      `    timeout=120,`,
      ')',
      'job_response.raise_for_status()',
      'job = job_response.json()',
      'request_id = job.get("id") or job.get("request_id") or job.get("task_id")',
      'if not request_id:',
      '    raise RuntimeError("Video response did not contain a task id")',
      '',
      'while True:',
      `    response = requests.get(f"${statusBaseUrl}/{request_id}", headers=headers, timeout=60)`,
      '    response.raise_for_status()',
      '    status = response.json()',
      '    state = str(status.get("status", "")).lower()',
      '    if state in {"done", "completed", "succeeded", "success"}:',
      '        video = status.get("video") or {}',
      '        print(video.get("url") or status.get("url") or status.get("output_url"))',
      '        break',
      '    if state in {"failed", "error", "cancelled", "canceled"}:',
      '        raise RuntimeError(str(status.get("error") or status))',
      '    time.sleep(3)',
    ].join('\n')
  }

  const jobType =
    lang === 'typescript'
      ? ` as { id?: string; request_id?: string; task_id?: string }`
      : ''
  const statusType =
    lang === 'typescript'
      ? ` as { status?: string; video?: { url?: string }; url?: string; output_url?: string; error?: unknown }`
      : ''
  return [
    `const headers = {`,
    `  Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `  'Content-Type': 'application/json',`,
    `}`,
    '',
    `const jobResponse = await fetch('${createUrl}', {`,
    `  method: 'POST',`,
    `  headers,`,
    `  body: JSON.stringify(${bodyJson.replaceAll(/^/gm, '  ').trimStart()}),`,
    `})`,
    `if (!jobResponse.ok) throw new Error(await jobResponse.text())`,
    `const job = (await jobResponse.json())${jobType}`,
    `const requestId = job.id ?? job.request_id ?? job.task_id`,
    `if (!requestId) throw new Error('Video response did not contain a task id')`,
    '',
    `while (true) {`,
    `  await new Promise((resolve) => setTimeout(resolve, 3000))`,
    `  const response = await fetch(\`${statusBaseUrl}/\${encodeURIComponent(requestId)}\`, { headers })`,
    `  if (!response.ok) throw new Error(await response.text())`,
    `  const status = (await response.json())${statusType}`,
    `  const state = status.status?.toLowerCase() ?? ''`,
    `  if (['done', 'completed', 'succeeded', 'success'].includes(state)) {`,
    `    console.log(status.video?.url ?? status.url ?? status.output_url)`,
    `    break`,
    `  }`,
    `  if (['failed', 'error', 'cancelled', 'canceled'].includes(state)) {`,
    `    throw new Error(JSON.stringify(status.error ?? status))`,
    `  }`,
    `}`,
  ].join('\n')
}
