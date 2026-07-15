import { useEffect, useRef, useState } from "react"
import { toast } from "sonner"

import { postStream } from "@/lib/http"
import { readChatCompletionStream } from "@/lib/openaiStream"

import {
  maxReferenceSize,
  type EnhanceDirection,
  type ImageGeneratorCopy,
} from "./copy"
import {
  initialEnhanceDirection,
  initialSize,
  persistEnhanceDirection,
  persistSize,
} from "./preferences"

export function useCreationDraft(t: ImageGeneratorCopy) {
  const promptInputRef = useRef<HTMLTextAreaElement>(null)
  const referenceInputRef = useRef<HTMLInputElement>(null)
  const referencePreviewURLRef = useRef("")
  const enhanceAbortRef = useRef<AbortController | null>(null)
  const [prompt, setPrompt] = useState("")
  const [enhanceDirection, setEnhanceDirection] =
    useState<EnhanceDirection>(initialEnhanceDirection)
  const [enhanceOriginalPrompt, setEnhanceOriginalPrompt] = useState<string | null>(null)
  const [size, setSize] = useState(initialSize)
  const [referenceImage, setReferenceImage] = useState("")
  const [referenceFile, setReferenceFile] = useState<File | null>(null)
  const [referenceName, setReferenceName] = useState("")
  const [enhancing, setEnhancing] = useState(false)

  useEffect(() => {
    persistSize(size)
  }, [size])

  useEffect(() => {
    persistEnhanceDirection(enhanceDirection)
  }, [enhanceDirection])

  useEffect(() => {
    return () => {
      enhanceAbortRef.current?.abort()
      revokeReferencePreview()
    }
  }, [])

  function selectReference(file?: File) {
    if (!file) {
      resetReference()
      return
    }
    if (!file.type.startsWith("image/")) {
      resetReference()
      toast.error(t.toast.selectImage)
      return
    }
    if (file.size > maxReferenceSize) {
      resetReference()
      toast.error(t.toast.referenceTooLarge)
      return
    }
    const previewURL = URL.createObjectURL(file)
    revokeReferencePreview()
    referencePreviewURLRef.current = previewURL
    setReferenceImage(previewURL)
    setReferenceFile(file)
    setReferenceName(file.name)
  }

  async function enhancePrompt() {
    if (enhancing) {
      return
    }
    const sourcePrompt = prompt.trim()
    if (!sourcePrompt) {
      toast.error(t.toast.enhanceRequired)
      return
    }

    enhanceAbortRef.current?.abort()
    const controller = new AbortController()
    enhanceAbortRef.current = controller
    setEnhanceOriginalPrompt(sourcePrompt)
    setEnhancing(true)

    let enhancedPrompt = ""
    try {
      const stream = await postStream(
        "/api/prompts/enhance",
        {
          prompt: sourcePrompt,
          direction: enhanceDirection,
        },
        { signal: controller.signal }
      )
      await readChatCompletionStream(stream, (delta) => {
        enhancedPrompt += delta
        if (enhancedPrompt.trim()) {
          setPrompt(enhancedPrompt)
        }
      })
      if (!enhancedPrompt.trim()) {
        setEnhanceOriginalPrompt(null)
        toast.error(t.toast.enhanceEmpty)
      }
    } catch (err) {
      if (controller.signal.aborted) {
        return
      }
      if (!enhancedPrompt.trim()) {
        setPrompt(sourcePrompt)
        setEnhanceOriginalPrompt(null)
        toast.error(err instanceof Error ? err.message : t.toast.enhanceFailed)
      } else {
        toast.error(t.toast.enhancePartial)
      }
    } finally {
      if (enhanceAbortRef.current === controller) {
        enhanceAbortRef.current = null
        setEnhancing(false)
      }
    }
  }

  function changePrompt(value: string) {
    setPrompt(value)
    setEnhanceOriginalPrompt(null)
  }

  function discardEnhancement() {
    enhanceAbortRef.current?.abort()
    if (enhanceOriginalPrompt !== null) {
      setPrompt(enhanceOriginalPrompt)
    }
    setEnhanceOriginalPrompt(null)
    setEnhancing(false)
  }

  function clearEnhancementUndo() {
    setEnhanceOriginalPrompt(null)
  }

  function reusePrompt(value: string) {
    setPrompt(value)
    setEnhanceOriginalPrompt(null)
    toast.success(t.toast.promptReused)
    window.requestAnimationFrame(() => {
      const input = promptInputRef.current
      if (!input || input.disabled) {
        return
      }
      input.focus()
      input.setSelectionRange(input.value.length, input.value.length)
    })
  }

  function resetReference() {
    revokeReferencePreview()
    setReferenceImage("")
    setReferenceFile(null)
    setReferenceName("")
    if (referenceInputRef.current) {
      referenceInputRef.current.value = ""
    }
  }

  function revokeReferencePreview() {
    if (!referencePreviewURLRef.current) {
      return
    }
    URL.revokeObjectURL(referencePreviewURLRef.current)
    referencePreviewURLRef.current = ""
  }

  return {
    prompt,
    enhanceDirection,
    enhanceOriginalPrompt,
    size,
    referenceImage,
    referenceFile,
    referenceName,
    enhancing,
    promptInputRef,
    referenceInputRef,
    setEnhanceDirection,
    setSize,
    selectReference,
    resetReference,
    changePrompt,
    enhancePrompt,
    discardEnhancement,
    clearEnhancementUndo,
    reusePrompt,
  }
}
