import type { ImageHistory, ImageStatus } from "@/lib/history"

import type { ImageGeneratorCopy } from "./copy"

/** 通过 fetch blob 下载图片，失败时在新标签页打开 */
export async function downloadImage(url: string, filename: string) {
  try {
    const res = await fetch(url)
    const blob = await res.blob()
    const blobUrl = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = blobUrl
    a.download = filename
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(blobUrl)
  } catch {
    window.open(url, "_blank")
  }
}

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

export function statusLabel(
  status: ImageStatus,
  t: ImageGeneratorCopy
): string {
  return t.status[status]
}
