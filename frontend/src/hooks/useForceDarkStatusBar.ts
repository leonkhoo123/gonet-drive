import { useEffect, useRef } from "react"

const DARK_COLOR = "#000000"

export function useForceDarkStatusBar(isActive: boolean) {
  const restoreValueRef = useRef<string>("#ffffff")
  const restoreIosStyleRef = useRef<string>("black-translucent")

  useEffect(() => {
    if (!isActive) return

    const themeMeta = document.querySelector('meta[name="theme-color"]')
    const iosMeta = document.querySelector(
      'meta[name="apple-mobile-web-app-status-bar-style"]',
    )

    if (themeMeta) {
      restoreValueRef.current = themeMeta.getAttribute("content") ?? "#ffffff"
      themeMeta.setAttribute("content", DARK_COLOR)
    }

    if (iosMeta) {
      restoreIosStyleRef.current =
        iosMeta.getAttribute("content") ?? "black-translucent"
      iosMeta.setAttribute("content", "black")
    }

    return () => {
      if (themeMeta) {
        themeMeta.setAttribute("content", restoreValueRef.current)
      }
      if (iosMeta) {
        iosMeta.setAttribute("content", restoreIosStyleRef.current)
      }
    }
  }, [isActive])
}
