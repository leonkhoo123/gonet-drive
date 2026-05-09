import { createContext, use, useEffect, useState } from "react"

type Theme = "dark" | "light" | "system"

interface ThemeProviderProps {
  children: React.ReactNode
  defaultTheme?: Theme
  storageKey?: string
}

interface ThemeProviderState {
  theme: Theme
  setTheme: (theme: Theme) => void
}

const ThemeProviderContext = createContext<ThemeProviderState | undefined>(undefined)

export function ThemeProvider({
  children,
  defaultTheme = "system",
  storageKey = "vite-ui-theme",
  ...props
}: ThemeProviderProps) {
  const [theme, setTheme] = useState<Theme>(
    () => {
      const stored = localStorage.getItem(storageKey);
      return stored === 'dark' || stored === 'light' || stored === 'system' ? stored : defaultTheme;
    }
  )

  useEffect(() => {
    const root = window.document.documentElement
    const meta = document.querySelector('meta[name="theme-color"]')

    const applyTheme = (resolvedTheme: "light" | "dark") => {
      root.classList.remove("light", "dark")
      root.classList.add(resolvedTheme)
      if (meta) {
        meta.setAttribute("content", resolvedTheme === "dark" ? "#0a0a0a" : "#ffffff")
      }
    }

    if (theme === "system") {
      const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)")
      const handler = (e: MediaQueryListEvent | MediaQueryList) => {
        applyTheme(e.matches ? "dark" : "light")
      }
      handler(mediaQuery)
      mediaQuery.addEventListener("change", handler)
      return () => { mediaQuery.removeEventListener("change", handler) }
    }

    applyTheme(theme)
  }, [theme])

  const value = {
    theme,
    setTheme: (theme: Theme) => {
      localStorage.setItem(storageKey, theme)
      setTheme(theme)
    },
  }

  return (
    <ThemeProviderContext {...props} value={value}>
      {children}
    </ThemeProviderContext>
  )
}

export const useTheme = () => {
  const context = use(ThemeProviderContext)

  if (context === undefined)
    throw new Error("useTheme must be used within a ThemeProvider")

  return context
}