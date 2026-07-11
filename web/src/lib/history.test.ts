import { beforeEach, describe, expect, test } from "bun:test"

import { $history, restoreHistory, type ImageHistory } from "./history"

function item(id: string): ImageHistory {
  return {
    id,
    mode: "text",
    prompt: id,
    size: "1:1",
    status: "completed",
    createdAt: "2026-07-10T08:00:00.000Z",
  }
}

describe("restoreHistory", () => {
  beforeEach(() => {
    $history.set([])
  })

  test("keeps records added after deletion ahead of restored records", () => {
    const live = Array.from({ length: 24 }, (_, index) => item(`live-${index}`))
    const removed = Array.from({ length: 24 }, (_, index) =>
      item(`removed-${index}`)
    )
    $history.set(live)

    restoreHistory(removed)

    expect($history.get().map((entry) => entry.id)).toEqual(
      live.map((entry) => entry.id)
    )
  })

  test("restores missing records when capacity is available", () => {
    $history.set([item("live")])

    restoreHistory([item("removed")])

    expect($history.get().map((entry) => entry.id)).toEqual([
      "live",
      "removed",
    ])
  })
})
