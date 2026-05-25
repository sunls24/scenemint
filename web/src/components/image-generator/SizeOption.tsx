function sizeRatioName(value: string) {
  if (value === "16:9") {
    return "landscape"
  }
  if (value === "9:16") {
    return "portrait"
  }
  return "square"
}

function SizeRatioFrame({ value }: { value: string }) {
  return (
    <span
      className="scene-size-ratio"
      data-ratio={sizeRatioName(value)}
      aria-hidden="true"
    />
  )
}

type SizeOptionProps = {
  value: string
  label: string
  compact?: boolean
}

export function SizeOption({
  value,
  label,
  compact = false,
}: SizeOptionProps) {
  return (
    <span
      className={
        compact
          ? "flex min-w-0 items-center gap-2"
          : "flex min-w-0 items-center gap-2.5"
      }
    >
      <span className="scene-size-ratio-slot" aria-hidden="true">
        <SizeRatioFrame value={value} />
      </span>
      <span className="flex min-w-0 flex-col leading-none">
        <span className="truncate">{label}</span>
        {!compact && (
          <span className="mt-1 text-xs text-muted-foreground">{value}</span>
        )}
      </span>
    </span>
  )
}
