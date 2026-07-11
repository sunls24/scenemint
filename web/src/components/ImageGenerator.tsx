import { lazy, Suspense } from "react"

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
import { cn } from "@/lib/utils"

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

  return (
    <main className="scene-page min-h-dvh w-full text-foreground">
      <Toaster richColors position="top-right" />
      <div
        className={cn(
          "fixed inset-0 z-[2147483647] flex items-center justify-center p-4 transition-colors duration-150",
          controller.turnstile.interactive
            ? "bg-background/80 backdrop-blur-sm"
            : "pointer-events-none bg-transparent"
        )}
        aria-hidden={!controller.turnstile.interactive}
      >
        <div
          ref={controller.turnstile.containerRef}
          className={cn(
            "min-h-[65px] min-w-[300px] max-w-[calc(100vw-2rem)] transition duration-150",
            controller.turnstile.interactive
              ? "scale-100 opacity-100"
              : "scale-95 opacity-0"
          )}
        />
      </div>

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
