import {
  ImageIcon,
  ListRestartIcon,
  Trash2Icon,
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
import { Skeleton } from "@/components/ui/skeleton"
import type { ImageHistory } from "@/lib/history"

import type { ImageGeneratorCopy } from "./copy"
import { FallbackImage } from "./FallbackImage"
import { sizeLabel, statusLabel, statusOf } from "./utils"

type HistoryPanelProps = {
  t: ImageGeneratorCopy
  history: ImageHistory[]
  reuseDisabled: boolean
  onClearHistory: () => void
  onOpenPreview: (item: ImageHistory | null) => void
  onReusePrompt: (prompt: string) => void
  onRemoveHistory: (id: string) => void
}

export function HistoryPanel({
  t,
  history,
  reuseDisabled,
  onClearHistory,
  onOpenPreview,
  onReusePrompt,
  onRemoveHistory,
}: HistoryPanelProps) {
  return (
    <Card className="scene-card lg:col-start-2 lg:col-span-1 xl:col-start-3 xl:row-start-1 xl:max-h-full xl:min-h-0 xl:overflow-hidden">
      <CardHeader>
        <div className="flex items-start justify-between gap-3">
          <div className="flex flex-col gap-1">
            <CardTitle className="text-[1.05rem]">{t.historyTitle}</CardTitle>
            <CardDescription>{t.historyCount(history.length)}</CardDescription>
          </div>
          <Button
            type="button"
            variant="destructive"
            size="sm"
            onClick={onClearHistory}
            disabled={history.length === 0}
            title={t.clearHistoryLabel}
            aria-label={t.clearHistoryLabel}
          >
            <Trash2Icon data-icon="inline-start" />
            {t.clearHistory}
          </Button>
        </div>
      </CardHeader>
      <CardContent className="flex min-h-0 flex-1 flex-col">
        <div className="scene-history-list flex max-h-[520px] flex-col gap-2 overflow-auto xl:min-h-0 xl:flex-1 xl:max-h-none">
          {history.length === 0 ? (
            <div className="scene-history-row flex items-center gap-3 rounded-lg border p-2">
              <span className="scene-empty-mark flex size-11 shrink-0 items-center justify-center rounded-lg">
                <ImageIcon />
              </span>
              <p className="text-sm text-muted-foreground">{t.historyEmpty}</p>
            </div>
          ) : (
            history.map((item) => {
              const itemStatus = statusOf(item)
              const itemActive =
                itemStatus === "queued" || itemStatus === "running"
              const completed = itemStatus === "completed" && Boolean(item.image)
              return (
                <div
                  key={item.id}
                  className="scene-history-row flex w-full items-center gap-3 rounded-lg border p-2"
                >
                  <button
                    type="button"
                    onClick={() => onOpenPreview(item)}
                    disabled={!completed}
                    className="flex aspect-square size-16 shrink-0 appearance-none items-center justify-center overflow-hidden rounded-md border-0 bg-transparent p-0 outline-none focus-visible:ring-3 focus-visible:ring-ring/50 disabled:cursor-default"
                    aria-label={`${t.currentPreview}: ${item.prompt}`}
                  >
                    {itemActive ? (
                      <Skeleton className="size-full" />
                    ) : itemStatus === "failed" ? (
                      <XCircleIcon className="text-muted-foreground" />
                    ) : item.image ? (
                      <FallbackImage
                        src={item.image}
                        alt=""
                        loading="lazy"
                        fetchPriority="low"
                        className="object-cover"
                        wrapperClassName="size-full rounded-md"
                      />
                    ) : (
                      <ImageIcon className="text-muted-foreground" />
                    )}
                  </button>
                  <span className="flex min-h-16 min-w-0 flex-1 flex-col justify-center gap-2">
                    <span className="flex w-full min-w-0 items-center justify-between gap-1">
                      <span className="min-w-0 flex-1 truncate text-sm font-medium">
                        {item.prompt}
                      </span>
                      <Button
                        type="button"
                        variant="outline"
                        size="icon-sm"
                        onClick={() => onReusePrompt(item.prompt)}
                        disabled={reuseDisabled}
                        title={t.reusePromptLabel}
                        aria-label={`${t.reusePromptLabel}: ${item.prompt}`}
                        className="shrink-0"
                      >
                        <ListRestartIcon data-icon="inline-start" />
                      </Button>
                    </span>
                    <span className="flex min-w-0 items-center justify-between gap-2">
                      <span className="flex min-w-0 items-center gap-1.5 overflow-hidden text-xs text-muted-foreground">
                        <Badge variant="outline">{t.mode[item.mode]}</Badge>
                        <Badge variant="secondary">
                          {statusLabel(itemStatus, t)}
                        </Badge>
                        <span className="shrink-0">
                          {sizeLabel(item.size, t)}
                        </span>
                      </span>
                      <Button
                        type="button"
                        variant="destructive"
                        size="icon-sm"
                        onClick={() => onRemoveHistory(item.id)}
                        title={t.removeHistoryLabel}
                        aria-label={`${t.removeHistoryLabel}: ${item.prompt}`}
                        className="scene-history-remove shrink-0 transition-opacity"
                      >
                        <Trash2Icon
                          data-icon="inline-start"
                          aria-hidden="true"
                        />
                      </Button>
                    </span>
                  </span>
                </div>
              )
            })
          )}
        </div>
      </CardContent>
    </Card>
  )
}
