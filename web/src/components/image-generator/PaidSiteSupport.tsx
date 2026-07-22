import {
  HeartHandshakeIcon,
  Loader2Icon,
  ShieldAlertIcon,
  ThumbsUpIcon,
} from "lucide-react"
import { useState } from "react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { getVisitorFingerprint } from "@/lib/fingerprint"
import { postJSON } from "@/lib/http"

import type { ImageGeneratorCopy } from "./copy"

const paidSiteSupportStorageKey = "scenemint:paidSiteSupport"

type PaidSiteSupportProps = {
  t: ImageGeneratorCopy
}

type VoteResponse = {
  count: number
  recorded: boolean
}

function hasVoted() {
  if (typeof window === "undefined") return false
  try {
    return window.localStorage.getItem(paidSiteSupportStorageKey) === "1"
  } catch {
    return false
  }
}

export function PaidSiteSupport({ t }: PaidSiteSupportProps) {
  const [voted, setVoted] = useState(hasVoted)
  const [voting, setVoting] = useState(false)

  async function vote() {
    if (voted || voting) return

    setVoting(true)
    try {
      const fingerprint = await getVisitorFingerprint()
      await postJSON<VoteResponse>("/api/vote", {
        fingerprint,
      })
      try {
        window.localStorage.setItem(paidSiteSupportStorageKey, "1")
      } catch {
        // The vote still succeeded when browser storage is unavailable.
      }
      setVoted(true)
      toast.success(t.paidSiteSupport.thanks)
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t.paidSiteSupport.failed
      )
    } finally {
      setVoting(false)
    }
  }

  return (
    <div className="scene-header-notice flex w-full flex-col gap-1 rounded-md px-2 py-1.5 md:w-auto lg:flex-row lg:items-center lg:gap-2">
      <div className="flex items-start gap-1.5" role="note">
        <ShieldAlertIcon className="mt-0.5 size-3.5 shrink-0" aria-hidden="true" />
        <p className="text-[0.7rem] leading-4 font-semibold lg:whitespace-nowrap">{t.contentPolicyNotice}</p>
      </div>
      {!voted && (
        <div className="flex items-center gap-1.5">
          <HeartHandshakeIcon className="size-3.5 shrink-0" aria-hidden="true" />
          <span className="min-w-0 flex-1 text-[0.7rem] font-semibold md:flex-none md:whitespace-nowrap">
            {t.paidSiteSupport.question}
          </span>
          <Button
            type="button"
            variant="ghost"
            size="xs"
            onClick={() => void vote()}
            disabled={voting}
            className="scene-support-button"
          >
            {voting ? (
              <Loader2Icon data-icon="inline-start" className="animate-spin" />
            ) : (
              <ThumbsUpIcon data-icon="inline-start" />
            )}
            {t.paidSiteSupport.action}
          </Button>
        </div>
      )}
    </div>
  )
}
