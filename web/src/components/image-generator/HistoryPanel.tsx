import { ImageIcon, ListRestartIcon, Trash2Icon, XCircleIcon } from "lucide-react"

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
import { Skeleton } from "@/components/ui/skeleton"
import type { ImageHistory } from "@/lib/history"

import type { ImageGeneratorCopy } from "./copy"
import { FallbackImage } from "./FallbackImage"
import { sizeLabel, statusLabel, statusOf } from "./utils"

type HistoryPanelProps = {
  t: ImageGeneratorCopy
  history: ImageHistory[]
  reuseDisabled: boolean
  canOpenPreview: (item: ImageHistory | null | undefined) => boolean
  onImageUnavailable: (url: string, unavailable: boolean) => void
  onClearHistory: () => void
  onOpenPreview: (item: ImageHistory | null) => void
  onReusePrompt: (prompt: string) => void
  onRemoveHistory: (id: string) => void
}

export function HistoryPanel({
  t,
  history,
  reuseDisabled,
  canOpenPreview,
  onImageUnavailable,
  onClearHistory,
  onOpenPreview,
  onReusePrompt,
  onRemoveHistory,
}: HistoryPanelProps) {
  return (
    <Card
      className="scene-history-panel scene-card min-h-[260px] gap-0 self-start py-0 lg:col-start-3 lg:row-start-1 lg:max-h-full lg:min-h-0"
      role="region"
      aria-labelledby="history-title"
    >
      <CardHeader className="scene-panel-heading shrink-0 border-b px-3.5 py-2.5">
        <div className="flex items-start justify-between gap-3">
          <div>
            <h2 id="history-title" className="text-lg font-semibold">
              {t.historyTitle}
            </h2>
            <CardDescription className="mt-0.5 text-xs">
              {t.historyCount(history.length)} · {t.historyRetention}
            </CardDescription>
          </div>
          {history.length > 0 && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={onClearHistory}
              title={t.clearHistoryLabel}
              aria-label={t.clearHistoryLabel}
              className="text-muted-foreground hover:text-destructive"
            >
              <Trash2Icon data-icon="inline-start" />
              {t.clearHistory}
            </Button>
          )}
        </div>
      </CardHeader>

      <CardContent className="flex min-h-0 flex-1 p-2.5">
        <div className="scene-history-list flex min-h-0 w-full flex-1 flex-col gap-2 overflow-y-auto">
          {history.length === 0 ? (
            <Empty className="scene-history-empty min-h-36 border-0 px-5">
              <EmptyHeader>
                <EmptyMedia variant="icon" className="scene-history-empty-mark size-10 rounded-full">
                  <ImageIcon />
                </EmptyMedia>
                <EmptyTitle>{t.historyEmpty}</EmptyTitle>
                <EmptyDescription>{t.historyEmptyDescription}</EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            history.map((item) => {
              const itemStatus = statusOf(item)
              const itemActive = itemStatus === "queued" || itemStatus === "running"
              const itemImage = item.image
              const itemPreviewable = canOpenPreview(item)
              return (
                <article key={item.id} className="scene-history-row flex items-center gap-3 rounded-lg border p-2.5">
                  <button
                    type="button"
                    onClick={() => onOpenPreview(item)}
                    disabled={!itemPreviewable}
                    className="flex aspect-square size-16 shrink-0 items-center justify-center overflow-hidden rounded-md border-0 bg-muted p-0 outline-none focus-visible:ring-3 focus-visible:ring-ring/50 disabled:cursor-default"
                    aria-label={`${t.currentPreview}: ${item.prompt}`}
                  >
                    {itemActive ? (
                      <Skeleton className="size-full rounded-none" />
                    ) : itemStatus === "failed" ? (
                      <XCircleIcon className="text-muted-foreground" />
                    ) : itemImage ? (
                      <FallbackImage
                        src={itemImage}
                        alt=""
                        loading="lazy"
                        fetchPriority="low"
                        className="object-cover"
                        wrapperClassName="size-full rounded-md"
                        onLoad={() => onImageUnavailable(itemImage, false)}
                        onError={() => onImageUnavailable(itemImage, true)}
                      />
                    ) : (
                      <ImageIcon className="text-muted-foreground" />
                    )}
                  </button>

                  <div className="flex min-w-0 flex-1 flex-col justify-between gap-2">
                    <p className="line-clamp-2 min-w-0 text-sm leading-5 font-medium">
                      {item.prompt}
                    </p>
                    <div className="flex flex-nowrap items-center gap-1 whitespace-nowrap text-xs text-muted-foreground">
                      <Badge variant="outline" className="px-1.5">{t.mode[item.mode]}</Badge>
                      <Badge variant="secondary" className="px-1.5">
                        {statusLabel(itemStatus, t)}
                      </Badge>
                      <span className="shrink-0">{sizeLabel(item.size, t)}</span>
                    </div>
                  </div>
                  <div className="flex shrink-0 flex-col items-center gap-1">
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => onReusePrompt(item.prompt)}
                      disabled={reuseDisabled}
                      title={t.reusePromptLabel}
                      aria-label={`${t.reusePromptLabel}: ${item.prompt}`}
                    >
                      <ListRestartIcon />
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => onRemoveHistory(item.id)}
                      title={t.removeHistoryLabel}
                      aria-label={`${t.removeHistoryLabel}: ${item.prompt}`}
                      className="text-muted-foreground hover:text-destructive"
                    >
                      <Trash2Icon />
                    </Button>
                  </div>
                </article>
              )
            })
          )}
        </div>
      </CardContent>
    </Card>
  )
}
