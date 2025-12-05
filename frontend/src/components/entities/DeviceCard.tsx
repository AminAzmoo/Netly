import React, { useMemo } from 'react'
import { Device, ProcessStep } from '../../types'
import CardShell from '../common/CardShell'
import StatusBadge from '../common/StatusBadge'
import VerticalProcessTimeline from '../common/VerticalProcessTimeline'
import { Server, Cpu, CircuitBoard, MapPin, Activity, Trash2, RefreshCw, Download, Terminal, Edit, LucideIcon } from 'lucide-react'

// Extended interface to handle the optional agentStatus safely
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
// ActionButton - Internal reusable component for action buttons
// ----------------------------------------------------------------------

interface ActionButtonProps {
  onClick?: () => void
  disabled?: boolean
  isActive?: boolean
  icon: LucideIcon
  label: string
  activeColorClass?: string
  hoverColorClass?: string
}

function ActionButton({
  onClick,
  disabled = false,
  isActive = false,
  icon: Icon,
  label,
  activeColorClass = '',
  hoverColorClass = ''
}: ActionButtonProps) {
  // When this button is active (performing action), it should NOT be disabled
  // When another action is processing, this button should be disabled
  const isDisabled = disabled && !isActive

  const handleClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation()
    if (!isDisabled && onClick) {
      onClick()
    }
  }

  // Build button class string
  const buttonClassName = `device-action-button group transition-colors${isActive ? ` ${activeColorClass}` : ''}${isDisabled ? ' opacity-50 cursor-default' : ''}`

  // Build icon class string
  const iconClassName = `text-gray-400 transition-colors${isActive ? ` ${activeColorClass}` : ` ${hoverColorClass}`}`

  return (
    <button
      type="button"
      onClick={handleClick}
      disabled={isDisabled}
      className={buttonClassName}
      title={label}
      aria-label={label}
    >
      <Icon size={16} className={iconClassName} />
    </button>
  )
}

// ----------------------------------------------------------------------
// InfoRow - Display label/value pairs
// ----------------------------------------------------------------------

interface InfoRowProps {
  label: string
  value?: string
}

function InfoRow({ label, value }: InfoRowProps) {
  if (!value) return null
  return (
    <div className="device-info-row">
      <span className="device-info-label">{label}</span>
      <span>{value}</span>
    </div>
  )
}

// ----------------------------------------------------------------------
// ResourceBar - Display resource usage with progress bar
// ----------------------------------------------------------------------

interface ResourceBarProps {
  label: string
  value: number
  icon: LucideIcon
}

function ResourceBar({ label, value, icon: Icon }: ResourceBarProps) {
  return (
    <div>
      <div className="device-stat-header">
        <div className="flex-center gap-1">
          <Icon size={14} className="text-muted" />
          <span className="device-info-label">{label}</span>
        </div>
        <span className="entity-stat-value">{value}%</span>
      </div>
      <div className="device-stat-track">
        <div
          className="device-stat-bar"
          style={{ width: `${value}%` }}
        />
      </div>
    </div>
  )
}

// ----------------------------------------------------------------------
// DeviceCard - Main component
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

  // Determine if "Install Agent" button should be shown
  const showInstallAgent = useMemo(() => {
    return (
      (agentStatus === 'not_installed' || agentStatus === 'error') &&
      !isOnline &&
      device.status !== 'Installing'
    )
  }, [agentStatus, isOnline, device.status])

  // Determine if "Manual Install Command" button should be shown
  const showManualInstall = agentStatus === 'error' || agentStatus === 'not_installed'

  // Status badge variant based on device status
  const statusVariant = useMemo(() => {
    switch (device.status) {
      case 'Online':
        return 'neonA'
      case 'Degraded':
        return 'warn'
      case 'Offline':
        return 'error'
      default:
        return 'default'
    }
  }, [device.status])

  return (
    <CardShell hover className="device-card-container">
      <div className="device-card-header">
        <div className="device-header-left">
          <div className="device-header-row">

            {/* Title Section */}
            <div className="device-title-row">
              <Server size={20} className="text-neon-a icon-mr-2" />
              <h3 className="entity-title">{device.name}</h3>
              <StatusBadge status={device.role} variant="default" />
              <StatusBadge status={device.status} variant={statusVariant} />
            </div>

            {/* Actions Toolbar */}
            <div className="device-actions-container flex gap-2">
              {showInstallAgent && (
                <ActionButton
                  onClick={onInstallAgent}
                  disabled={isProcessing}
                  isActive={isProcessing && processType === 'install-agent'}
                  activeColorClass="text-green-500"
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
                  hoverColorClass="group-hover:text-cyan-500"
                />
              )}

              <ActionButton
                onClick={onCleanup}
                disabled={isProcessing}
                isActive={isProcessing && processType === 'cleanup'}
                icon={RefreshCw}
                label="Cleanup Device"
                activeColorClass="animate-spin text-neon-a"
                hoverColorClass="group-hover:text-neon-a"
              />

              <ActionButton
                onClick={onEdit}
                disabled={isProcessing}
                icon={Edit}
                label="Edit Device"
                hoverColorClass="group-hover:text-blue-500"
              />

              <ActionButton
                onClick={onDelete}
                disabled={isProcessing}
                isActive={isProcessing && processType === 'delete'}
                icon={Trash2}
                label="Delete Device"
                activeColorClass="text-red-500"
                hoverColorClass="group-hover:text-red-500"
              />
            </div>
          </div>

          {/* Device Meta Info */}
          <div className="device-info-container">
            <InfoRow label="IP:" value={device.ip} />
            <div className="device-info-row">
              <MapPin size={14} className="text-muted icon-mr-1" />
              <span className="device-info-label">Location:</span>
              <div className="flex items-center gap-2">
                {device.flagCode && (
                  <img
                    src={`https://flagcdn.com/20x15/${device.flagCode}.png`}
                    alt={`${device.location} flag`}
                    className="device-flag-image"
                  />
                )}
                <span>{device.location}</span>
              </div>
            </div>
          </div>

          {/* Resource Stats (Only if Online) */}
          {isOnline && (
            <div className="device-stats-container">
              <ResourceBar label="CPU" value={device.cpu} icon={Cpu} />
              <ResourceBar label="RAM" value={device.ram} icon={CircuitBoard} />
            </div>
          )}

          {/* Footer / Timestamps */}
          <div className="device-last-action-container">
            {device.lastAction && (
              <div className="device-last-action">
                <Activity size={14} className="text-muted icon-mr-2" />
                <span>
                  Last action: <span className="text-muted">{device.lastAction}</span> {device.lastActionTime}
                </span>
              </div>
            )}

            {lastCleanupAt && (
              <div className="device-cleanup-time">
                <RefreshCw size={10} className="mr-1" />
                Last cleanup: {lastCleanupAt.toLocaleTimeString()}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Progress Timeline */}
      {isProcessing && processSteps && (
        <div className="device-process-container">
          <VerticalProcessTimeline
            steps={processSteps}
            variant="horizontal"
          />
        </div>
      )}

    </CardShell>
  )
}
