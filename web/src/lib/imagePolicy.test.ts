import { describe, expect, test } from "bun:test"

import { imageRetentionDays, isRecordExpired } from "./imagePolicy"

describe("image retention policy", () => {
  const day = 24 * 60 * 60 * 1000
  const now = Date.parse("2026-07-10T08:00:00.000Z")

  test("keeps images inside the retention window", () => {
    const createdAt = new Date(now - imageRetentionDays * day + 1).toISOString()
    expect(isRecordExpired(createdAt, now)).toBe(false)
  })

  test("expires images at the retention boundary", () => {
    const createdAt = new Date(now - imageRetentionDays * day).toISOString()
    expect(isRecordExpired(createdAt, now)).toBe(true)
  })

  test("does not expire invalid timestamps", () => {
    expect(isRecordExpired("", now)).toBe(false)
  })
})
