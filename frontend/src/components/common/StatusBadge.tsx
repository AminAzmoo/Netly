import { CheckCircle, AlertTriangle, XCircle, Info } from 'lucide-react'

interface StatusBadgeProps {
  status: string
  variant?: 'default' | 'neonA' | 'neonB' | 'warn' | 'error'
}

export default function StatusBadge({ status, variant = 'default' }: StatusBadgeProps) {
  const variants = {
    default: 'status-badge-default',
    neonA: 'status-badge-neon-a',
    neonB: 'status-badge-neon-b',
    warn: 'status-badge-warn',
    error: 'status-badge-error',
  }

  const getIcon = () => {
    switch (variant) {
      case 'neonA':
        return <CheckCircle size={12} className="icon-mr-1" />
      case 'warn':
        return <AlertTriangle size={12} className="icon-mr-1" />
      case 'error':
      case 'neonB':
        return <XCircle size={12} className="icon-mr-1" />
      default:
        return <Info size={12} className="icon-mr-1" />
    }
  }

  return (
    <span
      className={`status-badge ${variants[variant]}`}
    >
      {getIcon()}
      {status}
    </span>
  )
}
