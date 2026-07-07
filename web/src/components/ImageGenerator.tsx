import { useStore } from "@nanostores/react"
import {
  lazy,
  Suspense,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react"
import { toast } from "sonner"

import { Toaster } from "@/components/ui/sonner"
import {
  BrandHeader,
  type RelatedLinks,
} from "@/components/image-generator/BrandHeader"
import {
  copy,
  type EnhanceDirection,
  maxReferenceSize,
  type Language,
} from "@/components/image-generator/copy"
import { CurrentImagePanel } from "@/components/image-generator/CurrentImagePanel"
import { GeneratorForm } from "@/components/image-generator/GeneratorForm"
import { HistoryPanel } from "@/components/image-generator/HistoryPanel"
import { QuotaPanel } from "@/components/image-generator/QuotaPanel"
import {
  canPreview,
  isActive,
  type PreviewImage,
  type TaskResponse,
} from "@/components/image-generator/utils"
import {
  initialEnhanceDirection,
  initialLanguage,
  initialSize,
  persistEnhanceDirection,
  persistLanguage,
  persistSize,
} from "@/components/image-generator/preferences"
import { useQuota } from "@/components/image-generator/useQuota"
import { useTaskPolling } from "@/components/image-generator/useTaskPolling"
import { useTurnstile } from "@/components/image-generator/useTurnstile"
import {
  $currentTask,
  $history,
  archiveCurrentAndSet,
  clearHistory,
  removeHistory,
  type ImageHistory,
} from "@/lib/history"
import { postForm, postJSON, postStream } from "@/lib/http"
import { readChatCompletionStream } from "@/lib/openaiStream"
import { cn } from "@/lib/utils"

const ImagePreviewLightbox = lazy(
  () => import("@/components/image-generator/ImagePreviewLightbox")
)

type PromptRestore = {
  source: string
  result: string
}

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

type ImageGeneratorProps = {
  appVersion: string
  githubUrl: string
  relatedLinks: RelatedLinks
}

export function ImageGenerator({
  appVersion,
  githubUrl,
  relatedLinks,
}: ImageGeneratorProps) {
  const currentTask = useStore($currentTask)
  const history = useStore($history)
  const submittingRef = useRef(false)
  const enhanceAbortRef = useRef<AbortController | null>(null)
  const promptInputRef = useRef<HTMLTextAreaElement>(null)
  const referenceInputRef = useRef<HTMLInputElement>(null)
  const referencePreviewURLRef = useRef("")
  const [language, setLanguage] = useState<Language>(initialLanguage)
  const [prompt, setPrompt] = useState("")
  const [enhanceDirection, setEnhanceDirection] =
    useState<EnhanceDirection>(initialEnhanceDirection)
  const [size, setSize] = useState(initialSize)
  const [referenceImage, setReferenceImage] = useState("")
  const [referenceFile, setReferenceFile] = useState<File | null>(null)
  const [referenceName, setReferenceName] = useState("")
  const [loading, setLoading] = useState(false)
  const [enhancing, setEnhancing] = useState(false)
  const [promptRestore, setPromptRestore] = useState<PromptRestore | null>(null)
  const [submittingPreview, setSubmittingPreview] = useState<{
    prompt: string
    size: string
  } | null>(null)
  const [previewId, setPreviewId] = useState("")
  const [unavailableImageUrls, setUnavailableImageUrls] = useState<Set<string>>(
    () => new Set()
  )
  const t = copy[language]
  const turnstile = useTurnstile()
  const {
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
  } = useQuota(t, {
    getTurnstileToken: turnstile.getToken,
  })
  const currentActive = Boolean(currentTask && isActive(currentTask))
  const controlsDisabled = loading || currentActive || enhancing
  const turnstilePending = turnstile.pending
  const enhanceDisabled = controlsDisabled || turnstilePending || !prompt.trim()
  const generateDisabled =
    controlsDisabled ||
    turnstilePending ||
    Boolean(quotaError) ||
    !quotaReady ||
    !hasCredits
  const generateLabel = quotaError
    ? t.creditsUnavailable
    : turnstilePending
      ? t.verifyingHuman
    : !quotaReady
      ? t.preparingCredits
      : !hasCredits
        ? t.insufficientCredits
        : t.generate
  const restorePromptAvailable = Boolean(
    promptRestore && prompt === promptRestore.result
  )

  useTaskPolling(currentTask, history)

  function canOpenPreview(
    item: ImageHistory | null | undefined
  ): item is PreviewImage {
    return canPreview(item) && !unavailableImageUrls.has(item.image)
  }

  const previewItems = useMemo<PreviewImage[]>(() => {
    const seen = new Set<string>()
    const items: PreviewImage[] = []
    function push(item: ImageHistory | null | undefined) {
      if (!canOpenPreview(item) || seen.has(item.id)) {
        return
      }
      seen.add(item.id)
      items.push(item)
    }
    push(currentTask)
    for (const item of history) {
      push(item)
    }
    return items
  }, [currentTask, history, unavailableImageUrls])

  const previewIndex = previewId
    ? previewItems.findIndex((item) => item.id === previewId)
    : -1

  useEffect(() => {
    persistLanguage(language)
  }, [language])

  useEffect(() => {
    persistSize(size)
  }, [size])

  useEffect(() => {
    persistEnhanceDirection(enhanceDirection)
  }, [enhanceDirection])

  useEffect(() => {
    if (previewId && previewIndex < 0) {
      setPreviewId("")
    }
  }, [previewId, previewIndex])

  useEffect(() => {
    return () => {
      enhanceAbortRef.current?.abort()
      revokeReferencePreview()
    }
  }, [])

  function selectReference(file?: File) {
    if (!file) {
      resetReference()
      return
    }
    if (!file.type.startsWith("image/")) {
      resetReference()
      toast.error(t.toast.selectImage)
      return
    }
    if (file.size > maxReferenceSize) {
      resetReference()
      toast.error(t.toast.referenceTooLarge)
      return
    }
    const previewURL = URL.createObjectURL(file)
    revokeReferencePreview()
    referencePreviewURLRef.current = previewURL
    setReferenceImage(previewURL)
    setReferenceFile(file)
    setReferenceName(file.name)
  }

  async function generate() {
    if (submittingRef.current) {
      return
    }
    if (enhancing) {
      return
    }
    if (currentActive) {
      toast.info(t.toast.currentBusy)
      return
    }
    const submittedPrompt = prompt.trim()
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
    setSubmittingPreview({ prompt: submittedPrompt, size })
    setLoading(true)
    try {
      const turnstileToken = await turnstile.getToken()
      const data = referenceFile
        ? await postForm<TaskResponse>(
            "/api/images/generate",
            generateFormData(submittedPrompt, size, fingerprint, referenceFile),
            { turnstileToken }
          )
        : await postJSON<TaskResponse>("/api/images/generate", {
            prompt: submittedPrompt,
            size,
            fingerprint,
          }, {
            turnstileToken,
          })
      const remainingCredits = data.remainingCredits
      if (typeof remainingCredits === "number") {
        applyRemainingCredits(remainingCredits)
      }
      const item: ImageHistory = {
        ...data,
        prompt: submittedPrompt,
        size,
        status: data.status ?? "queued",
        referenceName: referenceName || undefined,
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

  async function enhancePrompt() {
    if (enhancing) {
      return
    }
    if (currentActive) {
      toast.info(t.toast.currentBusy)
      return
    }
    const originalPrompt = prompt
    const sourcePrompt = originalPrompt.trim()
    if (!sourcePrompt) {
      toast.error(t.toast.enhanceRequired)
      return
    }

    enhanceAbortRef.current?.abort()
    const controller = new AbortController()
    enhanceAbortRef.current = controller
    setPromptRestore(null)
    setEnhancing(true)

    let enhancedPrompt = ""
    let hasVisibleOutput = false
    try {
      const turnstileToken = await turnstile.getToken()
      const stream = await postStream(
        "/api/prompts/enhance",
        {
          prompt: sourcePrompt,
          direction: enhanceDirection,
        },
        {
          signal: controller.signal,
          turnstileToken,
        }
      )
      await readChatCompletionStream(stream, (delta) => {
        enhancedPrompt += delta
        if (!hasVisibleOutput && !enhancedPrompt.trim()) {
          return
        }
        hasVisibleOutput = true
        setPrompt(enhancedPrompt)
      })
      if (!enhancedPrompt.trim()) {
        setPrompt(originalPrompt)
        toast.error(t.toast.enhanceEmpty)
        return
      }
      setPromptRestore({ source: originalPrompt, result: enhancedPrompt })
    } catch (err) {
      if (controller.signal.aborted) {
        return
      }
      if (!hasVisibleOutput) {
        setPrompt(originalPrompt)
      } else {
        setPrompt(enhancedPrompt)
        setPromptRestore({ source: originalPrompt, result: enhancedPrompt })
      }
      toast.error(
        hasVisibleOutput
          ? t.toast.enhancePartial
          : err instanceof Error
            ? err.message
            : t.toast.enhanceFailed
      )
    } finally {
      if (enhanceAbortRef.current === controller) {
        enhanceAbortRef.current = null
        setEnhancing(false)
      }
    }
  }

  function changePrompt(value: string) {
    setPrompt(value)
    setPromptRestore((restore) =>
      restore && value === restore.result ? restore : null
    )
  }

  function restorePrompt() {
    if (!promptRestore || prompt !== promptRestore.result) {
      return
    }
    setPrompt(promptRestore.source)
    setPromptRestore(null)
  }

  function reusePrompt(value: string) {
    setPrompt(value)
    setPromptRestore(null)
    toast.success(t.toast.promptReused)

    if (controlsDisabled) {
      return
    }
    window.requestAnimationFrame(() => {
      const input = promptInputRef.current
      if (!input || input.disabled) {
        return
      }
      input.focus()
      input.setSelectionRange(input.value.length, input.value.length)
    })
  }

  function resetReference() {
    revokeReferencePreview()
    setReferenceImage("")
    setReferenceFile(null)
    setReferenceName("")
    if (referenceInputRef.current) {
      referenceInputRef.current.value = ""
    }
  }

  function revokeReferencePreview() {
    if (!referencePreviewURLRef.current) {
      return
    }
    URL.revokeObjectURL(referencePreviewURLRef.current)
    referencePreviewURLRef.current = ""
  }

  function openPreview(item: ImageHistory | null) {
    if (!canOpenPreview(item)) {
      return
    }
    setPreviewId(item.id)
  }

  function setImageUnavailable(url: string, unavailable: boolean) {
    setUnavailableImageUrls((current) => {
      if (current.has(url) === unavailable) {
        return current
      }
      const next = new Set(current)
      if (unavailable) {
        next.add(url)
      } else {
        next.delete(url)
      }
      return next
    })
  }

  return (
    <main className="scene-page min-h-dvh w-full px-3 py-4 text-foreground md:px-5 md:py-6 xl:min-h-0 xl:flex-1 xl:overflow-hidden xl:pb-4">
      <Toaster richColors position="top-right" />
      <div
        className={cn(
          "fixed inset-0 z-[2147483647] flex items-center justify-center p-4 transition-colors duration-150",
          turnstile.interactive
            ? "bg-background/70 backdrop-blur-sm"
            : "pointer-events-none bg-transparent"
        )}
        aria-hidden={!turnstile.interactive}
      >
        <div
          ref={turnstile.containerRef}
          className={cn(
            "min-h-[65px] min-w-[300px] max-w-[calc(100vw-2rem)] transition duration-150",
            turnstile.interactive
              ? "scale-100 opacity-100"
              : "scale-95 opacity-0"
          )}
        />
      </div>

      <div className="mx-auto flex w-full max-w-7xl flex-col gap-4 xl:h-full xl:min-h-0">
        <BrandHeader
          appVersion={appVersion}
          githubUrl={githubUrl}
          relatedLinks={relatedLinks}
          language={language}
          t={t}
          onLanguageChange={setLanguage}
        />

        <section className="grid w-full gap-4 lg:grid-cols-[320px_minmax(0,1fr)] xl:min-h-0 xl:flex-1 xl:grid-cols-[320px_minmax(0,1fr)_340px] xl:grid-rows-[minmax(0,1fr)] xl:items-start xl:overflow-hidden">
          <GeneratorForm
            t={t}
            prompt={prompt}
            enhanceDirection={enhanceDirection}
            size={size}
            referenceImage={referenceImage}
            referenceName={referenceName}
            controlsDisabled={controlsDisabled}
            enhanceDisabled={enhanceDisabled}
            generateDisabled={generateDisabled}
            generateLabel={generateLabel}
            loading={loading}
            enhancing={enhancing}
            currentActive={currentActive}
            promptInputRef={promptInputRef}
            referenceInputRef={referenceInputRef}
            quotaPanel={
              <QuotaPanel
                t={t}
                status={quotaStatus}
                fingerprint={fingerprint}
                loading={quotaLoading}
                signingIn={signingIn}
                actionDisabled={turnstilePending}
                error={quotaError}
                onCheckIn={() => void checkIn()}
                onRetry={() => void retryQuota()}
              />
            }
            restorePromptAvailable={restorePromptAvailable}
            onPromptChange={changePrompt}
            onEnhanceDirectionChange={setEnhanceDirection}
            onEnhance={() => void enhancePrompt()}
            onRestorePrompt={restorePrompt}
            onSizeChange={setSize}
            onReferenceChange={selectReference}
            onResetReference={resetReference}
            onGenerate={() => void generate()}
          />

          <CurrentImagePanel
            t={t}
            currentTask={currentTask}
            submittingPreview={submittingPreview}
            reuseDisabled={controlsDisabled}
            canOpenPreview={canOpenPreview}
            onImageUnavailable={setImageUnavailable}
            onOpenPreview={openPreview}
            onReusePrompt={reusePrompt}
          />

          <HistoryPanel
            t={t}
            history={history}
            reuseDisabled={controlsDisabled}
            canOpenPreview={canOpenPreview}
            onImageUnavailable={setImageUnavailable}
            onClearHistory={clearHistory}
            onOpenPreview={openPreview}
            onReusePrompt={reusePrompt}
            onRemoveHistory={removeHistory}
          />
        </section>
      </div>

      {previewIndex >= 0 && (
        <Suspense fallback={null}>
          <ImagePreviewLightbox
            index={previewIndex}
            previewItems={previewItems}
            t={t}
            onClose={() => setPreviewId("")}
            onView={setPreviewId}
          />
        </Suspense>
      )}
    </main>
  )
}
