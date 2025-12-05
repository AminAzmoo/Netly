import { ReactNode } from 'react'

interface DotGridBackgroundProps {
  children: ReactNode
}

export default function DotGridBackground({ children }: DotGridBackgroundProps) {
  return (
    <div className="app-background">
      {children}
    </div>
  )
}
