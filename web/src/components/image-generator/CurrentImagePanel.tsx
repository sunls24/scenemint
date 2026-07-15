import type { CSSProperties, Ref } from "react"
import {
  DownloadIcon,
  ImageIcon,
  ListRestartIcon,
  Loader2Icon,
  Maximize2Icon,
  ShieldAlertIcon,
} from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader } from "@/components/ui/card"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import type { ImageHistory } from "@/lib/history"
import { cn } from "@/lib/utils"

import type { ImageGeneratorCopy } from "./copy"
import { FallbackImage } from "./FallbackImage"
import { dimensionsOf, downloadImage, sizeLabel, statusLabel, statusOf } from "./utils"

type CurrentImagePanelProps = {
  panelRef?: Ref<HTMLDivElement>
  t: ImageGeneratorCopy
  selectedSize: string
  currentTask: ImageHistory | null
  submittingPreview: { prompt: string; size: string } | null
  reuseDisabled: boolean
  canOpenPreview: (item: ImageHistory | null | undefined) => boolean
  onImageUnavailable: (url: string, unavailable: boolean) => void
  onOpenPreview: (item: ImageHistory | null) => void
  onReusePrompt: (prompt: string) => void
}

export function CurrentImagePanel({
  panelRef,
  t,
  selectedSize,
  currentTask,
  submittingPreview,
  reuseDisabled,
  canOpenPreview,
  onImageUnavailable,
  onOpenPreview,
  onReusePrompt,
}: CurrentImagePanelProps) {
  const currentStatus = currentTask ? statusOf(currentTask) : undefined
  const currentImage = currentTask?.image
  const currentPreviewable = canOpenPreview(currentTask)
  const pendingLabel = submittingPreview ? t.submitting : t.generating
  const statusBadgeLabel = submittingPreview
    ? t.submitting
    : currentStatus
      ? statusLabel(currentStatus, t)
      : ""
  const statusBadgeVariant = submittingPreview
    ? "outline"
    : currentStatus === "failed"
      ? "destructive"
      : currentStatus === "completed"
        ? "secondary"
        : "outline"
  const currentDescription =
    submittingPreview?.prompt ||
    currentTask?.prompt ||
    (currentStatus && currentStatus !== "failed" ? statusLabel(currentStatus, t) : "")
  const reusablePrompt = submittingPreview?.prompt || currentTask?.prompt || ""
  const showGenerating =
    Boolean(submittingPreview) || currentStatus === "queued" || currentStatus === "running"
  const mediaCanvas = Boolean(
    currentTask && currentImage && currentStatus === "completed"
  )
  const emptyCanvas = !currentTask && !submittingPreview
  const placeholderCanvas =
    emptyCanvas || showGenerating || currentStatus === "failed"
  const canvasModeClass = mediaCanvas
    ? "is-media"
    : placeholderCanvas
      ? "is-placeholder"
      : ""
  const previewSize = submittingPreview?.size ?? currentTask?.size ?? selectedSize
  const previewDimensions = dimensionsOf(previewSize)
  const previewFrameStyle: CSSProperties & { "--scene-preview-ratio": string } = {
    "--scene-preview-ratio": String(previewDimensions.width / previewDimensions.height),
  }
  const handleDownload =
    currentTask?.image && currentPreviewable
      ? () =>
          downloadImage(
            currentTask.image,
            `scenemint-${currentTask.id ?? Date.now()}.png`
          )
      : undefined

  return (
    <Card
      ref={panelRef}
      className={cn(
        "scene-canvas-panel scene-card min-h-0 self-start gap-0 py-0",
        canvasModeClass
      )}
      role="region"
      aria-labelledby="current-image-title"
    >
      <CardHeader className="scene-panel-heading shrink-0 border-b px-3.5 py-2.5">
        <div className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-start gap-x-3 gap-y-0.5">
          <div className="min-w-0">
            <h2 id="current-image-title" className="text-lg font-semibold">
              {t.currentTitle}
            </h2>
          </div>
          <div className="flex shrink-0 items-center gap-1.5">
            {statusBadgeLabel && <Badge variant={statusBadgeVariant}>{statusBadgeLabel}</Badge>}
            {reusablePrompt && (
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                onClick={() => onReusePrompt(reusablePrompt)}
                disabled={reuseDisabled}
                title={t.reusePromptLabel}
                aria-label={`${t.reusePromptLabel}: ${reusablePrompt}`}
              >
                <ListRestartIcon />
              </Button>
            )}
            {handleDownload && (
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                onClick={handleDownload}
                title={t.lightbox.download}
                aria-label={t.lightbox.download}
              >
                <DownloadIcon />
              </Button>
            )}
          </div>
          {currentDescription && (
            <CardDescription className="col-span-2 line-clamp-1 text-xs">
              {currentDescription}
            </CardDescription>
          )}
        </div>
      </CardHeader>

      <CardContent className="scene-canvas-scroll flex min-h-0 flex-1 overflow-y-auto p-0">
        <div className="scene-current-preview-shell flex min-h-0 w-full flex-1 items-center justify-center">
          <div
            className="scene-current-frame relative flex w-full items-center justify-center overflow-hidden"
            data-status={currentStatus}
            style={previewFrameStyle}
          >
            {placeholderCanvas && (
              <span className="scene-canvas-ratio absolute top-3 right-3">
                {sizeLabel(previewSize, t)} · {previewSize}
              </span>
            )}
            {showGenerating ? (
              <div
                className="scene-current-media-pending absolute inset-0 flex flex-col items-center justify-center gap-2"
                role="status"
                aria-live="polite"
              >
                <Loader2Icon className="size-6 animate-spin motion-reduce:animate-none" />
                <span className="text-sm font-medium">{pendingLabel}</span>
              </div>
            ) : currentTask && currentStatus === "failed" ? (
              <Empty className="scene-current-empty-state max-w-md border-0 px-6">
                <EmptyHeader>
                  <EmptyMedia variant="icon" className="scene-canvas-empty-icon size-12 rounded-full">
                    <ShieldAlertIcon />
                  </EmptyMedia>
                  <EmptyTitle className="text-base">{t.currentFailed}</EmptyTitle>
                  <EmptyDescription className="max-w-md">
                    {currentTask.error || t.currentFailedDescription}
                  </EmptyDescription>
                </EmptyHeader>
              </Empty>
            ) : currentTask && currentImage ? (
              <button
                type="button"
                onClick={() => onOpenPreview(currentTask)}
                disabled={!currentPreviewable}
                className="scene-current-media group/image relative flex items-center justify-center overflow-hidden outline-none focus-visible:ring-3 focus-visible:ring-ring/50 disabled:cursor-default"
                aria-label={t.currentPreview}
              >
                <FallbackImage
                  src={currentImage}
                  alt={currentTask.prompt}
                  decoding="async"
                  className="object-contain"
                  wrapperClassName="size-full"
                  onLoad={() => onImageUnavailable(currentImage, false)}
                  onError={() => onImageUnavailable(currentImage, true)}
                />
                {currentPreviewable && (
                  <span className="scene-preview-open-indicator pointer-events-none absolute right-3 bottom-3 inline-flex items-center justify-center opacity-0 transition-opacity duration-200 group-hover/image:opacity-100 group-focus-visible/image:opacity-100">
                    <Maximize2Icon />
                  </span>
                )}
              </button>
            ) : (
              <Empty className="scene-current-empty-state max-w-md border-0 px-6">
                <EmptyHeader>
                  <EmptyMedia variant="icon" className="scene-canvas-empty-icon size-12 rounded-full">
                    <ImageIcon />
                  </EmptyMedia>
                  <EmptyTitle className="text-base">{t.currentEmpty}</EmptyTitle>
                </EmptyHeader>
              </Empty>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
