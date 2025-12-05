import { Device, ProcessStep } from '../../types'
import CardShell from '../common/CardShell'
import StatusBadge from '../common/StatusBadge'
import VerticalProcessTimeline from '../common/VerticalProcessTimeline'
import { Server, Cpu, CircuitBoard, MapPin, Activity, Trash2, RefreshCw, Download } from 'lucide-react'

interface DeviceCardProps {
  device: Device
  onCleanup?: () => void
  onDelete?: () => void
  onInstallAgent?: () => void
  isProcessing?: boolean
  processSteps?: ProcessStep[]
  processType?: 'cleanup' | 'delete' | 'install-agent'
  lastCleanupAt?: Date
}

export default function DeviceCard({ 
  device, 
  onCleanup, 
  onDelete,
  onInstallAgent, 
  isProcessing, 
  processSteps,
  processType,
  lastCleanupAt
}: DeviceCardProps) {
  const agentStatus = (device as any).agentStatus || 'not_installed'
  const showInstallAgent = (agentStatus === 'not_installed' || agentStatus === 'error') && device.status !== 'Online' && device.status !== 'Installing'
  const statusVariant =
    device.status === 'Online'
      ? 'neonA'
      : device.status === 'Degraded'
      ? 'warn'
      : device.status === 'Offline'
      ? 'error'
      : 'default'

  return (
    <CardShell hover className="card-relative overflow-hidden">
      <div className="entity-card-header relative z-10">
        <div className="entity-header-left w-full">
          <div className="flex justify-between items-start mb-4">
            <div className="entity-title-row">
              <Server size={20} className="text-neon-a icon-mr-2" />
              <h3 className="entity-title">{device.name}</h3>
              <StatusBadge status={device.role} variant="default" />
              <StatusBadge status={device.status} variant={statusVariant} />
            </div>
            
            <div className="flex items-center gap-2">
              {showInstallAgent && (
                <button 
                  onClick={(e) => { e.stopPropagation(); onInstallAgent?.() }}
                  disabled={isProcessing}
                  className="p-2 hover:bg-green-500/20 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed group"
                  title="Install Agent"
                >
                  <Download size={16} className={`text-gray-400 group-hover:text-green-500 ${isProcessing && processType === 'install-agent' ? 'text-green-500' : ''}`} />
                </button>
              )}
              <button 
                onClick={(e) => { e.stopPropagation(); onCleanup?.() }}
                disabled={isProcessing}
                className="p-2 hover:bg-white/10 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed group"
                title="Cleanup Device"
              >
                <RefreshCw size={16} className={`text-gray-400 group-hover:text-neon-a ${isProcessing && processType === 'cleanup' ? 'animate-spin text-neon-a' : ''}`} />
              </button>
              <button 
                onClick={(e) => { e.stopPropagation(); onDelete?.() }}
                disabled={isProcessing}
                className="p-2 hover:bg-red-500/20 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed group"
                title="Delete Device"
              >
                <Trash2 size={16} className={`text-gray-400 group-hover:text-red-500 ${isProcessing && processType === 'delete' ? 'text-red-500' : ''}`} />
              </button>
            </div>
          </div>

          <div className="entity-info-container">
            <div className="entity-info-row">
              <span className="entity-info-label">IP:</span>
              <span>{device.ip}</span>
            </div>
            <div className="entity-info-row">
              <MapPin size={14} className="text-muted icon-mr-1" />
              <span className="entity-info-label">Location:</span>
              <span>{device.location}</span>
            </div>
          </div>

          {device.status === 'Online' && (
            <div className="entity-stats-container">
              <div>
                <div className="entity-stat-header">
                  <div className="flex-center gap-1">
                    <Cpu size={14} className="text-muted" />
                    <span className="entity-info-label">CPU</span>
                  </div>
                  <span className="entity-stat-value">{device.cpu}%</span>
                </div>
                <div className="entity-stat-track">
                  <div
                    className="entity-stat-bar"
                    style={{ width: `${device.cpu}%` }}
                  ></div>
                </div>
              </div>
              <div>
                <div className="entity-stat-header">
                  <div className="flex-center gap-1">
                    <CircuitBoard size={14} className="text-muted" />
                    <span className="entity-info-label">RAM</span>
                  </div>
                  <span className="entity-stat-value">{device.ram}%</span>
                </div>
                <div className="entity-stat-track">
                  <div
                    className="entity-stat-bar"
                    style={{ width: `${device.ram}%` }}
                  ></div>
                </div>
              </div>
            </div>
          )}

          <div className="flex justify-between items-center mt-4">
            {device.lastAction && (
              <div className="entity-last-action flex-center">
                <Activity size={14} className="text-muted icon-mr-2" />
                <span>
                  Last action: <span className="text-muted">{device.lastAction}</span> {device.lastActionTime}
                </span>
              </div>
            )}
            
            {lastCleanupAt && (
              <div className="text-xs text-gray-500 flex items-center gap-1">
                <RefreshCw size={10} />
                Last cleanup: {lastCleanupAt.toLocaleTimeString()}
              </div>
            )}
          </div>
        </div>
      </div>

      {isProcessing && processSteps && (
        <div className="mt-4 animate-fade-in">
          <VerticalProcessTimeline 
            steps={processSteps} 
            variant="horizontal"
          />
        </div>
      )}
    </CardShell>
  )
}
