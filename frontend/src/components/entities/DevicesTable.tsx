import { Device, ProcessStep } from '../../types'
import StatusBadge from '../common/StatusBadge'
import CardShell from '../common/CardShell'
import VerticalProcessTimeline from '../common/VerticalProcessTimeline'
import { Download, RefreshCw, Trash2, Terminal, Edit3 } from 'lucide-react'

interface DevicesTableProps {
  devices: Device[]
  processes: Record<string, any>
  onCleanup: (id: string) => void
  onDelete: (id: string) => void
  onInstallAgent: (id: string) => void
  onShowInstallCommand: (id: string) => void
  onEdit: (device: Device) => void
  getStepsWithState: (template: ProcessStep[], currentIndex: number) => ProcessStep[]
  CLEANUP_STEPS: ProcessStep[]
  DELETE_STEPS: ProcessStep[]
  INSTALL_AGENT_STEPS: ProcessStep[]
}

export default function DevicesTable({ 
  devices, 
  processes, 
  onCleanup, 
  onDelete, 
  onInstallAgent,
  onShowInstallCommand,
  onEdit,
  getStepsWithState,
  CLEANUP_STEPS,
  DELETE_STEPS,
  INSTALL_AGENT_STEPS
}: DevicesTableProps) {
  return (
    <CardShell className="data-table-container">
      <div className="data-table-wrapper">
        <table className="data-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Role</th>
              <th>Status</th>
              <th>IP Address</th>
              <th>Location</th>
              <th>Resources</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {devices.map((device) => {
              const statusVariant =
                device.status === 'Online'
                  ? 'neonA'
                  : device.status === 'Degraded'
                  ? 'warn'
                  : device.status === 'Offline'
                  ? 'error'
                  : 'default'

              const agentStatus = (device as any).agentStatus || 'not_installed'
              const showInstallAgent = agentStatus === 'not_installed' || agentStatus === 'error'
              const process = processes[device.id]
              const isProcessing = !!process
              
              let steps: ProcessStep[] | undefined
              if (process) {
                const template = process.type === 'cleanup' ? CLEANUP_STEPS 
                  : process.type === 'delete' ? DELETE_STEPS 
                  : INSTALL_AGENT_STEPS
                steps = getStepsWithState(template, process.stepIndex)
              }

              return (
                <tr key={device.id}>
                  <td className="data-table-cell-primary">{device.name}</td>
                  <td>
                    <StatusBadge status={device.role} variant="default" />
                  </td>
                  <td>
                    <StatusBadge status={device.status} variant={statusVariant} />
                  </td>
                  <td className="data-table-cell-mono">{device.ip}</td>
                  <td>
                    <div className="flex items-center">
                      {device.flagCode && (
                        <img 
                          src={`https://flagcdn.com/20x15/${device.flagCode}.png`}
                          alt={device.location}
                          className="device-table-flag-image"
                        />
                      )}
                      <span>{device.location}</span>
                    </div>
                  </td>
                  <td>
                    {device.status === 'Online' || device.status === 'Degraded' ? (
                      <div className="data-table-resources">
                        <div className="data-table-resource-row">
                          <span>CPU</span>
                          <span className={device.cpu > 80 ? 'text-neon-b' : ''}>
                            {device.cpu}%
                          </span>
                        </div>
                        <div className="data-table-resource-row">
                          <span>RAM</span>
                          <span className={device.ram > 80 ? 'text-neon-b' : ''}>
                            {device.ram}%
                          </span>
                        </div>
                      </div>
                    ) : (
                      <span className="text-muted">-</span>
                    )}
                  </td>
                  <td>
                    <div className="device-table-actions-container">
                      {showInstallAgent && (
                        <button 
                          onClick={() => onInstallAgent(device.id)}
                          disabled={isProcessing}
                          className="device-table-action-button device-table-action-button-install"
                          title="Auto Install Agent"
                        >
                          <Download size={14} className="text-gray-400 hover:text-green-500" />
                        </button>
                      )}
                      {(agentStatus === 'error' || agentStatus === 'not_installed') && (
                        <button 
                          onClick={() => onShowInstallCommand(device.id)}
                          disabled={isProcessing}
                          className="device-table-action-button device-table-action-button-command"
                          title="Get Install Command"
                        >
                          <Terminal size={14} className="text-gray-400 hover:text-cyan-500" />
                        </button>
                      )}
                      <button 
                        onClick={() => onCleanup(device.id)}
                        disabled={isProcessing}
                        className="device-table-action-button device-table-action-button-cleanup"
                        title="Cleanup"
                      >
                        <RefreshCw size={14} className="text-gray-400" />
                      </button>
                      <button 
                        onClick={() => onEdit(device)}
                        className="device-table-action-button device-table-action-button-edit"
                        title="Edit"
                      >
                        <Edit3 size={14} className="text-gray-400 hover:text-blue-500" />
                      </button>
                      <button 
                        onClick={() => onDelete(device.id)}
                        className="device-table-action-button device-table-action-button-delete"
                        title="Delete"
                      >
                        <Trash2 size={14} className="text-gray-400 hover:text-red-500" />
                      </button>
                      {isProcessing && steps && (
                        <div className="device-table-process-container">
                          <VerticalProcessTimeline steps={steps} variant="horizontal" />
                        </div>
                      )}
                    </div>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </CardShell>
  )
}
