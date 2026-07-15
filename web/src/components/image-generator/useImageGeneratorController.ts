import { useEffect, useRef, useState } from "react"
import { toast } from "sonner"

import { copy, type Language } from "./copy"
import { initialLanguage, persistLanguage } from "./preferences"
import { useCreationDraft } from "./useCreationDraft"
import { useHistoryPreview } from "./useHistoryPreview"
import { useQuota } from "./useQuota"
import { isActive, type TaskResponse } from "./utils"
import { archiveCurrentAndSet, type ImageHistory } from "@/lib/history"
import { postForm, postJSON } from "@/lib/http"

function generateFormData(
  prompt: string,
  size: string,
  fingerprint: string,
  referenceFile: File
) {
  const form = new FormData()
  form.set("prompt", prompt)
  form.set("size", size)
  form.set("fingerprint", fingerprint)
  form.set("image", referenceFile, referenceFile.name)
  return form
}

export function useImageGeneratorController() {
  const submittingRef = useRef(false)
  const [language, setLanguage] = useState<Language>(initialLanguage)
  const [loading, setLoading] = useState(false)
  const [submittingPreview, setSubmittingPreview] = useState<{
    prompt: string
    size: string
  } | null>(null)
  const t = copy[language]
  const draft = useCreationDraft(t)
  const historyPreview = useHistoryPreview(t)
  const {
    fingerprint,
    quotaStatus,
    quotaLoading,
    quotaError,
    signingIn,
    checkIn,
    retryQuota,
    applyRemainingCredits,
  } = useQuota(t)

  const currentActive = Boolean(
    historyPreview.currentTask && isActive(historyPreview.currentTask)
  )
  const editingDisabled = loading || draft.enhancing
  const reuseDisabled = loading || draft.enhancing
  const enhanceDisabled = loading || draft.enhancing || !draft.prompt.trim()
  const primaryDisabled =
    loading ||
    draft.enhancing ||
    signingIn ||
    currentActive ||
    Boolean(quotaError) ||
    !quotaStatus ||
    quotaStatus.balance <= 0 ||
    !draft.prompt.trim()

  useEffect(() => {
    persistLanguage(language)
  }, [language])

  async function generate() {
    if (submittingRef.current) {
      return
    }
    if (currentActive) {
      toast.info(t.toast.currentBusy)
      return
    }
    const submittedPrompt = draft.prompt.trim()
    if (!submittedPrompt) {
      toast.error(t.toast.promptRequired)
      return
    }
    if (!fingerprint) {
      toast.error(t.quota.fingerprintFailed)
      return
    }
    if (quotaError) {
      toast.error(quotaError)
      return
    }
    if (!quotaStatus || quotaStatus.balance <= 0) {
      toast.info(
        quotaStatus
          ? t.quota.noCredits(quotaStatus.dailyGrant)
          : t.quota.noCreditsFallback
      )
      return
    }

    submittingRef.current = true
    draft.clearEnhancementUndo()
    setSubmittingPreview({ prompt: submittedPrompt, size: draft.size })
    setLoading(true)
    try {
      const data = draft.referenceFile
        ? await postForm<TaskResponse>(
            "/api/images/generate",
            generateFormData(
              submittedPrompt,
              draft.size,
              fingerprint,
              draft.referenceFile
            )
          )
        : await postJSON<TaskResponse>("/api/images/generate", {
            prompt: submittedPrompt,
            size: draft.size,
            fingerprint,
          })
      if (typeof data.remainingCredits === "number") {
        applyRemainingCredits(data.remainingCredits)
      }
      const item: ImageHistory = {
        ...data,
        prompt: submittedPrompt,
        size: draft.size,
        status: data.status ?? "queued",
        referenceName: draft.referenceName || undefined,
      }
      archiveCurrentAndSet(item)
      toast.success(t.toast.submitted)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err))
    } finally {
      submittingRef.current = false
      setSubmittingPreview(null)
      setLoading(false)
    }
  }

  async function runPrimaryAction() {
    if (loading || draft.enhancing || signingIn) {
      return
    }
    await generate()
  }

  return {
    language,
    t,
    setLanguage,
    currentTask: historyPreview.currentTask,
    history: historyPreview.history,
    prompt: draft.prompt,
    enhanceDirection: draft.enhanceDirection,
    enhanceOriginalPrompt: draft.enhanceOriginalPrompt,
    size: draft.size,
    referenceImage: draft.referenceImage,
    referenceName: draft.referenceName,
    enhancing: draft.enhancing,
    submittingPreview,
    editingDisabled,
    reuseDisabled,
    enhanceDisabled,
    primaryDisabled,
    quotaStatus,
    quotaLoading,
    quotaError,
    signingIn,
    promptInputRef: draft.promptInputRef,
    referenceInputRef: draft.referenceInputRef,
    previewIndex: historyPreview.previewIndex,
    previewItems: historyPreview.previewItems,
    canOpenPreview: historyPreview.canOpenPreview,
    setEnhanceDirection: draft.setEnhanceDirection,
    setSize: draft.setSize,
    selectReference: draft.selectReference,
    resetReference: draft.resetReference,
    changePrompt: draft.changePrompt,
    enhancePrompt: draft.enhancePrompt,
    discardEnhancement: draft.discardEnhancement,
    runPrimaryAction,
    checkIn,
    retryQuota,
    openPreview: historyPreview.openPreview,
    setPreviewId: historyPreview.setPreviewId,
    setImageUnavailable: historyPreview.setImageUnavailable,
    reusePrompt: draft.reusePrompt,
    removeHistory: historyPreview.removeHistory,
    clearHistory: historyPreview.clearHistory,
  }
}
