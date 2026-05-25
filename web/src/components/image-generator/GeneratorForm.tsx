import {
  LightbulbIcon,
  ListPlusIcon,
  Loader2Icon,
  SparklesIcon,
  Trash2Icon,
  Undo2Icon,
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
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { cn } from "@/lib/utils"

import {
  enhanceDirectionValues,
  sizeValues,
  type EnhanceDirection,
  type ImageGeneratorCopy,
} from "./copy"
import { FallbackImage } from "./FallbackImage"
import { SizeOption } from "./SizeOption"
import { sizeLabel } from "./utils"

type GeneratorFormProps = {
  t: ImageGeneratorCopy
  prompt: string
  enhanceDirection: EnhanceDirection
  size: string
  referenceImage: string
  referenceName: string
  controlsDisabled: boolean
  enhanceDisabled: boolean
  restorePromptAvailable: boolean
  generateDisabled: boolean
  generateLabel?: string
  loading: boolean
  enhancing: boolean
  currentActive: boolean
  promptInputRef: RefObject<HTMLTextAreaElement | null>
  referenceInputRef: RefObject<HTMLInputElement | null>
  quotaPanel?: ReactNode
  onPromptChange: (value: string) => void
  onEnhanceDirectionChange: (value: EnhanceDirection) => void
  onEnhance: () => void
  onRestorePrompt: () => void
  onSizeChange: (value: string) => void
  onReferenceChange: (file?: File) => void
  onResetReference: () => void
  onGenerate: () => void
}

export function GeneratorForm({
  t,
  prompt,
  enhanceDirection,
  size,
  referenceImage,
  referenceName,
  controlsDisabled,
  enhanceDisabled,
  restorePromptAvailable,
  generateDisabled,
  generateLabel,
  loading,
  enhancing,
  currentActive,
  promptInputRef,
  referenceInputRef,
  quotaPanel,
  onPromptChange,
  onEnhanceDirectionChange,
  onEnhance,
  onRestorePrompt,
  onSizeChange,
  onReferenceChange,
  onResetReference,
  onGenerate,
}: GeneratorFormProps) {
  const selectedSizeLabel = sizeLabel(size, t)

  return (
    <div className="h-fit min-w-0 px-1 lg:sticky lg:top-5 xl:static xl:max-h-full xl:min-h-0 xl:overflow-auto xl:px-1">
      <FieldGroup className="gap-4">
        {quotaPanel}

        <Field data-disabled={controlsDisabled || undefined}>
          <FieldLabel htmlFor="prompt">{t.promptLabel}</FieldLabel>
          <Textarea
            ref={promptInputRef}
            id="prompt"
            value={prompt}
            onChange={(event) => onPromptChange(event.target.value)}
            placeholder={t.promptPlaceholder}
            disabled={controlsDisabled}
            className="min-h-28 max-h-64 resize-none overflow-y-auto bg-background/70 shadow-inner"
          />
        </Field>

        <Field data-disabled={controlsDisabled || undefined}>
          <div className="flex items-center justify-between gap-2">
            <FieldLabel>{t.enhanceLabel}</FieldLabel>
            <div className="flex items-center gap-2">
              {restorePromptAvailable ? (
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={onRestorePrompt}
                  disabled={controlsDisabled}
                >
                  <Undo2Icon data-icon="inline-start" />
                  {t.restorePrompt}
                </Button>
              ) : null}
              <Button
                type="button"
                size="sm"
                variant="default"
                onClick={onEnhance}
                disabled={enhanceDisabled}
              >
                {enhancing ? (
                  <Loader2Icon
                    data-icon="inline-start"
                    className="animate-spin"
                  />
                ) : (
                  <WandSparklesIcon data-icon="inline-start" />
                )}
                {enhancing ? t.enhancing : t.enhance}
              </Button>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-2">
            {enhanceDirectionValues.map((value) => {
              const DirectionIcon =
                value === "details" ? ListPlusIcon : LightbulbIcon
              const selected = value === enhanceDirection

              return (
                <Button
                  key={value}
                  type="button"
                  variant="outline"
                  size="sm"
                  aria-pressed={selected}
                  disabled={controlsDisabled}
                  onClick={() => onEnhanceDirectionChange(value)}
                  className={cn(
                    "w-full",
                    selected &&
                      "border-primary/55 bg-primary/10 text-primary hover:border-primary/70 hover:bg-primary/15 hover:text-primary"
                  )}
                >
                  <DirectionIcon data-icon="inline-start" />
                  {t.enhanceDirections[value]}
                </Button>
              )
            })}
          </div>
        </Field>

        <Field data-disabled={controlsDisabled || undefined}>
          <FieldLabel htmlFor={referenceImage ? undefined : "reference"}>
            {t.referenceLabel}
          </FieldLabel>
          <Input
            ref={referenceInputRef}
            id="reference"
            type="file"
            accept="image/png,image/jpeg,image/webp"
            disabled={controlsDisabled}
            onChange={(event) => onReferenceChange(event.target.files?.[0])}
            className="hidden"
          />
          {referenceImage ? (
            <div className="scene-history-row flex items-center gap-3 rounded-lg border p-2.5 shadow-sm">
              <FallbackImage
                src={referenceImage}
                alt={referenceName || t.referenceLabel}
                className="object-cover"
                wrapperClassName="aspect-square size-16 shrink-0 rounded-md"
              />
              <span className="min-w-0 flex-1 truncate text-sm font-medium">
                {referenceName}
              </span>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={onResetReference}
                disabled={controlsDisabled}
                className="ml-auto"
              >
                <Trash2Icon data-icon="inline-start" />
                {t.removeReference}
              </Button>
            </div>
          ) : (
            <>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => referenceInputRef.current?.click()}
                disabled={controlsDisabled}
                className="w-fit"
              >
                <UploadIcon data-icon="inline-start" />
                {t.selectReference}
              </Button>
              <FieldDescription className="text-xs">
                {t.referenceHelp}
              </FieldDescription>
            </>
          )}
        </Field>

        <div className="grid gap-3">
          <Field data-disabled={controlsDisabled || undefined}>
            <FieldLabel>{t.sizeLabel}</FieldLabel>
            <Select
              value={size}
              onValueChange={onSizeChange}
              disabled={controlsDisabled}
            >
              <SelectTrigger className="w-full">
                <SelectValue>
                  <SizeOption value={size} label={selectedSizeLabel} compact />
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {sizeValues.map((value) => (
                    <SelectItem key={value} value={value}>
                      <SizeOption value={value} label={t.sizes[value]} />
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>
        </div>

        <Button
          type="button"
          size="lg"
          onClick={onGenerate}
          disabled={generateDisabled}
          className="w-full"
        >
          {loading || currentActive ? (
            <Loader2Icon data-icon="inline-start" className="animate-spin" />
          ) : (
            <SparklesIcon data-icon="inline-start" />
          )}
          {loading
            ? t.submitting
            : currentActive
              ? t.generating
              : (generateLabel ?? t.generate)}
        </Button>
      </FieldGroup>
    </div>
  )
}
