import { useCallback, useEffect, useState } from "react"
import { toast } from "sonner"

import { getVisitorFingerprint } from "@/lib/fingerprint"
import { postJSON } from "@/lib/http"

import type { ImageGeneratorCopy } from "./copy"
import type { QuotaStatus } from "./QuotaPanel"

type UseQuotaOptions = {
  ensureTurnstileVerified?: () => Promise<void>
}

export function useQuota(t: ImageGeneratorCopy, options: UseQuotaOptions = {}) {
  const { ensureTurnstileVerified } = options
  const [fingerprint, setFingerprint] = useState("")
  const [quotaStatus, setQuotaStatus] = useState<QuotaStatus | null>(null)
  const [quotaLoading, setQuotaLoading] = useState(true)
  const [quotaError, setQuotaError] = useState("")
  const [signingIn, setSigningIn] = useState(false)

  const quotaReady = Boolean(fingerprint && quotaStatus && !quotaLoading)
  const hasCredits = (quotaStatus?.balance ?? 0) > 0

  const refreshQuota = useCallback(
    async (fingerprintValue = fingerprint) => {
      if (!fingerprintValue) {
        return undefined
      }
      setQuotaLoading(true)
      try {
        const status = await postJSON<QuotaStatus>("/api/quota/status", {
          fingerprint: fingerprintValue,
        })
        setQuotaStatus(status)
        setQuotaError("")
        return status
      } catch (err) {
        setQuotaError(err instanceof Error ? err.message : t.quota.loadFailed)
        return undefined
      } finally {
        setQuotaLoading(false)
      }
    },
    [fingerprint, t.quota.loadFailed]
  )

  useEffect(() => {
    let disposed = false

    async function prepareFingerprint() {
      setQuotaLoading(true)
      try {
        const visitorId = await getVisitorFingerprint()
        if (!disposed) {
          setFingerprint(visitorId)
        }
      } catch (err) {
        if (!disposed) {
          setQuotaError(
            err instanceof Error ? err.message : t.quota.fingerprintFailed
          )
          setQuotaLoading(false)
        }
      }
    }

    void prepareFingerprint()
    return () => {
      disposed = true
    }
  }, [])

  useEffect(() => {
    if (!fingerprint) {
      return
    }
    void refreshQuota()
  }, [fingerprint, refreshQuota])

  const checkIn = useCallback(async () => {
    if (!fingerprint || signingIn) {
      return
    }
    setSigningIn(true)
    try {
      const previous = quotaStatus
      await ensureTurnstileVerified?.()
      const status = await postJSON<QuotaStatus>(
        "/api/quota/check-in",
        {
          fingerprint,
        }
      )
      setQuotaStatus(status)
      setQuotaError("")
      if (status.balance >= status.cap && !status.signedToday) {
        toast.info(t.quota.fullMessage)
      } else if (previous?.signedToday || status.balance === previous?.balance) {
        toast.info(t.quota.alreadySigned)
      } else {
        toast.success(t.quota.checkInSuccess(status.balance))
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err))
    } finally {
      setSigningIn(false)
    }
  }, [ensureTurnstileVerified, fingerprint, quotaStatus, signingIn, t.quota])

  const retryQuota = useCallback(async () => {
    if (fingerprint) {
      await refreshQuota()
      return
    }
    setQuotaLoading(true)
    setQuotaError("")
    try {
      const visitorId = await getVisitorFingerprint()
      setFingerprint(visitorId)
    } catch (err) {
      setQuotaError(
        err instanceof Error ? err.message : t.quota.fingerprintFailed
      )
      setQuotaLoading(false)
    }
  }, [fingerprint, refreshQuota, t.quota.fingerprintFailed])

  const applyRemainingCredits = useCallback((remainingCredits: number) => {
    setQuotaStatus((current) =>
      current ? { ...current, balance: remainingCredits } : current
    )
  }, [])

  return {
    fingerprint,
    quotaStatus,
    quotaLoading,
    quotaError,
    signingIn,
    quotaReady,
    hasCredits,
    checkIn,
    retryQuota,
    applyRemainingCredits,
  }
}
