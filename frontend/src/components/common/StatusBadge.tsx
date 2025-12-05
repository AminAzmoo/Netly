import { CheckCircle, AlertTriangle, XCircle, Info } from 'lucide-react'

export interface StatusBadgeProps {
  status: string
  variant?: 'default' | 'neonA' | 'neonB' | 'warn' | 'error'
  size?: 'sm' | 'md'
}

export default function StatusBadge({ status, variant = 'default', size = 'md' }: StatusBadgeProps) {
  const variants = {
    default: 'status-badge-default',
    neonA: 'status-badge-neon-a',
    neonB: 'status-badge-neon-b',
    warn: 'status-badge-warn',
    error: 'status-badge-error',
  }

  const sizes = {
    sm: 'status-badge-sm',
    md: '',
  }

  const iconSize = size === 'sm' ? 10 : 12

  const getIcon = () => {
    switch (variant) {
      case 'neonA':
        return <CheckCircle size={iconSize} className="icon-mr-1" />
      case 'warn':
        return <AlertTriangle size={iconSize} className="icon-mr-1" />
      case 'error':
      case 'neonB':
        return <XCircle size={iconSize} className="icon-mr-1" />
      default:
        return <Info size={iconSize} className="icon-mr-1" />
    }
  }

  return (
    <span
      className={`status-badge ${variants[variant]} ${sizes[size]}`.trim()}
    >
      {getIcon()}
      {status}
    </span>
  )
}
