import type { SVGProps } from "react"
import { LanguagesIcon } from "lucide-react"

import { Button } from "@/components/ui/button"

import type { ImageGeneratorCopy, Language } from "./copy"

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
  language: Language
  t: ImageGeneratorCopy
  onLanguageChange: (language: Language) => void
}

export function BrandHeader({
  appVersion,
  githubUrl,
  language,
  t,
  onLanguageChange,
}: BrandHeaderProps) {
  return (
    <header className="flex w-full flex-col gap-4 md:flex-row md:items-center md:justify-between xl:shrink-0">
      <div className="scene-brand-lockup ml-1 flex items-center gap-3 md:ml-1.5 md:gap-4">
        <img
          src="/brand-logo.png"
          alt=""
          aria-hidden="true"
          width="296"
          height="256"
          className="scene-brand-mark h-12 w-auto shrink-0 md:h-16"
        />
        <div className="scene-brand-copy flex min-w-0 flex-col gap-2">
          <h1 className="scene-brand-wordmark text-[1.58rem] font-bold leading-none tracking-normal md:text-[1.9rem]">
            <span className="scene-brand-wordmark-main">Scene</span>
            <span className="scene-brand-wordmark-accent">Mint</span>
          </h1>
          <p className="scene-brand-slogan max-w-2xl text-[0.88rem] font-medium leading-5 text-muted-foreground md:text-[0.95rem]">
            {t.appDescription}
          </p>
        </div>
      </div>
      <div className="scene-header-actions flex flex-col items-start gap-1.5 md:items-end md:pb-1">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => onLanguageChange(language === "zh" ? "en" : "zh")}
          className="bg-background/70 shadow-sm"
        >
          <LanguagesIcon data-icon="inline-start" />
          {t.language}
        </Button>
        <p className="font-mono flex flex-wrap items-center gap-x-3 gap-y-1 text-xs leading-5 text-muted-foreground/70 md:justify-end">
          <span>Vibe coding by Codex App</span>
          <span aria-hidden="true">/</span>
          <a
            href={githubUrl}
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-1 rounded-sm transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40"
            aria-label={`SceneMint GitHub repository v${appVersion}`}
          >
            <GithubIcon className="size-3.5" aria-hidden="true" />
            GitHub
            <span>v{appVersion}</span>
          </a>
        </p>
      </div>
    </header>
  )
}
