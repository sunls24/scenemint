import { lazy, Suspense, useEffect, useRef } from "react"

import { Toaster } from "@/components/ui/sonner"
import {
  BrandHeader,
  type RelatedLinks,
} from "@/components/image-generator/BrandHeader"
import { CurrentImagePanel } from "@/components/image-generator/CurrentImagePanel"
import { GeneratorForm } from "@/components/image-generator/GeneratorForm"
import { HistoryPanel } from "@/components/image-generator/HistoryPanel"
import { QuotaPanel } from "@/components/image-generator/QuotaPanel"
import { useImageGeneratorController } from "@/components/image-generator/useImageGeneratorController"

const ImagePreviewLightbox = lazy(
  () => import("@/components/image-generator/ImagePreviewLightbox")
)

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
  const controller = useImageGeneratorController()
  const { t } = controller
  const currentImagePanelRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (
      !controller.submittingPreview ||
      !window.matchMedia("(max-width: 1023px)").matches
    ) {
      return
    }

    const reducedMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)"
    ).matches
    const frame = window.requestAnimationFrame(() => {
      currentImagePanelRef.current?.scrollIntoView({
        behavior: reducedMotion ? "auto" : "smooth",
        block: "start",
      })
    })

    return () => window.cancelAnimationFrame(frame)
  }, [controller.submittingPreview])

  return (
    <main className="scene-page min-h-dvh w-full text-foreground">
      <Toaster richColors position="top-right" />
      <div className="scene-shell mx-auto flex min-h-dvh w-full max-w-[1540px] flex-col px-3 py-3 sm:px-5 sm:py-4 lg:h-dvh lg:min-h-0 lg:overflow-hidden xl:px-6">
        <BrandHeader
          appVersion={appVersion}
          githubUrl={githubUrl}
          relatedLinks={relatedLinks}
          language={controller.language}
          t={t}
          onLanguageChange={controller.setLanguage}
        />

        <section className="scene-workspace grid min-h-0 flex-1 gap-3 pt-3 lg:grid-cols-[320px_minmax(0,1fr)_300px] lg:grid-rows-[minmax(0,1fr)] lg:items-stretch lg:overflow-hidden">
          <GeneratorForm
            t={t}
            prompt={controller.prompt}
            enhanceDirection={controller.enhanceDirection}
            enhanceOriginalPrompt={controller.enhanceOriginalPrompt}
            size={controller.size}
            referenceImage={controller.referenceImage}
            referenceName={controller.referenceName}
            editingDisabled={controller.editingDisabled}
            enhanceDisabled={controller.enhanceDisabled}
            enhancing={controller.enhancing}
            primaryDisabled={controller.primaryDisabled}
            promptInputRef={controller.promptInputRef}
            referenceInputRef={controller.referenceInputRef}
            quotaPanel={
              <QuotaPanel
                t={t}
                status={controller.quotaStatus}
                loading={controller.quotaLoading}
                signingIn={controller.signingIn}
                error={controller.quotaError}
                onCheckIn={() => void controller.checkIn()}
                onRetry={() => void controller.retryQuota()}
              />
            }
            onPromptChange={controller.changePrompt}
            onEnhanceDirectionChange={controller.setEnhanceDirection}
            onEnhance={() => void controller.enhancePrompt()}
            onDiscardEnhancement={controller.discardEnhancement}
            onSizeChange={controller.setSize}
            onReferenceChange={controller.selectReference}
            onResetReference={controller.resetReference}
            onPrimaryAction={() => void controller.runPrimaryAction()}
          />

          <CurrentImagePanel
            panelRef={currentImagePanelRef}
            t={t}
            selectedSize={controller.size}
            currentTask={controller.currentTask}
            submittingPreview={controller.submittingPreview}
            reuseDisabled={controller.reuseDisabled}
            canOpenPreview={controller.canOpenPreview}
            onImageUnavailable={controller.setImageUnavailable}
            onOpenPreview={controller.openPreview}
            onReusePrompt={controller.reusePrompt}
          />

          <HistoryPanel
            t={t}
            history={controller.history}
            reuseDisabled={controller.reuseDisabled}
            canOpenPreview={controller.canOpenPreview}
            onImageUnavailable={controller.setImageUnavailable}
            onClearHistory={controller.clearHistory}
            onOpenPreview={controller.openPreview}
            onReusePrompt={controller.reusePrompt}
            onRemoveHistory={controller.removeHistory}
          />
        </section>
      </div>

      {controller.previewIndex >= 0 && (
        <Suspense fallback={null}>
          <ImagePreviewLightbox
            index={controller.previewIndex}
            previewItems={controller.previewItems}
            t={t}
            onClose={() => controller.setPreviewId("")}
            onView={controller.setPreviewId}
          />
        </Suspense>
      )}
    </main>
  )
}
