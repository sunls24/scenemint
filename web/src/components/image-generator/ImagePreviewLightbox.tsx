import { useMemo } from "react"
import Lightbox, { type Slide } from "yet-another-react-lightbox"
import Counter from "yet-another-react-lightbox/plugins/counter"
import Download from "yet-another-react-lightbox/plugins/download"
import Zoom from "yet-another-react-lightbox/plugins/zoom"
import "yet-another-react-lightbox/plugins/counter.css"
import "yet-another-react-lightbox/styles.css"

import type { ImageGeneratorCopy } from "./copy"
import type { PreviewImage } from "./utils"
import { downloadImage } from "./utils"

type ImagePreviewLightboxProps = {
  index: number
  previewItems: PreviewImage[]
  t: ImageGeneratorCopy
  onClose: () => void
  onView: (id: string) => void
}

export default function ImagePreviewLightbox({
  index,
  previewItems,
  t,
  onClose,
  onView,
}: ImagePreviewLightboxProps) {
  const slides = useMemo<Slide[]>(
    () =>
      previewItems.map((item) => ({
        src: item.image,
        alt: item.prompt,
        download: {
          url: item.image,
          filename: `scenemint-${item.id}.png`,
        },
      })),
    [previewItems]
  )

  return (
    <Lightbox
      className="scene-lightbox"
      open={index >= 0}
      close={onClose}
      index={Math.max(index, 0)}
      slides={slides}
      plugins={[Counter, Zoom, Download]}
      counter={{ separator: "/" }}
      carousel={{
        finite: slides.length <= 1,
        imageFit: "contain",
        padding: 24,
        preload: 0,
      }}
      controller={{ closeOnBackdropClick: true }}
      zoom={{
        maxZoomPixelRatio: 3,
        scrollToZoom: true,
        doubleClickMaxStops: 3,
        pinchZoomV4: true,
      }}
      download={{
        download: ({ slide }) => {
          const dl = slide.download
          if (typeof dl === "object" && dl !== null) {
            downloadImage(dl.url, dl.filename)
          }
        },
      }}
      labels={{
        Close: t.lightbox.close,
        Download: t.lightbox.download,
        Previous: t.lightbox.previous,
        Next: t.lightbox.next,
        "Zoom in": t.lightbox.zoomIn,
        "Zoom out": t.lightbox.zoomOut,
        "Photo gallery": t.lightbox.gallery,
        "{index} of {total}": t.lightbox.slideCount,
      }}
      on={{
        view: ({ index }) => {
          onView(previewItems[index]?.id ?? "")
        },
      }}
    />
  )
}
