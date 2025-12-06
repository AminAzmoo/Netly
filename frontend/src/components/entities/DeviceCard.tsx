import React, { useMemo } from 'react'
import { Device, ProcessStep } from '../../types'
import CardShell from '../common/CardShell'
import StatusBadge from '../common/StatusBadge'
import VerticalProcessTimeline from '../common/VerticalProcessTimeline'
import { 
  Server, Cpu, CircuitBoard, MapPin, Activity, 
  Trash2, RefreshCw, Download, Terminal, Edit, LucideIcon 
} from 'lucide-react'
// standard utility for cleaner class merging (highly recommended)
import { clsx, type ClassValue } from 'clsx' 
import { twMerge } from 'tailwind-merge'

// Utility to merge tailwind classes safely
function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

// ----------------------------------------------------------------------
// Interfaces
// ----------------------------------------------------------------------

interface ExtendedDevice extends Device {
  agentStatus?: 'installed' | 'not_installed' | 'error' | string;
}

interface DeviceCardProps {
  device: ExtendedDevice
  onCleanup?: () => void
  onDelete?: () => void
  onInstallAgent?: () => void
  onShowInstallCommand?: () => void
  onEdit?: () => void
  isProcessing?: boolean
  processSteps?: ProcessStep[]
  processType?: 'cleanup' | 'delete' | 'install-agent'
  lastCleanupAt?: Date
}

// ----------------------------------------------------------------------
// Sub-Components
// ----------------------------------------------------------------------

interface ActionButtonProps {
  onClick?: () => void
  disabled?: boolean
  isActive?: boolean
  icon: LucideIcon
  label: string
  className?: string // Allow overriding base styles
  activeClass?: string
  hoverClass?: string
}

function ActionButton({
  onClick,
  disabled = false,
  isActive = false,
  icon: Icon,
  label,
  className,
  activeClass,
  hoverClass
}: ActionButtonProps) {
  // Logic: Disabled if global disable is true, UNLESS this specific button is the active one
  const isDisabled = disabled && !isActive

  const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation()
    if (!isDisabled && onClick) onClick()
  }

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={isDisabled}
      title={label}
      aria-label={label}
      className={cn(
        "group transition-colors p-2 rounded-md", // Base styles
        isActive ? activeClass : "text-gray-400",
        isDisabled ? "opacity-50 cursor-not-allowed" : "hover:bg-white/5",
        className
      )}
    >
      <Icon 
        size={16} 
        className={cn(
          "transition-colors",
          isActive ? "text-current" : hoverClass
        )} 
      />
    </button>
  )
}

function InfoRow({ label, value }: { label: string; value?: string }) {
  if (!value) return null
  return (
    <div className="flex items-center text-xs gap-2">
      <span className="text-gray-500 font-medium">{label}</span>
      <span className="text-gray-300">{value}</span>
    </div>
  )
}

function ResourceBar({ label, value, icon: Icon }: { label: string; value: number; icon: LucideIcon }) {
  // Color coding based on load
  const loadColor = value > 90 ? 'bg-red-500' : value > 70 ? 'bg-yellow-500' : 'bg-green-500';
  
  return (
    <div className="w-full">
      <div className="flex justify-between items-center mb-1">
        <div className="flex items-center gap-1.5 text-xs text-gray-400">
          <Icon size={12} />
          <span>{label}</span>
        </div>
        <span className="text-xs font-mono">{value}%</span>
      </div>
      <div className="h-1.5 w-full bg-gray-700/50 rounded-full overflow-hidden">
        <div
          className={cn("h-full transition-all duration-500 ease-out", loadColor)}
          style={{ width: `${value}%` }}
        />
      </div>
    </div>
  )
}

// ----------------------------------------------------------------------
// Main Component
// ----------------------------------------------------------------------

export default function DeviceCard({
  device,
  onCleanup,
  onDelete,
  onInstallAgent,
  onShowInstallCommand,
  onEdit,
  isProcessing = false,
  processSteps,
  processType,
  lastCleanupAt
}: DeviceCardProps) {

  // Derived state
  const agentStatus = device.agentStatus || 'not_installed'
  const isOnline = device.status === 'Online'

  const showInstallAgent = useMemo(() => {
    return (
      (agentStatus === 'not_installed' || agentStatus === 'error') &&
      !isOnline &&
      device.status !== 'Installing'
    )
  }, [agentStatus, isOnline, device.status])

  const showManualInstall = agentStatus === 'error' || agentStatus === 'not_installed'

  const statusVariant = useMemo(() => {
    switch (device.status) {
      case 'Online': return 'neonA' // Assuming this maps to green/cyan
      case 'Degraded': return 'warn'
      case 'Offline': return 'error'
      default: return 'default'
    }
  }, [device.status])

  // Handle Date Hydration issues if using Next.js/SSR
  const formattedCleanupTime = useMemo(() => {
    if (!lastCleanupAt) return null;
    return new Intl.DateTimeFormat('en-US', { 
      hour: 'numeric', minute: 'numeric', second: 'numeric' 
    }).format(lastCleanupAt);
  }, [lastCleanupAt]);

  return (
    <CardShell hover className="relative flex flex-col gap-4 p-4 border border-gray-800 bg-gray-900/50">
      
      {/* Header Row */}
      <div className="flex justify-between items-start">
        <div className="flex flex-col gap-3 w-full">
          
          {/* Title & Status */}
          <div className="flex items-center justify-between w-full">
            <div className="flex items-center gap-3">
              <div className="p-2 bg-gray-800 rounded-lg border border-gray-700">
                <Server size={20} className="text-neon-a" />
              </div>
              <div>
                <h3 className="font-semibold text-lg leading-tight text-white">{device.name}</h3>
                <div className="flex gap-2 mt-1">
                  <StatusBadge status={device.role} variant="default" size="sm" />
                  <StatusBadge status={device.status} variant={statusVariant} size="sm" />
                </div>
              </div>
            </div>

            {/* Actions Toolbar */}
            <div className="flex gap-1">
              {showInstallAgent && (
                <ActionButton
                  onClick={onInstallAgent}
                  disabled={isProcessing}
                  isActive={isProcessing && processType === 'install-agent'}
                  activeClass="text-green-400 bg-green-400/10"
                  icon={Download}
                  label="Auto Install Agent"
                />
              )}

              {showManualInstall && (
                <ActionButton
                  onClick={onShowInstallCommand}
                  disabled={isProcessing}
                  icon={Terminal}
                  label="Manual Install Command"
                  hoverClass="group-hover:text-cyan-400"
                />
              )}

              <ActionButton
                onClick={onCleanup}
                disabled={isProcessing}
                isActive={isProcessing && processType === 'cleanup'}
                icon={RefreshCw}
                label="Cleanup Device"
                activeClass="animate-spin text-neon-a"
                hoverClass="group-hover:text-neon-a"
              />

              <ActionButton
                onClick={onEdit}
                disabled={isProcessing}
                icon={Edit}
                label="Edit Device"
                hoverClass="group-hover:text-blue-400"
              />

              <ActionButton
                onClick={onDelete}
                disabled={isProcessing}
                isActive={isProcessing && processType === 'delete'}
                icon={Trash2}
                label="Delete Device"
                activeClass="text-red-500 bg-red-500/10"
                hoverClass="group-hover:text-red-500"
              />
            </div>
          </div>

          <hr className="border-gray-800" />

          {/* Device Meta Grid */}
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div className="space-y-2">
              <InfoRow label="IP Address" value={device.ip} />
              <div className="flex items-center text-xs gap-2">
                 <span className="text-gray-500 font-medium">Location</span>
                 <div className="flex items-center gap-2 text-gray-300">
                    {device.flagCode ? (
                      <img
                        loading="lazy"
                        src={`https://flagcdn.com/20x15/${device.flagCode}.png`}
                        alt="" // Decorative, text is next to it
                        className="w-5 h-[15px] object-cover rounded-sm shadow-sm"
                      />
                    ) : <MapPin size={14} />}
                    <span>{device.location}</span>
                 </div>
              </div>
            </div>

            {/* Resources (Only if Online) */}
            <div className="space-y-2">
              {isOnline ? (
                <>
                  <ResourceBar label="CPU Usage" value={device.cpu} icon={Cpu} />
                  <ResourceBar label="Memory" value={device.ram} icon={CircuitBoard} />
                </>
              ) : (
                <div className="h-full flex items-center justify-center text-xs text-gray-600 italic border border-dashed border-gray-800 rounded">
                  Metrics unavailable (Offline)
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Footer / Timestamps */}
      {(device.lastAction || formattedCleanupTime) && (
        <div className="flex justify-between items-center text-[10px] text-gray-500 pt-2 border-t border-gray-800/50">
           {device.lastAction && (
             <div className="flex items-center gap-1.5">
                <Activity size={12} />
                <span>Last: <span className="text-gray-400">{device.lastAction}</span> ({device.lastActionTime})</span>
             </div>
           )}
           
           {formattedCleanupTime && (
             <div className="flex items-center gap-1.5 ml-auto">
                <RefreshCw size={10} />
                <span>Cleaned: {formattedCleanupTime}</span>
             </div>
           )}
        </div>
      )}

      {/* Progress Timeline Overlay or Section */}
      {isProcessing && processSteps && (
        <div className="mt-2 pt-2 border-t border-gray-800">
          <VerticalProcessTimeline
            steps={processSteps}
            variant="horizontal"
          />
        </div>
      )}

    </CardShell>
  )
}