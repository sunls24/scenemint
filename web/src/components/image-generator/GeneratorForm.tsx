import {
  ImagePlusIcon,
  LightbulbIcon,
  ListPlusIcon,
  Loader2Icon,
  SparklesIcon,
  Trash2Icon,
  UploadIcon,
  WandSparklesIcon,
} from "lucide-react"
import type { ReactNode, RefObject } from "react"

import { Button } from "@/components/ui/button"
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { cn } from "@/lib/utils"

import {
  enhanceDirectionValues,
  sizeValues,
  type EnhanceDirection,
  type ImageGeneratorCopy,
} from "./copy"
import { FallbackImage } from "./FallbackImage"
import { SizeOption } from "./SizeOption"

type GeneratorFormProps = {
  t: ImageGeneratorCopy
  prompt: string
  enhanceDirection: EnhanceDirection
  enhanceOriginalPrompt: string | null
  size: string
  referenceImage: string
  referenceName: string
  editingDisabled: boolean
  enhanceDisabled: boolean
  enhancing: boolean
  primaryDisabled: boolean
  promptInputRef: RefObject<HTMLTextAreaElement | null>
  referenceInputRef: RefObject<HTMLInputElement | null>
  quotaPanel?: ReactNode
  onPromptChange: (value: string) => void
  onEnhanceDirectionChange: (value: EnhanceDirection) => void
  onEnhance: () => void
  onDiscardEnhancement: () => void
  onSizeChange: (value: string) => void
  onReferenceChange: (file?: File) => void
  onResetReference: () => void
  onPrimaryAction: () => void
}

export function GeneratorForm({
  t,
  prompt,
  enhanceDirection,
  enhanceOriginalPrompt,
  size,
  referenceImage,
  referenceName,
  editingDisabled,
  enhanceDisabled,
  enhancing,
  primaryDisabled,
  promptInputRef,
  referenceInputRef,
  quotaPanel,
  onPromptChange,
  onEnhanceDirectionChange,
  onEnhance,
  onDiscardEnhancement,
  onSizeChange,
  onReferenceChange,
  onResetReference,
  onPrimaryAction,
}: GeneratorFormProps) {
  const enhancementApplied = enhanceOriginalPrompt !== null

  return (
    <aside
      className="scene-controls-panel flex min-h-0 self-start flex-col overflow-hidden lg:max-h-full"
      aria-labelledby="creation-controls-title"
    >
      <div className="scene-panel-heading shrink-0 border-b px-3.5 py-2.5">
        <div className="scene-panel-index" aria-hidden="true">01</div>
        <h2 id="creation-controls-title" className="mt-0.5 text-lg font-semibold">
          {t.controlsTitle}
        </h2>
      </div>

      <div className="scene-controls-scroll min-h-0 flex-1 overflow-y-auto px-4 py-4">
        <FieldGroup className="gap-5">
          {quotaPanel}

          <Field data-disabled={editingDisabled || undefined}>
            <FieldLabel htmlFor="prompt" className="scene-field-label">
              {t.promptLabel}
            </FieldLabel>
            <Textarea
              ref={promptInputRef}
              id="prompt"
              value={prompt}
              onChange={(event) => onPromptChange(event.target.value)}
              placeholder={t.promptPlaceholder}
              disabled={editingDisabled || enhancing}
              className={cn(
                "scene-prompt-input min-h-32 max-h-72 resize-none overflow-y-auto",
                enhancementApplied && "is-enhancement-preview"
              )}
            />
            {enhancementApplied && (
              <div
                className="scene-enhance-preview-bar flex items-center justify-between gap-3 rounded-md px-2.5 py-1.5"
                role="status"
                aria-live="polite"
              >
                <div className="flex min-w-0 items-center gap-1.5 text-xs font-medium">
                  {enhancing ? (
                    <Loader2Icon className="size-3.5 shrink-0 animate-spin" />
                  ) : (
                    <WandSparklesIcon className="size-3.5 shrink-0" />
                  )}
                  <span className="truncate">
                    {enhancing ? t.enhancing : t.enhancementApplied}
                  </span>
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={onDiscardEnhancement}
                >
                  {t.discardEnhanced}
                </Button>
              </div>
            )}
          </Field>

          <Field data-disabled={editingDisabled || undefined}>
            <div className="flex items-center justify-between gap-2">
              <FieldLabel className="scene-field-label">{t.enhanceLabel}</FieldLabel>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={onEnhance}
                disabled={enhanceDisabled}
                className="scene-enhance-button"
              >
                {enhancing ? (
                  <Loader2Icon data-icon="inline-start" className="animate-spin" />
                ) : (
                  <WandSparklesIcon data-icon="inline-start" />
                )}
                {enhancing ? t.enhancing : t.enhance}
              </Button>
            </div>
            <ToggleGroup
              value={[enhanceDirection]}
              onValueChange={(values) => {
                const nextDirection = values[0] as EnhanceDirection | undefined
                if (nextDirection) onEnhanceDirectionChange(nextDirection)
              }}
              disabled={editingDisabled}
              variant="outline"
              spacing={1}
              aria-label={t.enhanceLabel}
              className="scene-enhance-toggle-group grid w-full grid-cols-2"
            >
              {enhanceDirectionValues.map((value) => {
                const DirectionIcon =
                  value === "details" ? ListPlusIcon : LightbulbIcon
                return (
                  <ToggleGroupItem
                    key={value}
                    value={value}
                    aria-label={t.enhanceDirections[value]}
                    className="scene-choice-button w-full"
                  >
                    <DirectionIcon data-icon="inline-start" />
                    {t.enhanceDirections[value]}
                  </ToggleGroupItem>
                )
              })}
            </ToggleGroup>
          </Field>

          <Field data-disabled={editingDisabled || undefined}>
            <FieldLabel htmlFor={referenceImage ? undefined : "reference"} className="scene-field-label">
              {t.referenceLabel}
            </FieldLabel>
            <Input
              ref={referenceInputRef}
              id="reference"
              type="file"
              accept="image/png,image/jpeg,image/webp"
              disabled={editingDisabled}
              onChange={(event) => onReferenceChange(event.target.files?.[0])}
              className="hidden"
            />
            {referenceImage ? (
              <div className="scene-reference-preview flex items-center gap-3 rounded-lg border p-2.5">
                <FallbackImage
                  src={referenceImage}
                  alt={referenceName || t.referenceLabel}
                  className="object-cover"
                  wrapperClassName="aspect-square size-16 shrink-0 rounded-md"
                />
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-medium">{referenceName}</span>
                  <span className="mt-1 block text-xs text-muted-foreground">
                    {t.referenceEnabled}
                  </span>
                </span>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  onClick={onResetReference}
                  disabled={editingDisabled}
                  aria-label={t.removeReference}
                >
                  <Trash2Icon />
                </Button>
              </div>
            ) : (
              <button
                type="button"
                onClick={() => referenceInputRef.current?.click()}
                disabled={editingDisabled}
                className="scene-upload-zone flex w-full items-center gap-3 rounded-lg border px-3 py-3 text-left disabled:cursor-not-allowed disabled:opacity-50"
              >
                <span className="scene-upload-mark flex size-9 shrink-0 items-center justify-center rounded-md">
                  <ImagePlusIcon aria-hidden="true" />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="flex items-center gap-1.5 text-sm font-medium">
                    <UploadIcon aria-hidden="true" />
                    {t.selectReference}
                  </span>
                  <span className="mt-0.5 block text-xs text-muted-foreground">
                    {t.referenceHelp}
                  </span>
                </span>
              </button>
            )}
          </Field>

          <Field data-disabled={editingDisabled || undefined}>
            <FieldLabel className="scene-field-label">{t.sizeLabel}</FieldLabel>
            <ToggleGroup
              value={[size]}
              onValueChange={(values) => {
                const nextSize = values[0]
                if (nextSize) onSizeChange(nextSize)
              }}
              disabled={editingDisabled}
              spacing={1}
              aria-label={t.sizeLabel}
              className="scene-size-toggle-group grid w-full grid-cols-3"
            >
              {sizeValues.map((value) => (
                <ToggleGroupItem
                  key={value}
                  value={value}
                  aria-label={`${t.sizes[value]} ${value}`}
                  className="scene-size-toggle-item w-full"
                >
                  <SizeOption value={value} label={t.sizes[value]} compact />
                </ToggleGroupItem>
              ))}
            </ToggleGroup>
          </Field>
        </FieldGroup>
      </div>

      <div className="scene-primary-action-wrap shrink-0 border-t p-3">
        <Button
          type="button"
          size="lg"
          onClick={onPrimaryAction}
          disabled={primaryDisabled}
          className="scene-primary-action h-11 w-full"
        >
          <SparklesIcon data-icon="inline-start" />
          {t.generate}
        </Button>
      </div>
    </aside>
  )
}
