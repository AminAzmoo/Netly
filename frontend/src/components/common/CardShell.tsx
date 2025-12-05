import { ReactNode } from 'react'

interface CardShellProps {
  children: ReactNode
  className?: string
  hover?: boolean
}

export default function CardShell({ children, className = '', hover = false }: CardShellProps) {
  return (
    <div
      className={`card-shell ${
        hover ? 'card-shell-hover' : ''
      } ${className}`}
    >
      {children}
    </div>
  )
}
