import {
  enhanceDirectionStorageKey,
  enhanceDirectionValues,
  languageStorageKey,
  sizeStorageKey,
  sizeValues,
  type EnhanceDirection,
  type Language,
} from "./copy"

export const defaultLanguage: Language = "zh"
export const defaultSize = "1:1"
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
  return window.localStorage.getItem(key)
}

function writePreference(key: string, value: string) {
  if (typeof window === "undefined") {
    return
  }
  window.localStorage.setItem(key, value)
}

function browserLanguage(): Language {
  if (typeof window === "undefined") {
    return defaultLanguage
  }
  const languages = [
    ...(window.navigator.languages ?? []),
    window.navigator.language,
  ]
  const language = languages.find((value) => value.trim() !== "")
  if (!language) {
    return defaultLanguage
  }
  return language.toLowerCase().startsWith("zh") ? "zh" : "en"
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
  const browserLanguage = () => {
    const language =
      window.navigator.languages?.find((value) => value.trim() !== "") ||
      window.navigator.language ||
      defaultLanguage
    return language.toLowerCase().startsWith("zh") ? "zh" : "en"
  }
  const stored = window.localStorage.getItem(languageStorageKey)
  const language = stored === "zh" || stored === "en" ? stored : browserLanguage()
  document.documentElement.lang = documentLanguage(language)
})()
`.trim()
