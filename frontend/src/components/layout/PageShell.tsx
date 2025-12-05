import { ReactNode } from 'react'
import { Hash } from 'lucide-react'

interface PageShellProps {
  title: string
  subtitle: string
  children: ReactNode
  headerRight?: ReactNode
}

export default function PageShell({ title, subtitle, children, headerRight }: PageShellProps) {
  return (
    <div className="page-shell-container">
      <div className="page-header">
        <div>
          <h1 className="page-title">
            <Hash size={32} className="text-neon-a" />
            {title}
          </h1>
          <p className="page-subtitle">
            {subtitle}
          </p>
        </div>
        {headerRight && <div className="page-header-right">{headerRight}</div>}
      </div>
      {children}
    </div>
  )
}
