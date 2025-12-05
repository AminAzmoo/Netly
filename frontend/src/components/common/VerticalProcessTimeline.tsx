import { ProcessStep } from '../../types'
import {
  Check, X, Loader2, Circle,
  ListStart, Terminal, Download, HardDriveDownload, ServerCog, Activity,
  Map, Lock, Route, Thermometer, Zap,
  Network, FileCode, Upload, Wifi, CheckCircle
} from 'lucide-react'

const iconMap: Record<string, React.ComponentType<any>> = {
  ListStart, Terminal, Download, HardDriveDownload, ServerCog, Activity,
  Map, Lock, Route, Thermometer, Zap,
  Network, FileCode, Upload, Wifi, CheckCircle
}

interface VerticalProcessTimelineProps {
  steps: ProcessStep[]
  variant?: 'default' | 'card-edge' | 'row-mini' | 'horizontal'
  className?: string
  disableSeparator?: boolean
}

export default function VerticalProcessTimeline({ steps, variant = 'default', className = '', disableSeparator = false }: VerticalProcessTimelineProps) {
  if (variant === 'horizontal') {
    const content = (
      <div className="flex items-center justify-between relative w-full px-1">
        {/* Connecting Line */}
        <div className="absolute top-4 left-0 right-0 h-0.5 bg-white/10 -z-0" />
        
        {steps.map((step) => {
          const IconComponent = step.icon ? iconMap[step.icon] : null
          const isActive = step.state === 'running'
          const isDone = step.state === 'done'
          const isError = step.state === 'error'
          
          return (
            <div key={step.id} className="flex flex-col items-center relative z-10 group min-w-[40px]">
              <div className={`w-8 h-8 rounded-full border flex items-center justify-center mb-2 transition-all duration-300 bg-black ${
                isActive 
                  ? 'border-neon-a shadow-[0_0_15px_rgba(0,229,255,0.5)] scale-110' 
                  : isDone
                  ? 'border-neon-a/50 bg-neon-a/10'
                  : isError
                  ? 'border-neon-b shadow-[0_0_15px_rgba(196,0,255,0.5)]'
                  : 'border-gray-700'
              }`}>
                 {isActive ? (
                   <Loader2 size={14} className="text-neon-a animate-spin" />
                 ) : isError ? (
                   <X size={14} className="text-neon-b" />
                 ) : isDone ? (
                   <Check size={14} className="text-neon-a" />
                 ) : IconComponent ? (
                   <IconComponent size={14} className="text-gray-600" />
                 ) : (
                   <Circle size={8} className="text-gray-600" />
                 )}
              </div>
              
              <span className={`text-[9px] font-medium uppercase tracking-wider text-center max-w-[60px] leading-tight ${
                isActive ? 'text-neon-a' :
                isDone ? 'text-gray-500' :
                isError ? 'text-neon-b' :
                'text-gray-700'
              }`}>
                {step.label}
              </span>
            </div>
          )
        })}
      </div>
    )

    if (disableSeparator) {
      return <div className={className}>{content}</div>
    }

    return (
      <div className={`mt-4 border-t border-white/10 pt-4 ${className}`}>
        {content}
      </div>
    )
  }

  if (variant === 'row-mini') {
    return (
      <div className={`flex items-center gap-3 text-xs ${className}`}>
         {steps.map((step) => (
            <div key={step.id} className="flex items-center gap-1">
                <div className={`w-2 h-2 rounded-full ${
                    step.state === 'running' ? 'bg-neon-a animate-pulse shadow-[0_0_8px_rgba(124,58,237,0.6)]' :
                    step.state === 'done' ? 'bg-neon-a' :
                    step.state === 'error' ? 'bg-neon-b' :
                    'bg-gray-700 border border-gray-600'
                }`} />
                <span className={`${
                    step.state === 'running' || step.state === 'done' ? 'text-gray-200' : 
                    step.state === 'error' ? 'text-neon-b' : 'text-gray-600'
                }`}>
                    {step.label}
                </span>
            </div>
         ))}
      </div>
    )
  }

  // Default Vertical Timeline
  return (
    <div className={`timeline-container ${className}`}>
      <div className="timeline-line">
        <div className="timeline-glow-sweep"></div>
      </div>
      
      <div className="timeline-steps-wrapper">
        {steps.map((step) => {
          const IconComponent = step.icon ? iconMap[step.icon] : null
          
          return (
            <div key={step.id} className="timeline-step-item">
              <div className="timeline-step-inner">
                {/* Step dot */}
                <div
                  className={`timeline-step-dot ${
                    step.state === 'running'
                      ? 'timeline-step-dot-running'
                      : step.state === 'done'
                      ? 'timeline-step-dot-done'
                      : step.state === 'error'
                      ? 'timeline-step-dot-error'
                      : 'timeline-step-dot-pending'
                  }`}
                >
                  {step.state === 'done' && (
                    <Check size={12} className="text-neon-a" />
                  )}
                  {step.state === 'error' && (
                    <X size={12} className="text-neon-b" />
                  )}
                  {step.state === 'running' && (
                    <Loader2 size={12} className="text-neon-a animate-spin" />
                  )}
                  {step.state === 'pending' && (
                    <Circle size={8} className="text-muted" fill="currentColor" />
                  )}
                </div>

                {/* Step Icon */}
                {IconComponent && (
                  <IconComponent 
                    size={14} 
                    className={`mr-2 ${
                      step.state === 'running' || step.state === 'done' 
                        ? 'text-neon-a' 
                        : 'text-muted'
                    }`} 
                  />
                )}

                {/* Label */}
                <span
                  className={`timeline-step-label ${
                    step.state === 'running'
                      ? 'timeline-step-label-running'
                      : step.state === 'done'
                      ? 'timeline-step-label-done'
                      : step.state === 'error'
                      ? 'timeline-step-label-error'
                      : 'timeline-step-label-pending'
                  }`}
                >
                  {step.label}
                </span>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
