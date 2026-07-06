import type { CSSProperties } from "react"
import {
  DownloadIcon,
  ImageIcon,
  ListRestartIcon,
  Loader2Icon,
  Maximize2Icon,
  XCircleIcon,
} from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import type { ImageHistory } from "@/lib/history"

import type { ImageGeneratorCopy } from "./copy"
import { FallbackImage } from "./FallbackImage"
import {
  dimensionsOf,
  downloadImage,
  statusLabel,
  statusOf,
} from "./utils"

type CurrentImagePanelProps = {
  t: ImageGeneratorCopy
  currentTask: ImageHistory | null
  submittingPreview: {
    prompt: string
    size: string
  } | null
  reuseDisabled: boolean
  onOpenPreview: (item: ImageHistory | null) => void
  onReusePrompt: (prompt: string) => void
}

export function CurrentImagePanel({
  t,
  currentTask,
  submittingPreview,
  reuseDisabled,
  onOpenPreview,
  onReusePrompt,
}: CurrentImagePanelProps) {
  const currentStatus = currentTask ? statusOf(currentTask) : undefined
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
    (currentStatus && currentStatus !== "failed"
      ? statusLabel(currentStatus, t)
      : "")
  const reusablePrompt = submittingPreview?.prompt || currentTask?.prompt || ""
  const showGenerating =
    Boolean(submittingPreview) ||
    currentStatus === "queued" ||
    currentStatus === "running"
  const previewDimensions = dimensionsOf(
    submittingPreview?.size ?? currentTask?.size ?? "1:1"
  )
  const previewFrameStyle: CSSProperties & {
    "--scene-preview-ratio": string
  } = {
    "--scene-preview-ratio": String(
      previewDimensions.width / previewDimensions.height
    ),
  }

  const handleDownload = currentTask?.image
    ? () =>
        downloadImage(
          currentTask.image,
          `scenemint-${currentTask.id ?? Date.now()}.png`
        )
    : undefined

  return (
    <Card className="scene-card scene-card-clear min-h-0 overflow-hidden xl:max-h-full">
      <CardHeader>
        <div className="flex min-w-0 flex-col gap-1">
          <div className="flex min-w-0 items-start justify-between gap-3">
            <CardTitle className="min-w-0 text-[1.05rem]">
              {t.currentTitle}
            </CardTitle>
            <div className="flex shrink-0 items-center justify-end gap-2">
              {statusBadgeLabel && (
                <Badge
                  variant={statusBadgeVariant}
                  className="w-fit"
                >
                  {statusBadgeLabel}
                </Badge>
              )}
              {reusablePrompt ? (
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => onReusePrompt(reusablePrompt)}
                  disabled={reuseDisabled}
                  title={t.reusePromptLabel}
                  aria-label={`${t.reusePromptLabel}: ${reusablePrompt}`}
                  className="w-fit"
                >
                  <ListRestartIcon data-icon="inline-start" />
                  {t.reusePrompt}
                </Button>
              ) : null}
              {handleDownload && (
                <Button
                  type="button"
                  variant="outline"
                  size="icon-sm"
                  onClick={handleDownload}
                  title={t.lightbox.download}
                  aria-label={t.lightbox.download}
                >
                  <DownloadIcon />
                </Button>
              )}
            </div>
          </div>
          {currentDescription && (
            <CardDescription className="line-clamp-2">
              {currentDescription}
            </CardDescription>
          )}
        </div>
      </CardHeader>
      <CardContent className="flex min-h-0 flex-col gap-4">
        <div className="scene-current-preview-shell flex min-h-0 items-center justify-center">
          <div
            className="scene-current-frame relative flex w-full items-center justify-center overflow-hidden rounded-lg"
            style={previewFrameStyle}
          >
            {showGenerating ? (
              <div
                className="scene-current-media scene-current-media-pending relative flex items-center justify-center overflow-hidden rounded-lg"
                role="status"
                aria-live="polite"
              >
                <div className="scene-generating-core relative inline-flex items-center gap-2.5 rounded-full px-4 py-2.5 text-sm font-medium">
                  <Loader2Icon className="animate-spin motion-reduce:animate-none" />
                  {pendingLabel}
                </div>
              </div>
            ) : currentTask && currentStatus === "failed" ? (
              <div className="scene-current-empty-state flex flex-col items-center justify-center gap-3 px-6 text-center text-sm text-muted-foreground">
                <XCircleIcon />
                <span>{currentTask.error || t.currentFailed}</span>
              </div>
            ) : currentTask?.image ? (
              <button
                type="button"
                onClick={() => onOpenPreview(currentTask)}
                className="scene-current-media group/image relative flex items-center justify-center overflow-hidden rounded-lg outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
                aria-label={t.currentPreview}
              >
                <FallbackImage
                  src={currentTask.image}
                  alt={currentTask.prompt}
                  decoding="async"
                  className="object-contain"
                  wrapperClassName="size-full rounded-lg"
                  errorLabel={t.imageExpired}
                />
                <span className="scene-preview-open-indicator pointer-events-none absolute right-3 bottom-3 inline-flex items-center justify-center opacity-0 transition-opacity duration-200 group-hover/image:opacity-100 group-focus-visible/image:opacity-100">
                  <Maximize2Icon />
                </span>
              </button>
            ) : (
              <div className="scene-current-empty-state flex flex-col items-center justify-center gap-2.5 text-center text-muted-foreground">
                <span className="scene-empty-mark flex size-11 items-center justify-center rounded-lg">
                  <ImageIcon />
                </span>
                <span className="text-sm font-medium text-foreground">
                  {t.currentEmpty}
                </span>
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
