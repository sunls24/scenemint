import {
  enhanceDirectionStorageKey,
  enhanceDirectionValues,
  languageStorageKey,
  sizeStorageKey,
  sizeValues,
  type EnhanceDirection,
  type Language,
} from "./copy"

export const defaultLanguage: Language = "en"
export const defaultSize = "9:16"
export const defaultEnhanceDirection: EnhanceDirection = "details"

function isLanguage(value: string | null): value is Language {
  return value === "zh" || value === "en"
}

function isSize(value: string | null): value is (typeof sizeValues)[number] {
  return sizeValues.some((size) => size === value)
}

function isEnhanceDirection(
  value: string | null
): value is EnhanceDirection {
  return enhanceDirectionValues.some((direction) => direction === value)
}

function readPreference(key: string) {
  if (typeof window === "undefined") {
    return null
  }
  try {
    return window.localStorage.getItem(key)
  } catch {
    return null
  }
}

function writePreference(key: string, value: string) {
  if (typeof window === "undefined") {
    return
  }
  try {
    window.localStorage.setItem(key, value)
  } catch {
    // Preference persistence is optional; keep the app usable if storage is blocked.
  }
}

function browserLanguage(): Language {
  if (typeof window === "undefined") {
    return defaultLanguage
  }
  return window.navigator.language.toLowerCase().startsWith("zh")
    ? "zh"
    : defaultLanguage
}

export function initialLanguage(): Language {
  const stored = readPreference(languageStorageKey)
  return isLanguage(stored) ? stored : browserLanguage()
}

export function initialSize() {
  const stored = readPreference(sizeStorageKey)
  return isSize(stored) ? stored : defaultSize
}

export function initialEnhanceDirection(): EnhanceDirection {
  const stored = readPreference(enhanceDirectionStorageKey)
  return isEnhanceDirection(stored) ? stored : defaultEnhanceDirection
}

export function documentLanguage(language: Language) {
  return language === "zh" ? "zh-CN" : "en"
}

export function persistLanguage(language: Language) {
  writePreference(languageStorageKey, language)
  if (typeof document !== "undefined") {
    document.documentElement.lang = documentLanguage(language)
  }
}

export function persistSize(size: string) {
  writePreference(sizeStorageKey, size)
}

export function persistEnhanceDirection(direction: EnhanceDirection) {
  writePreference(enhanceDirectionStorageKey, direction)
}

export const languageBootScript = `
(() => {
  const defaultLanguage = ${JSON.stringify(defaultLanguage)}
  const languageStorageKey = ${JSON.stringify(languageStorageKey)}
  const documentLanguage = (language) => language === "zh" ? "zh-CN" : "en"
  try {
    const stored = window.localStorage.getItem(languageStorageKey)
    const language =
      stored === "zh" || stored === "en"
        ? stored
        : window.navigator.language.toLowerCase().startsWith("zh")
          ? "zh"
          : defaultLanguage
    document.documentElement.lang = documentLanguage(language)
  } catch {
    document.documentElement.lang = documentLanguage(defaultLanguage)
  }
})()
`.trim()
