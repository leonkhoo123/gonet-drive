import { getConfig } from '@/config'
import { useState } from 'react'

interface LogoProps {
  src?: string
  className?: string
}

export function Logo({ src, className = '' }: LogoProps) {
  const [hasError, setHasError] = useState(false)

  if (hasError) {
    return null
  }

  return (
    <img
      src={src ?? `${getConfig().apiBaseUrl}/config/logo`}
      alt="Logo"
      className={className}
      onError={() => { setHasError(true); }}
    />
  )
}
