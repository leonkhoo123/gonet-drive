import { Moon, Sun } from "lucide-react"
import { Button } from "@/components/ui/button"
import { useTheme } from "@/components/theme-provider"
import { useEffect, useState } from "react"

export function ShareModeToggle() {
  const { theme, setTheme } = useTheme()
  const [isSystemDark, setIsSystemDark] = useState(false)

  useEffect(() => {
    setIsSystemDark(window.matchMedia("(prefers-color-scheme: dark)").matches)

    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)")
    const handler = (e: MediaQueryListEvent) => { setIsSystemDark(e.matches) }
    mediaQuery.addEventListener("change", handler)
    return () => { mediaQuery.removeEventListener("change", handler) }
  }, [])

  const toggleTheme = () => {
    // Mark that the user has manually toggled the theme
    sessionStorage.setItem("share_theme_toggled", "true")
    
    if (theme === "system") {
      setTheme(isSystemDark ? "light" : "dark")
    } else if (theme === "dark") {
      setTheme("light")
    } else {
      setTheme("dark")
    }
  }

  const isDark = theme === "dark" || (theme === "system" && isSystemDark)

  return (
    <Button 
      variant="ghost" 
      size="icon" 
      className="h-8 w-8 text-muted-foreground hover:text-foreground" 
      onClick={toggleTheme} 
      title="Toggle Theme"
    >
      {isDark ? (
        <Moon className="h-[1.2rem] w-[1.2rem] transition-all" />
      ) : (
        <Sun className="h-[1.2rem] w-[1.2rem] transition-all" />
      )}
      <span className="sr-only">Toggle theme</span>
    </Button>
  )
}
