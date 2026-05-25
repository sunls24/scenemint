import { expect, test } from "bun:test"

import { readChatCompletionStream } from "./openaiStream"

const encoder = new TextEncoder()

function streamFrom(chunks: string[]) {
  return new ReadableStream<Uint8Array>({
    start(controller) {
      for (const chunk of chunks) {
        controller.enqueue(encoder.encode(chunk))
      }
      controller.close()
    },
  })
}

test("reads split Chat Completions SSE frames", async () => {
  const deltas: string[] = []

  await readChatCompletionStream(
    streamFrom([
      'data: {"choices":[{"delta":{"content":"雨',
      '天"}}]}\r\n\r\n',
      'data: {"choices":[{"delta":{"content":"咖啡"}}]}\n\n',
      'data: {"choices":[{"delta":{"content":"馆"}}]}\n\n',
      "data: [DONE]\n\n",
    ]),
    (delta) => deltas.push(delta)
  )

  expect(deltas.join("")).toBe("雨天咖啡馆")
})

test("throws on upstream SSE error frames", async () => {
  let caught: unknown

  try {
    await readChatCompletionStream(
      streamFrom(['data: {"error":{"message":"上游失败"}}\n\n']),
      () => {}
    )
  } catch (err) {
    caught = err
  }

  expect(caught).toBeInstanceOf(Error)
  expect((caught as Error).message).toBe("上游失败")
})
