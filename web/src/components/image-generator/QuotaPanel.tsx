import {
  FingerprintIcon,
  GiftIcon,
  Loader2Icon,
  RotateCcwIcon,
  ScanLineIcon,
} from "lucide-react"
import type { CSSProperties } from "react"

import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"

import type { ImageGeneratorCopy } from "./copy"

export type QuotaStatus = {
  balance: number
  signedToday: boolean
  cap: number
  dailyGrant: number
}

type QuotaPanelProps = {
  t: ImageGeneratorCopy
  status: QuotaStatus | null
  fingerprint: string
  loading: boolean
  signingIn: boolean
  error: string
  onCheckIn: () => void
  onRetry: () => void
}

function fingerprintPreview(fingerprint: string) {
  if (fingerprint.length <= 12) {
    return fingerprint
  }
  return `${fingerprint.slice(0, 6)}...${fingerprint.slice(-4)}`
}

export function QuotaPanel({
  t,
  status,
  fingerprint,
  loading,
  signingIn,
  error,
  onCheckIn,
  onRetry,
}: QuotaPanelProps) {
  const balance = status?.balance ?? 0
  const cap = status?.cap ?? 100
  const description = status
    ? t.quota.description(status.dailyGrant)
    : t.quota.descriptionFallback
  const fill = cap > 0 ? Math.max(0, Math.min(100, (balance / cap) * 100)) : 0
  const atCap = Boolean(status && status.balance >= status.cap)
  const checkInDisabled =
    loading ||
    signingIn ||
    Boolean(error) ||
    !status ||
    status.signedToday
  const checkInLabel = signingIn
    ? t.quota.signingIn
    : status?.signedToday
      ? t.quota.signed
      : atCap
        ? t.quota.full
        : t.quota.checkIn

  const statusText = error || description

  return (
    <div className="scene-quota-strip rounded-lg border px-2.5 py-1.5">
      <div className="flex min-w-0 items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-1.5">
          <span className="scene-quota-strip-mark flex size-5 shrink-0 items-center justify-center rounded-md">
            {loading ? <ScanLineIcon /> : <GiftIcon />}
          </span>

          {loading ? (
            <Skeleton className="h-3.5 w-28 rounded-4xl" />
          ) : (
            <p className="min-w-0 truncate text-xs leading-none font-semibold text-foreground">
              {statusText}
            </p>
          )}
        </div>

        {error ? (
          <Button
            type="button"
            size="xs"
            variant="outline"
            disabled={loading}
            onClick={onRetry}
            className="shrink-0"
          >
            <RotateCcwIcon data-icon="inline-start" />
            {t.quota.retry}
          </Button>
        ) : (
          <Button
            type="button"
            size="xs"
            variant={status?.signedToday || atCap ? "secondary" : "default"}
            disabled={checkInDisabled}
            onClick={onCheckIn}
            className="shrink-0"
          >
            {signingIn ? (
              <Loader2Icon data-icon="inline-start" className="animate-spin" />
            ) : (
              <GiftIcon data-icon="inline-start" />
            )}
            {checkInLabel}
          </Button>
        )}
      </div>

      <div className="mt-1.5 flex min-h-[1.41rem] min-w-0 items-center gap-4">
        <div className="flex min-w-0 flex-1 items-center gap-1.5">
          <div className="scene-quota-meter min-w-8 flex-1" aria-hidden="true">
            <span style={{ "--quota-fill": `${fill}%` } as CSSProperties} />
          </div>
          {loading ? (
            <Skeleton className="h-3 w-10 shrink-0 rounded-4xl" />
          ) : (
            <span className="shrink-0 text-[0.66rem] leading-none font-medium text-muted-foreground tabular-nums">
              {balance}/{cap}
            </span>
          )}
        </div>

        {loading ? (
          <Skeleton className="h-[1.41rem] w-30 shrink-0 rounded-md" />
        ) : fingerprint ? (
          <span className="scene-fingerprint-preview inline-flex shrink-0 items-center gap-1.5 rounded-md px-2 py-1 text-[0.68rem] leading-none font-medium tabular-nums">
            <FingerprintIcon aria-hidden="true" />
            <span>{fingerprintPreview(fingerprint)}</span>
          </span>
        ) : (
          <span className="scene-fingerprint-preview inline-flex shrink-0 items-center gap-1.5 rounded-md px-2 py-1 text-[0.68rem] leading-none font-medium">
            <FingerprintIcon aria-hidden="true" />
            <span>{t.quota.unavailable}</span>
          </span>
        )}
      </div>
    </div>
  )
}
