import { ImageOffIcon, RotateCwIcon } from "lucide-react"
import { useState, type ImgHTMLAttributes } from "react"

import { cn } from "@/lib/utils"

type ImageSource = ImgHTMLAttributes<HTMLImageElement>["src"]

type FallbackImageProps = ImgHTMLAttributes<HTMLImageElement> & {
  wrapperClassName?: string
}

export function FallbackImage({
  wrapperClassName,
  className,
  src,
  alt,
  onLoad,
  onError,
  ...props
}: FallbackImageProps) {
  const [loadedSrc, setLoadedSrc] = useState<ImageSource>()
  const [failedSrc, setFailedSrc] = useState<ImageSource>()
  const loaded = Boolean(src) && loadedSrc === src
  const failed = Boolean(src) && failedSrc === src

  return (
    <span
      className={cn(
        "scene-image-fallback relative flex items-center justify-center overflow-hidden",
        wrapperClassName
      )}
      data-state={loaded ? "loaded" : failed ? "error" : "loading"}
    >
      <img
        {...props}
        key={src}
        src={src}
        alt={alt}
        className={cn("scene-image-fallback-img", className)}
        onLoad={(event) => {
          event.currentTarget.style.visibility = "visible"
          setFailedSrc(undefined)
          setLoadedSrc(src)
          onLoad?.(event)
        }}
        onError={(event) => {
          event.currentTarget.style.visibility = "hidden"
          setFailedSrc(src)
          onError?.(event)
        }}
      />
      {!loaded && !failed && (
        <span
          className="scene-image-fallback-mark"
          aria-hidden="true"
        >
          <RotateCwIcon />
        </span>
      )}
      {failed && (
        <span className="scene-image-fallback-error">
          <ImageOffIcon aria-hidden="true" />
        </span>
      )}
    </span>
  )
}
