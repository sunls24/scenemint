import type { SVGProps } from "react"
import { ArrowUpRightIcon, LanguagesIcon } from "lucide-react"

import { Button } from "@/components/ui/button"

import type { ImageGeneratorCopy, Language } from "./copy"
import { PaidSiteSupport } from "./PaidSiteSupport"

export type RelatedLinks = {
  divination: string
  tempMail: string
}

function GithubIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 16 16" fill="currentColor" {...props}>
      <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82A7.65 7.65 0 0 1 8 3.86c.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z" />
    </svg>
  )
}

type BrandHeaderProps = {
  appVersion: string
  githubUrl: string
  relatedLinks: RelatedLinks
  language: Language
  t: ImageGeneratorCopy
  onLanguageChange: (language: Language) => void
}

export function BrandHeader({
  appVersion,
  githubUrl,
  relatedLinks,
  language,
  t,
  onLanguageChange,
}: BrandHeaderProps) {
  return (
    <header className="scene-topbar flex shrink-0 flex-col gap-3 border-b py-2.5 md:flex-row md:items-center md:justify-between">
      <div className="flex min-w-0 items-center gap-3">
        <img
          src="/brand-logo.png"
          alt=""
          aria-hidden="true"
          width="296"
          height="256"
          draggable={false}
          loading="eager"
          decoding="async"
          className="scene-brand-mark h-11 w-auto shrink-0 sm:h-12"
        />
        <div className="min-w-0">
          <div className="flex items-baseline">
            <h1 className="scene-brand-wordmark truncate text-[1.45rem] font-semibold tracking-[-0.04em] sm:text-[1.65rem]">
              SceneMint
            </h1>
          </div>
          <p className="text-xs text-muted-foreground sm:text-sm">
            {t.appDescription}
          </p>
        </div>
      </div>

      <div className="scene-topbar-actions flex min-w-0 flex-wrap items-center gap-1.5 md:justify-end">
        <PaidSiteSupport t={t} />

        <nav
          className="scene-related-nav mr-auto flex items-center gap-1 md:mr-1"
          aria-label={t.relatedLinksLabel}
        >
          <a href={relatedLinks.divination} target="_blank" rel="noreferrer">
            {t.relatedLinks.divination}
            <ArrowUpRightIcon aria-hidden="true" />
          </a>
          <a href={relatedLinks.tempMail} target="_blank" rel="noreferrer">
            {t.relatedLinks.tempMail}
            <ArrowUpRightIcon aria-hidden="true" />
          </a>
        </nav>

        <Button
          render={<a href={githubUrl} target="_blank" rel="noreferrer" />}
          variant="ghost"
          size="sm"
          className="scene-utility-button"
          aria-label={`SceneMint GitHub repository v${appVersion}`}
        >
          <GithubIcon data-icon="inline-start" aria-hidden="true" />
          <span>v{appVersion}</span>
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => onLanguageChange(language === "zh" ? "en" : "zh")}
          className="scene-utility-button"
        >
          <LanguagesIcon data-icon="inline-start" />
          {t.language}
        </Button>
      </div>
    </header>
  )
}
