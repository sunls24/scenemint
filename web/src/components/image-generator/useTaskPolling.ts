import { useEffect, useMemo, useRef } from "react"

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
  type TaskResponse,
} from "./utils"

export function useTaskPolling(
  currentTask: ImageHistory | null,
  history: ImageHistory[]
) {
  const latestItemsRef = useRef<{
    currentTask: ImageHistory | null
    history: ImageHistory[]
  }>({ currentTask, history })
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
    .map((item) => item.id)
    .join("|")

  useEffect(() => {
    latestItemsRef.current = { currentTask, history }
  }, [currentTask, history])

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
    if (!activeSignature) {
      return
    }

    let disposed = false
    let timer = 0
    let polling = false
    let controller: AbortController | null = null
    let errorStreak = 0
    const activeIds = activeSignature.split("|")
    const latestItem = (id: string) => {
      const latest = latestItemsRef.current
      if (latest.currentTask?.id === id) {
        return latest.currentTask
      }
      return latest.history.find((item) => item.id === id)
    }

    function schedule(delay: number) {
      if (disposed) {
        return
      }
      if (timer) {
        window.clearTimeout(timer)
      }
      timer = window.setTimeout(() => void poll(), delay)
    }

    async function poll() {
      if (disposed || polling) {
        return
      }
      if (document.hidden) {
        schedule(5_000)
        return
      }

      polling = true
      controller = new AbortController()
      let hadError = false
      await Promise.all(
        activeIds.map(async (id) => {
          try {
            const task = await getJSON<TaskResponse>(
              `/api/images/tasks/${encodeURIComponent(id)}`,
              { signal: controller?.signal }
            )
            if (disposed) {
              return
            }
            const item = latestItem(id)
            if (!item || !isActive(item)) {
              return
            }
            updateTask(id, mergeTask(item, task))
          } catch (err) {
            if (err instanceof DOMException && err.name === "AbortError") {
              return
            }
            hadError = true
          }
        })
      )
      polling = false
      controller = null
      errorStreak = hadError ? Math.min(errorStreak + 1, 3) : 0
      schedule(Math.min(10_000, 2_000 * 2 ** errorStreak))
    }

    function handleVisibilityChange() {
      if (!document.hidden) {
        if (timer) {
          window.clearTimeout(timer)
        }
        void poll()
      }
    }

    document.addEventListener("visibilitychange", handleVisibilityChange)
    void poll()
    return () => {
      disposed = true
      controller?.abort()
      document.removeEventListener("visibilitychange", handleVisibilityChange)
      if (timer) {
        window.clearTimeout(timer)
      }
    }
  }, [activeSignature])
}
