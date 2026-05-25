import { RotateCwIcon } from "lucide-react"
import { useState, type ImgHTMLAttributes } from "react"

import { cn } from "@/lib/utils"

type ImageSource = ImgHTMLAttributes<HTMLImageElement>["src"]

type FallbackImageProps = Omit<
  ImgHTMLAttributes<HTMLImageElement>,
  "onLoad" | "onError"
> & {
  wrapperClassName?: string
}

export function FallbackImage({
  wrapperClassName,
  className,
  src,
  alt,
  ...props
}: FallbackImageProps) {
  const [loadedSrc, setLoadedSrc] = useState<ImageSource>()
  const loaded = Boolean(src) && loadedSrc === src

  return (
    <span
      className={cn(
        "scene-image-fallback relative flex items-center justify-center overflow-hidden",
        wrapperClassName
      )}
      data-state={loaded ? "loaded" : "loading"}
    >
      <img
        {...props}
        key={src}
        src={src}
        alt={alt}
        className={cn("scene-image-fallback-img", className)}
        onLoad={(event) => {
          event.currentTarget.style.visibility = "visible"
          setLoadedSrc(src)
        }}
        onError={(event) => {
          event.currentTarget.style.visibility = "hidden"
        }}
      />
      {!loaded && (
        <span
          className="scene-image-fallback-mark"
          aria-hidden="true"
        >
          <RotateCwIcon />
        </span>
      )}
    </span>
  )
}
