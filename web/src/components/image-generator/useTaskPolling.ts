import { useEffect, useMemo } from "react"

import {
  removeHistory,
  setCurrentTask,
  updateTask,
  type ImageHistory,
} from "@/lib/history"
import { getJSON } from "@/lib/http"

import {
  isActive,
  mergeTask,
  statusOf,
  type TaskResponse,
} from "./utils"

export function useTaskPolling(
  currentTask: ImageHistory | null,
  history: ImageHistory[]
) {
  const activeItems = useMemo(() => {
    const items: ImageHistory[] = []
    if (currentTask && isActive(currentTask)) {
      items.push(currentTask)
    }
    for (const item of history) {
      if (item.id !== currentTask?.id && isActive(item)) {
        items.push(item)
      }
    }
    return items
  }, [currentTask, history])

  const activeSignature = activeItems
    .map((item) => `${item.id}:${statusOf(item)}:${item.updatedAt ?? ""}`)
    .join("|")

  useEffect(() => {
    if (currentTask) {
      return
    }
    const active = history.find(isActive)
    if (!active) {
      return
    }
    setCurrentTask(active)
    removeHistory(active.id)
  }, [currentTask, history])

  useEffect(() => {
    if (activeItems.length === 0) {
      return
    }

    let disposed = false
    async function poll() {
      await Promise.all(
        activeItems.map(async (item) => {
          try {
            const task = await getJSON<TaskResponse>(
              `/api/images/tasks/${encodeURIComponent(item.id)}`
            )
            if (disposed) {
              return
            }
            updateTask(item.id, mergeTask(item, task))
          } catch {
            // Keep active tasks pending on transient network/server errors.
          }
        })
      )
    }

    void poll()
    const timer = window.setInterval(() => void poll(), 3000)
    return () => {
      disposed = true
      window.clearInterval(timer)
    }
  }, [activeSignature])
}
