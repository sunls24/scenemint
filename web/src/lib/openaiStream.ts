type ChatCompletionStreamChunk = {
  choices?: Array<{
    delta?: {
      content?: string
    }
  }>
  error?: {
    message?: string
  }
}

export async function readChatCompletionStream(
  stream: ReadableStream<Uint8Array>,
  onDelta: (delta: string) => void
) {
  const reader = stream.getReader()
  const decoder = new TextDecoder()
  let buffer = ""

  try {
    while (true) {
      const { value, done } = await reader.read()
      if (done) {
        break
      }
      buffer += decoder.decode(value, { stream: true })
      buffer = drainSSEBuffer(buffer, onDelta)
    }
    buffer += decoder.decode()
    if (buffer.trim()) {
      processSSEFrame(buffer, onDelta)
    }
  } finally {
    reader.releaseLock()
  }
}

function drainSSEBuffer(
  buffer: string,
  onDelta: (delta: string) => void
): string {
  let next = buffer
  while (true) {
    const end = sseFrameEnd(next)
    if (!end) {
      return next
    }
    processSSEFrame(next.slice(0, end.index), onDelta)
    next = next.slice(end.index + end.length)
  }
}

function sseFrameEnd(buffer: string): { index: number; length: number } | null {
  const lf = buffer.indexOf("\n\n")
  const crlf = buffer.indexOf("\r\n\r\n")
  if (lf < 0) {
    return crlf < 0 ? null : { index: crlf, length: 4 }
  }
  if (crlf < 0 || lf < crlf) {
    return { index: lf, length: 2 }
  }
  return { index: crlf, length: 4 }
}

function processSSEFrame(frame: string, onDelta: (delta: string) => void) {
  const data = frame
    .split(/\r?\n/)
    .filter((line) => line.startsWith("data:"))
    .map((line) => line.slice(5).trimStart())
    .join("\n")
    .trim()

  if (!data || data === "[DONE]") {
    return
  }

  const chunk = JSON.parse(data) as ChatCompletionStreamChunk
  if (chunk.error?.message) {
    throw new Error(chunk.error.message)
  }
  for (const choice of chunk.choices ?? []) {
    const content = choice.delta?.content
    if (content) {
      onDelta(content)
    }
  }
}
