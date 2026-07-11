import { CircleGaugeIcon, GiftIcon, Loader2Icon, RotateCcwIcon } from "lucide-react"
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
  loading: boolean
  signingIn: boolean
  error: string
  onCheckIn: () => void
  onRetry: () => void
}

export function QuotaPanel({
  t,
  status,
  loading,
  signingIn,
  error,
  onCheckIn,
  onRetry,
}: QuotaPanelProps) {
  const balance = status?.balance ?? 0
  const cap = status?.cap ?? 100
  const fill = cap > 0 ? Math.max(0, Math.min(100, (balance / cap) * 100)) : 0
  const atCap = Boolean(status && status.balance >= status.cap)
  const showCheckIn = Boolean(
    status && !status.signedToday && !atCap
  )
  return (
    <section className="scene-quota-panel rounded-lg border p-2.5" aria-label={t.quota.remaining}>
      <div className="flex items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2">
          <span className="scene-quota-icon flex size-7 shrink-0 items-center justify-center rounded-md">
            <CircleGaugeIcon aria-hidden="true" />
          </span>
          <div className="flex min-w-0 items-baseline gap-2">
            <span className="text-[0.68rem] font-medium text-muted-foreground">
              {t.quota.remaining}
            </span>
            {loading ? (
              <Skeleton className="h-4 w-14" />
            ) : (
              <span className="flex items-baseline gap-1">
                <span className="text-lg leading-none font-semibold tabular-nums">{balance}</span>
                <span className="text-xs text-muted-foreground">/ {cap}</span>
              </span>
            )}
          </div>
        </div>

        {error ? (
          <Button type="button" size="xs" variant="outline" onClick={onRetry} disabled={loading}>
            <RotateCcwIcon data-icon="inline-start" />
            {t.quota.retry}
          </Button>
        ) : showCheckIn ? (
          <Button
            type="button"
            size="xs"
            variant="outline"
            disabled={signingIn || atCap}
            onClick={onCheckIn}
          >
            {signingIn ? (
              <Loader2Icon data-icon="inline-start" className="animate-spin" />
            ) : (
              <GiftIcon data-icon="inline-start" />
            )}
            {t.quota.checkIn} +{status?.dailyGrant ?? 0}
          </Button>
        ) : status?.signedToday || atCap ? (
          <span className="scene-quota-state">
            {status?.signedToday ? t.quota.signed : t.quota.full}
          </span>
        ) : null}
      </div>

      <div className="scene-quota-meter mt-1.5" aria-hidden="true">
        <span style={{ "--quota-fill": `${fill}%` } as CSSProperties} />
      </div>
      {error && (
        <p className="mt-1 truncate text-[0.68rem] leading-4 text-destructive" title={error}>
          {error}
        </p>
      )}
    </section>
  )
}
