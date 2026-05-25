import type { ImageHistory, ImageStatus } from "@/lib/history"

import type { ImageGeneratorCopy } from "./copy"

export type TaskResponse = Omit<ImageHistory, "referenceName"> & {
  remainingCredits?: number
}
export type PreviewImage = ImageHistory & { image: string }

export function statusOf(item: ImageHistory): ImageStatus {
  if (item.status) {
    return item.status
  }
  return item.image ? "completed" : "queued"
}

export function isActive(item: ImageHistory) {
  const status = statusOf(item)
  return status === "queued" || status === "running"
}

export function canPreview(
  item: ImageHistory | null | undefined
): item is PreviewImage {
  if (!item?.image) {
    return false
  }
  return statusOf(item) === "completed"
}

export function dimensionsOf(size: string) {
  const ratioMatch = /^(\d+):(\d+)$/.exec(size)
  if (ratioMatch) {
    return { width: Number(ratioMatch[1]), height: Number(ratioMatch[2]) }
  }
  const pixelMatch = /^(\d+)x(\d+)$/.exec(size)
  if (pixelMatch) {
    return { width: Number(pixelMatch[1]), height: Number(pixelMatch[2]) }
  }
  return { width: 1, height: 1 }
}

export function sizeLabel(size: string, t: ImageGeneratorCopy) {
  return t.sizes[size as keyof typeof t.sizes] ?? size
}

export function mergeTask(
  item: ImageHistory,
  task: TaskResponse
): ImageHistory {
  return {
    ...item,
    mode: task.mode || item.mode,
    size: task.size || item.size,
    status: task.status ?? item.status,
    image: task.image || item.image,
    error: task.error,
    revisedPrompt: task.revisedPrompt || item.revisedPrompt,
    updatedAt: task.updatedAt || item.updatedAt,
  }
}

export function readFile(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result))
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(file)
  })
}

export function statusLabel(
  status: ImageStatus,
  t: ImageGeneratorCopy
): string {
  return t.status[status]
}
