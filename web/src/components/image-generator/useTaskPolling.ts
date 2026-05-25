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
    const activeIds = activeSignature.split("|")
    const latestItem = (id: string) => {
      const latest = latestItemsRef.current
      if (latest.currentTask?.id === id) {
        return latest.currentTask
      }
      return latest.history.find((item) => item.id === id)
    }

    async function poll() {
      await Promise.all(
        activeIds.map(async (id) => {
          try {
            const task = await getJSON<TaskResponse>(
              `/api/images/tasks/${encodeURIComponent(id)}`
            )
            if (disposed) {
              return
            }
            const item = latestItem(id)
            if (!item || !isActive(item)) {
              return
            }
            updateTask(id, mergeTask(item, task))
          } catch {
            // Keep active tasks pending on transient network/server errors.
          }
        })
      )
      if (!disposed) {
        timer = window.setTimeout(() => void poll(), 2000)
      }
    }

    void poll()
    return () => {
      disposed = true
      if (timer) {
        window.clearTimeout(timer)
      }
    }
  }, [activeSignature])
}
