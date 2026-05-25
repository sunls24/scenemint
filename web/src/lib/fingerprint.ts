let visitorIdPromise: Promise<string> | undefined

export function getVisitorFingerprint() {
  if (!visitorIdPromise) {
    visitorIdPromise = (async () => {
      const { default: FingerprintJS } = await import(
        "@fingerprintjs/fingerprintjs"
      )
      const agent = await FingerprintJS.load()
      const result = await agent.get()
      return result.visitorId
    })().catch((err) => {
      visitorIdPromise = undefined
      throw err
    })
  }
  return visitorIdPromise
}
