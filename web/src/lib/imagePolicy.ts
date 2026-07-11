export const imageRetentionDays = 7
export const maxHistoryItems = 24

const imageRetentionMs = imageRetentionDays * 24 * 60 * 60 * 1000

export function isRecordExpired(createdAt: string, now = Date.now()) {
  const timestamp = Date.parse(createdAt)
  return Number.isFinite(timestamp) && now - timestamp >= imageRetentionMs
}
