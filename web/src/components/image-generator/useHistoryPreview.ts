import { useStore } from "@nanostores/react"
import { useEffect, useMemo, useState } from "react"
import { toast } from "sonner"

import {
  $currentTask,
  $history,
  clearHistory,
  removeHistory,
  restoreHistory,
  type ImageHistory,
} from "@/lib/history"

import type { ImageGeneratorCopy } from "./copy"
import { useTaskPolling } from "./useTaskPolling"
import { canPreview, type PreviewImage } from "./utils"

export function useHistoryPreview(t: ImageGeneratorCopy) {
  const currentTask = useStore($currentTask)
  const history = useStore($history)
  const [previewId, setPreviewId] = useState("")
  const [unavailableImageUrls, setUnavailableImageUrls] = useState<Set<string>>(
    () => new Set()
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
    if (previewId && previewIndex < 0) {
      setPreviewId("")
    }
  }, [previewId, previewIndex])

  function openPreview(item: ImageHistory | null) {
    if (canOpenPreview(item)) {
      setPreviewId(item.id)
    }
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

  function removeHistoryWithUndo(id: string) {
    const item = history.find((entry) => entry.id === id)
    if (!item) {
      return
    }
    removeHistory(id)
    toast(t.toast.historyRemoved, {
      action: {
        label: t.toast.undo,
        onClick: () => restoreHistory([item]),
      },
    })
  }

  function clearHistoryWithUndo() {
    const items = [...history]
    if (!items.length) {
      return
    }
    clearHistory()
    toast(t.toast.historyCleared, {
      action: {
        label: t.toast.undo,
        onClick: () => restoreHistory(items),
      },
    })
  }

  return {
    currentTask,
    history,
    previewIndex,
    previewItems,
    canOpenPreview,
    openPreview,
    setPreviewId,
    setImageUnavailable,
    removeHistory: removeHistoryWithUndo,
    clearHistory: clearHistoryWithUndo,
  }
}
