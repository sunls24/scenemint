import { persistentAtom } from "@nanostores/persistent"

import { isRecordExpired, maxHistoryItems } from "@/lib/imagePolicy"

export type ImageStatus = "queued" | "running" | "completed" | "failed"

export type ImageHistory = {
  id: string
  mode: "text" | "image"
  prompt: string
  size: string
  status?: ImageStatus
  image?: string
  error?: string
  revisedPrompt?: string
  referenceName?: string
  createdAt: string
  updatedAt?: string
}

function uniqueHistory(items: ImageHistory[]) {
  const seen = new Set<string>()
  const next: ImageHistory[] = []
  for (const item of items) {
    if (seen.has(item.id)) {
      continue
    }
    seen.add(item.id)
    next.push(item)
  }
  return next.slice(0, maxHistoryItems)
}

function isHistoryItem(value: unknown): value is ImageHistory {
  if (!value || typeof value !== "object") {
    return false
  }
  const item = value as Partial<ImageHistory>
  return (
    typeof item.id === "string" &&
    typeof item.prompt === "string" &&
    typeof item.size === "string" &&
    typeof item.createdAt === "string" &&
    (item.mode === "text" || item.mode === "image")
  )
}

function decodeHistory(value: string) {
  try {
    const parsed = JSON.parse(value) as unknown
    if (!Array.isArray(parsed)) {
      return []
    }
    return uniqueHistory(
      parsed.filter(isHistoryItem).filter((item) => !isRecordExpired(item.createdAt))
    )
  } catch {
    return []
  }
}

function decodeCurrentTask(value: string) {
  try {
    const parsed = JSON.parse(value) as unknown
    if (!isHistoryItem(parsed) || isRecordExpired(parsed.createdAt)) {
      return null
    }
    return parsed
  } catch {
    return null
  }
}

export const $history = persistentAtom<ImageHistory[]>(
  "scenemint:history",
  [],
  {
    encode: JSON.stringify,
    decode: decodeHistory,
  }
)

export const $currentTask = persistentAtom<ImageHistory | null>(
  "scenemint:current",
  null,
  {
    encode: JSON.stringify,
    decode: decodeCurrentTask,
  }
)

function updateHistory(id: string, patch: Partial<ImageHistory>) {
  $history.set(
    $history.get().map((item) =>
      item.id === id ? { ...item, ...patch } : item
    )
  )
}

export function removeHistory(id: string) {
  $history.set($history.get().filter((item) => item.id !== id))
}

export function setCurrentTask(item: ImageHistory | null) {
  $currentTask.set(item)
}

function updateCurrentTask(patch: Partial<ImageHistory>) {
  const current = $currentTask.get()
  if (!current) {
    return
  }
  $currentTask.set({ ...current, ...patch })
}

export function archiveCurrentAndSet(next: ImageHistory) {
  const current = $currentTask.get()
  const history = $history
    .get()
    .filter((item) => item.id !== next.id && item.id !== current?.id)

  if (current && current.id !== next.id) {
    $history.set(uniqueHistory([current, ...history]))
  } else {
    $history.set(uniqueHistory(history))
  }
  $currentTask.set(next)
}

export function updateTask(id: string, patch: Partial<ImageHistory>) {
  const current = $currentTask.get()
  if (current?.id === id) {
    updateCurrentTask(patch)
    return
  }
  updateHistory(id, patch)
}

export function clearHistory() {
  $history.set([])
}

export function restoreHistory(items: ImageHistory[]) {
  $history.set(uniqueHistory([...$history.get(), ...items]))
}
