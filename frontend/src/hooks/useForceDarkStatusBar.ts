import { useEffect, useRef } from "react"

const DARK_COLOR = "#000000"

export function useForceDarkStatusBar(isActive: boolean) {
  const restoreValueRef = useRef<string>("#ffffff")

  useEffect(() => {
    if (!isActive) return

    const meta = document.querySelector('meta[name="theme-color"]')
    if (!meta) return

    restoreValueRef.current = meta.getAttribute("content") ?? "#ffffff"
    meta.setAttribute("content", DARK_COLOR)

    const observer = new MutationObserver(() => {
      const current = meta.getAttribute("content")
      if (current && current !== DARK_COLOR) {
        restoreValueRef.current = current
        meta.setAttribute("content", DARK_COLOR)
      }
    })

    observer.observe(meta, { attributes: true, attributeFilter: ["content"] })

    return () => {
      observer.disconnect()
      meta.setAttribute("content", restoreValueRef.current)
    }
  }, [isActive])
}
