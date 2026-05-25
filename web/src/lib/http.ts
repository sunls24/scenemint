type APIEnvelope<T> = {
  code: number
  message: string
  data?: T
}

type SessionResponse = {
  csrfToken: string
}

type PostOptions = {
  refreshed?: boolean
  signal?: AbortSignal
}

let csrfToken: string | undefined
let csrfTokenPromise: Promise<string> | undefined

async function readEnvelope<T>(resp: Response): Promise<APIEnvelope<T> | undefined> {
  const text = await resp.text()
  try {
    return text ? (JSON.parse(text) as APIEnvelope<T>) : undefined
  } catch {
    return undefined
  }
}

function dataOrThrow<T>(resp: Response, payload: APIEnvelope<T> | undefined): T {
  if (!payload) {
    if (!resp.ok) {
      throw new Error(`${resp.status} ${resp.statusText}`)
    }
    throw new Error("接口响应格式不正确")
  }
  if (typeof payload.code !== "number") {
    if (!resp.ok) {
      throw new Error(`${resp.status} ${resp.statusText}`)
    }
    throw new Error("接口响应格式不正确")
  }
  if (!resp.ok || payload.code !== 0) {
    throw new Error(payload.message || `${resp.status} ${resp.statusText}`)
  }
  return payload.data as T
}

async function fetchCSRFToken(refresh = false): Promise<string> {
  if (!refresh && csrfToken) {
    return csrfToken
  }
  if (!refresh && csrfTokenPromise) {
    return csrfTokenPromise
  }

  csrfTokenPromise = (async () => {
    const resp = await fetch("/api/session", {
      credentials: "same-origin",
    })
    const payload = await readEnvelope<SessionResponse>(resp)
    const data = dataOrThrow(resp, payload)
    if (!data?.csrfToken) {
      throw new Error("会话响应格式不正确")
    }
    csrfToken = data.csrfToken
    return data.csrfToken
  })()

  try {
    return await csrfTokenPromise
  } finally {
    csrfTokenPromise = undefined
  }
}

export async function postJSON<T>(path: string, body: unknown): Promise<T> {
  const resp = await postWithCSRF(path, body)
  const payload = await readEnvelope<T>(resp)
  return dataOrThrow(resp, payload)
}

export async function postForm<T>(path: string, body: FormData): Promise<T> {
  const resp = await postWithCSRF(path, body)
  const payload = await readEnvelope<T>(resp)
  return dataOrThrow(resp, payload)
}

export async function postStream(
  path: string,
  body: unknown,
  signal?: AbortSignal
): Promise<ReadableStream<Uint8Array>> {
  const resp = await postWithCSRF(path, body, { signal })
  const contentType = resp.headers.get("Content-Type") ?? ""
  if (!resp.ok || !contentType.includes("text/event-stream")) {
    const payload = await readEnvelope<unknown>(resp)
    dataOrThrow(resp, payload)
    throw new Error(`${resp.status} ${resp.statusText}`)
  }
  if (!resp.body) {
    throw new Error("流式响应不可用")
  }
  return resp.body
}

async function postWithCSRF(
  path: string,
  body: unknown | FormData,
  options: PostOptions = {}
): Promise<Response> {
  const token = await fetchCSRFToken(options.refreshed)
  const formBody = body instanceof FormData
  const headers: Record<string, string> = {
    "X-CSRF-Token": token,
  }
  if (!formBody) {
    headers["Content-Type"] = "application/json"
  }

  const resp = await fetch(path, {
    method: "POST",
    credentials: "same-origin",
    headers,
    body: formBody ? body : JSON.stringify(body),
    signal: options.signal,
  })

  if (resp.status === 403 && !options.refreshed) {
    csrfToken = undefined
    await resp.body?.cancel()
    return postWithCSRF(path, body, {
      ...options,
      refreshed: true,
    })
  }
  return resp
}

export async function getJSON<T>(path: string): Promise<T> {
  const resp = await fetch(path, {
    credentials: "same-origin",
  })
  const payload = await readEnvelope<T>(resp)
  return dataOrThrow(resp, payload)
}
