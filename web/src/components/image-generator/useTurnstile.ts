import { useCallback, useEffect, useRef, useState } from "react"

import {
  getTurnstileConfig,
  isTurnstileVerified,
  verifyTurnstileToken,
} from "@/lib/http"

type TurnstileAPI = {
  render: (
    container: HTMLElement,
    options: {
      sitekey: string
      execution: "execute"
      appearance: "interaction-only"
      callback: (token: string) => void
      "error-callback": () => void
      "expired-callback": () => void
      "timeout-callback": () => void
      "before-interactive-callback": () => void
      "after-interactive-callback": () => void
    }
  ) => string
  execute: (widgetId: string) => void
  reset: (widgetId: string) => void
  remove: (widgetId: string) => void
}

declare global {
  interface Window {
    turnstile?: TurnstileAPI
    sceneMintTurnstileLoaded?: () => void
  }
}

let scriptPromise: Promise<void> | undefined
const scriptTimeoutMs = 15_000
const tokenTimeoutMs = 30_000

type TokenRequest = {
  resolve: (token: string) => void
  reject: (err: Error) => void
  timeout: number
}

type ReadyRequest = {
  resolve: () => void
  reject: (err: Error) => void
  timeout: number
}

export function useTurnstile() {
  const containerRef = useRef<HTMLDivElement>(null)
  const widgetRef = useRef<string>("")
  const tokenRequestRef = useRef<TokenRequest | null>(null)
  const readyRequestsRef = useRef<ReadyRequest[]>([])
  const verificationPromiseRef = useRef<Promise<void> | null>(null)
  const stateRef = useRef({
    configReady: false,
    enabled: false,
    ready: false,
    error: "",
  })
  const [configReady, setConfigReady] = useState(false)
  const [enabled, setEnabled] = useState(false)
  const [siteKey, setSiteKey] = useState("")
  const [ready, setReady] = useState(false)
  const [error, setError] = useState("")
  const [interactive, setInteractive] = useState(false)

  const settleReadyRequests = useCallback((err?: Error) => {
    if (!err) {
      const state = stateRef.current
      if (!state.configReady) {
        return
      }
      if (state.error) {
        err = new Error(state.error)
      } else if (
        state.enabled &&
        !isTurnstileVerified() &&
        !state.ready
      ) {
        return
      }
    }

    const requests = readyRequestsRef.current
    readyRequestsRef.current = []
    for (const request of requests) {
      window.clearTimeout(request.timeout)
      if (err) {
        request.reject(err)
      } else {
        request.resolve()
      }
    }
  }, [])

  const completeTokenRequest = useCallback((token?: string, err?: Error) => {
    setInteractive(false)
    const request = tokenRequestRef.current
    if (!request) {
      return
    }
    window.clearTimeout(request.timeout)
    tokenRequestRef.current = null
    if (err) {
      request.reject(err)
      return
    }
    if (!token) {
      request.reject(new Error("人机校验失败，请重试"))
      return
    }
    request.resolve(token)
  }, [])

  useEffect(() => {
    stateRef.current = {
      configReady,
      enabled,
      ready,
      error,
    }
    settleReadyRequests()
  }, [configReady, enabled, error, ready, settleReadyRequests])

  useEffect(() => {
    return () => {
      settleReadyRequests(new Error("人机校验已取消"))
    }
  }, [settleReadyRequests])

  useEffect(() => {
    let disposed = false

    async function prepare() {
      try {
        const config = await getTurnstileConfig()
        if (disposed) {
          return
        }
        setEnabled(config.enabled)
        setSiteKey(config.siteKey)
        setError(config.enabled && !config.siteKey ? "人机校验未配置" : "")
        setConfigReady(true)
        if (!config.enabled || config.verifiedUntil > Date.now()) {
          setReady(true)
        }
      } catch (err) {
        if (!disposed) {
          setError(err instanceof Error ? err.message : "人机校验配置读取失败")
          setConfigReady(true)
        }
      }
    }

    void prepare()
    return () => {
      disposed = true
    }
  }, [])

  useEffect(() => {
    if (!enabled || !siteKey || !containerRef.current) {
      return
    }
    let disposed = false

    async function renderWidget() {
      try {
        await loadTurnstileScript()
        if (disposed || !containerRef.current || !window.turnstile) {
          return
        }
        const widgetId = window.turnstile.render(containerRef.current, {
          sitekey: siteKey,
          execution: "execute",
          appearance: "interaction-only",
          callback: (token) => {
            completeTokenRequest(token)
          },
          "error-callback": () => {
            completeTokenRequest(undefined, new Error("人机校验失败，请重试"))
          },
          "expired-callback": () => {
            completeTokenRequest(undefined, new Error("人机校验已过期，请重试"))
          },
          "timeout-callback": () => {
            completeTokenRequest(undefined, new Error("人机校验超时，请重试"))
          },
          "before-interactive-callback": () => {
            setInteractive(true)
          },
          "after-interactive-callback": () => {
            setInteractive(false)
          },
        })
        widgetRef.current = widgetId
        setError("")
        setReady(true)
      } catch (err) {
        if (!disposed) {
          setError(err instanceof Error ? err.message : "人机校验脚本加载失败")
        }
      }
    }

    void renderWidget()
    return () => {
      disposed = true
      if (widgetRef.current && window.turnstile) {
        window.turnstile.remove(widgetRef.current)
      }
      widgetRef.current = ""
      completeTokenRequest(undefined, new Error("人机校验已取消"))
      setInteractive(false)
      setReady(false)
    }
  }, [completeTokenRequest, enabled, siteKey])

  const waitUntilReady = useCallback(() => {
    const state = stateRef.current
    if (state.error) {
      return Promise.reject(new Error(state.error))
    }
    if (
      state.configReady &&
      (!state.enabled || isTurnstileVerified() || state.ready)
    ) {
      return Promise.resolve()
    }

    return new Promise<void>((resolve, reject) => {
      const timeout = window.setTimeout(() => {
        readyRequestsRef.current = readyRequestsRef.current.filter(
          (request) => request.resolve !== resolve
        )
        reject(new Error("人机校验尚未就绪，请重试"))
      }, scriptTimeoutMs)
      readyRequestsRef.current.push({ resolve, reject, timeout })
      settleReadyRequests()
    })
  }, [settleReadyRequests])

  const requestToken = useCallback(() => {
    const widgetId = widgetRef.current
    if (!stateRef.current.ready || !widgetId || !window.turnstile) {
      throw new Error("人机校验尚未就绪")
    }
    if (tokenRequestRef.current) {
      throw new Error("人机校验正在进行")
    }

    window.turnstile.reset(widgetId)
    setInteractive(false)
    return new Promise<string>((resolve, reject) => {
      const timeout = window.setTimeout(() => {
        if (tokenRequestRef.current) {
          completeTokenRequest(undefined, new Error("人机校验超时，请重试"))
          window.turnstile?.reset(widgetId)
        }
      }, tokenTimeoutMs)
      tokenRequestRef.current = { resolve, reject, timeout }
      window.turnstile?.execute(widgetId)
    })
  }, [completeTokenRequest])

  const ensureVerified = useCallback(async () => {
    await waitUntilReady()

    const state = stateRef.current
    if (isTurnstileVerified() || !state.enabled) {
      return
    }
    if (state.error) {
      throw new Error(state.error)
    }
    if (verificationPromiseRef.current) {
      await verificationPromiseRef.current
      return
    }

    const verificationPromise = requestToken().then(verifyTurnstileToken)
    verificationPromiseRef.current = verificationPromise
    try {
      await verificationPromise
    } finally {
      if (verificationPromiseRef.current === verificationPromise) {
        verificationPromiseRef.current = null
      }
    }
  }, [requestToken, waitUntilReady])

  return {
    interactive,
    containerRef,
    ensureVerified,
  }
}

function loadTurnstileScript() {
  if (window.turnstile) {
    return Promise.resolve()
  }
  if (scriptPromise) {
    return scriptPromise
  }

  scriptPromise = new Promise<void>((resolve, reject) => {
    let settled = false
    let timeout = 0
    function finish(err?: Error) {
      if (settled) {
        return
      }
      settled = true
      window.clearTimeout(timeout)
      if (window.sceneMintTurnstileLoaded === onLoad) {
        delete window.sceneMintTurnstileLoaded
      }
      if (err) {
        scriptPromise = undefined
        reject(err)
        return
      }
      resolve()
    }

    function onLoad() {
      finish()
    }

    window.sceneMintTurnstileLoaded = onLoad

    const script = document.createElement("script")
    script.src =
      "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit&onload=sceneMintTurnstileLoaded"
    script.async = true
    script.defer = true
    script.onerror = () => {
      finish(new Error("人机校验脚本加载失败，请刷新页面后重试"))
    }
    timeout = window.setTimeout(() => {
      script.remove()
      finish(new Error("人机校验脚本加载超时，请刷新页面后重试"))
    }, scriptTimeoutMs)
    document.head.appendChild(script)
  })

  return scriptPromise
}
