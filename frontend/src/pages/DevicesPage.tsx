import { useState, useCallback, useEffect, useRef } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import PageShell from '../components/layout/PageShell'
import DeviceCard from '../components/entities/DeviceCard'
import DevicesTable from '../components/entities/DevicesTable'
import ViewToggle from '../components/common/ViewToggle'
import AddDeviceModal from '../components/devices/AddDeviceModal'
import InstallCommandModal from '../components/InstallCommandModal'
import Toast from '../components/common/Toast'
import { useToast } from '../hooks/useToast'
import { api } from '../lib/api'
import { Device, ProcessStep } from '../types'

// Define steps templates
const CLEANUP_STEPS: ProcessStep[] = [
  { id: '1', label: 'Queued', state: 'pending', icon: 'ListStart' },
  { id: '2', label: 'Collecting state', state: 'pending', icon: 'Activity' },
  { id: '3', label: 'Clearing stale tunnels', state: 'pending', icon: 'Network' },
  { id: '4', label: 'Flushing logs/cache', state: 'pending', icon: 'HardDriveDownload' },
  { id: '5', label: 'Done', state: 'pending', icon: 'CheckCircle' },
]

const DELETE_STEPS: ProcessStep[] = [
  { id: '1', label: 'Queued', state: 'pending', icon: 'ListStart' },
  { id: '2', label: 'Draining traffic', state: 'pending', icon: 'Activity' },
  { id: '3', label: 'Deregistering node', state: 'pending', icon: 'ServerCog' },
  { id: '4', label: 'Removing from inventory', state: 'pending', icon: 'HardDriveDownload' },
  { id: '5', label: 'Done', state: 'pending', icon: 'CheckCircle' },
]

const INSTALL_AGENT_STEPS: ProcessStep[] = [
  { id: '1', label: 'Connecting to server', state: 'pending', icon: 'ListStart' },
  { id: '2', label: 'Installing dependencies', state: 'pending', icon: 'FileCode' },
  { id: '3', label: 'Deploying agent', state: 'pending', icon: 'Upload' },
  { id: '4', label: 'Starting service', state: 'pending', icon: 'Activity' },
  { id: '5', label: 'Online', state: 'pending', icon: 'CheckCircle' },
]

type ProcessType = 'cleanup' | 'delete' | 'install-agent'
type AgentStatus = 'not_installed' | 'installing' | 'online' | 'error'

export interface DeviceWithState extends Device {
  lastCleanupAt?: Date
  agentStatus?: AgentStatus
  currentProcess?: ProcessType | null
  currentStepIndex?: number
}

interface ActiveProcess {
  type: ProcessType
  stepIndex: number
  agentStatus?: AgentStatus
}

export default function DevicesPage() {
  const queryClient = useQueryClient()
  const { toasts, showToast, removeToast } = useToast()
  const [view, setView] = useState<'card' | 'table'>('card')
  const [processes, setProcesses] = useState<Record<string, ActiveProcess>>({})
  const [agentStatuses, setAgentStatuses] = useState<Record<string, AgentStatus>>({})
  const [isAddDeviceOpen, setIsAddDeviceOpen] = useState(false)
  const [commandModalNodeId, setCommandModalNodeId] = useState<string | null>(null)
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)
  const intervalsRef = useRef<Record<string, number>>({})

  const { data: rawDevices = [], isLoading, refetch: refetchDevices } = useQuery({
    queryKey: ['nodes'],
    queryFn: () => api.getNodes(),
  })

  const devices: DeviceWithState[] = rawDevices.map((node: any) => {
    const nodeId = String(node.id)
    const process = processes[nodeId]
    const backendStatus = node.status?.toLowerCase() || 'pending'
    const agentStatus = agentStatuses[nodeId] || process?.agentStatus || (backendStatus === 'online' ? 'online' : 'not_installed')
    
    let deviceStatus = 'Offline'
    if (process?.type === 'install-agent') {
      deviceStatus = 'Installing'
    } else if (backendStatus === 'online') {
      deviceStatus = 'Online'
    } else if (backendStatus === 'installing') {
      deviceStatus = 'Installing'
    } else if (backendStatus === 'error') {
      deviceStatus = 'Error'
    } else if (backendStatus === 'pending') {
      deviceStatus = 'Pending'
    }
    
    return {
      id: nodeId,
      name: node.name,
      role: node.role,
      ip: node.ip,
      location: node.geo_data?.country || 'Unknown',
      status: deviceStatus,
      cpu: Math.round(node.stats?.cpu_usage || 0),
      ram: Math.round(node.stats?.ram_usage || 0),
      lastAction: process?.type === 'install-agent' ? 'Installing agent' : (agentStatus === 'online' ? 'Agent running' : 'Waiting'),
      lastActionTime: new Date(node.updated_at).toLocaleString(),
      agentStatus,
      currentProcess: process?.type || null,
      currentStepIndex: process?.stepIndex,
      flagCode: node.geo_data?.flag || null,
    }
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteNode(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['nodes'] })
    },
  })

  const cleanupMutation = useMutation({
    mutationFn: (data: { node_id: number; mode: 'soft' | 'hard' }) => api.cleanupNode(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['nodes'] })
    },
  })

  useEffect(() => {
    return () => {
      Object.values(intervalsRef.current).forEach(clearInterval)
    }
  }, [])

  const getStepsWithState = (template: ProcessStep[], currentIndex: number): ProcessStep[] => {
    return template.map((step, index) => {
      let state: ProcessStep['state'] = 'pending'
      if (index < currentIndex) state = 'done'
      else if (index === currentIndex) state = 'running'
      return { ...step, state }
    })
  }

  const runProcess = useCallback((deviceId: string, type: ProcessType, stepsTemplate: ProcessStep[]) => {
    if (intervalsRef.current[deviceId]) {
      clearInterval(intervalsRef.current[deviceId])
      delete intervalsRef.current[deviceId]
    }

    const initialAgentStatus = type === 'install-agent' ? 'installing' : undefined
    let currentStep = 0
    const totalSteps = stepsTemplate.length
    
    console.log(`[${type}] Starting process for device ${deviceId}, total steps: ${totalSteps}`)
    setProcesses(prev => ({ ...prev, [deviceId]: { type, stepIndex: currentStep, agentStatus: initialAgentStatus } }))

    const interval = setInterval(() => {
      currentStep++
      console.log(`[${type}] Device ${deviceId} - Step ${currentStep}/${totalSteps}`)
      
      setProcesses(prev => {
        if (!prev[deviceId]) return prev
        return { ...prev, [deviceId]: { ...prev[deviceId], stepIndex: currentStep } }
      })

      if (currentStep >= totalSteps - 1) {
        console.log(`[${type}] Device ${deviceId} - Process complete`)
        setTimeout(() => {
          clearInterval(interval)
          delete intervalsRef.current[deviceId]
          
          if (type === 'install-agent') {
            setAgentStatuses(prev => ({ ...prev, [deviceId]: 'online' }))
          } else if (type === 'cleanup') {
            setAgentStatuses(prev => ({ ...prev, [deviceId]: 'not_installed' }))
          }
          
          setProcesses(prev => {
            const next = { ...prev }
            delete next[deviceId]
            return next
          })
        }, 1000)
      }
    }, 1500)

    intervalsRef.current[deviceId] = interval
  }, [])

  const handleCleanup = (deviceId: string) => {
    if (processes[deviceId]) return
    runProcess(deviceId, 'cleanup', CLEANUP_STEPS)
    cleanupMutation.mutate({ node_id: Number(deviceId), mode: 'soft' }, {
      onError: () => {
        // Even if backend fails, we still complete the UI process
        console.log('Cleanup backend failed, but UI process continues')
      }
    })
  }

  const handleDelete = (deviceId: string) => {
    if (processes[deviceId]) return
    setDeleteConfirmId(deviceId)
  }

  const confirmDelete = (deviceId: string) => {
    runProcess(deviceId, 'delete', DELETE_STEPS)
    deleteMutation.mutate(deviceId, {
      onSuccess: () => showToast('Device deleted successfully', 'success'),
      onError: () => showToast('Failed to delete device', 'error')
    })
    setDeleteConfirmId(null)
  }

  const handleInstallAgent = async (deviceId: string) => {
    const device = devices.find(d => d.id === deviceId)
    if (!device || device.currentProcess) return
    if (device.agentStatus === 'online') return
    
    setProcesses(prev => ({ ...prev, [deviceId]: { type: 'install-agent', stepIndex: 0, agentStatus: 'installing' } }))
    
    try {
      const response: any = await api.installAgent(deviceId)
      const taskId = response.task_id
      
      let pollCount = 0
      const pollInterval = setInterval(async () => {
        pollCount++
        try {
          const task: any = await api.getTaskStatus(taskId)
          
          const stepMap: Record<string, number> = {
            'pending': 0,
            'running': 1,
            'completed': 4,
            'failed': 0
          }
          const stepIndex = stepMap[task.status] || 1
          
          if (task.status === 'completed') {
            clearInterval(pollInterval)
            delete intervalsRef.current[deviceId]
            setProcesses(prev => ({ ...prev, [deviceId]: { type: 'install-agent', stepIndex: 4, agentStatus: 'online' } }))
            setTimeout(() => {
              setAgentStatuses(prev => ({ ...prev, [deviceId]: 'online' }))
              setProcesses(prev => {
                const next = { ...prev }
                delete next[deviceId]
                return next
              })
              refetchDevices()
              showToast('Agent installed successfully', 'success')
            }, 1000)
          } else if (task.status === 'failed') {
            clearInterval(pollInterval)
            delete intervalsRef.current[deviceId]
            setAgentStatuses(prev => ({ ...prev, [deviceId]: 'error' }))
            setProcesses(prev => {
              const next = { ...prev }
              delete next[deviceId]
              return next
            })
            showToast(task.error || 'Agent installation failed', 'error')
          } else if (task.status === 'running') {
            setProcesses(prev => ({ ...prev, [deviceId]: { type: 'install-agent', stepIndex, agentStatus: 'installing' } }))
          }
          
          if (pollCount > 120) {
            clearInterval(pollInterval)
            delete intervalsRef.current[deviceId]
            showToast('Installation timeout', 'error')
          }
        } catch (err) {
          console.error('Poll error:', err)
        }
      }, 2000)
      
      intervalsRef.current[deviceId] = pollInterval
    } catch (err: any) {
      console.error('Install agent failed:', err)
      setAgentStatuses(prev => ({ ...prev, [deviceId]: 'error' }))
      setProcesses(prev => {
        const next = { ...prev }
        delete next[deviceId]
        return next
      })
      showToast(err.message || 'Failed to start agent installation', 'error')
    }
  }

  return (
    <>
    <PageShell
      title="Devices"
      subtitle="Manage and monitor all Netly nodes"
      headerRight={
        <div className="flex gap-3">
          <button className="btn-primary-glow" onClick={() => setIsAddDeviceOpen(true)}>
            Add Device
          </button>
        </div>
      }
    >
      {/* Filters */}
      <div className="filter-bar">
        <div className="filter-group">
          <select className="filter-select">
            <option>All Status</option>
            <option>Online</option>
            <option>Installing</option>
            <option>Degraded</option>
            <option>Offline</option>
          </select>
          <select className="filter-select">
            <option>All Roles</option>
            <option>Entry</option>
            <option>Exit</option>
            <option>Hybrid</option>
            <option>Internal</option>
          </select>
          <input
            type="text"
            placeholder="Search by name, IP, tag..."
            className="filter-input"
          />
        </div>
        <ViewToggle value={view} onChange={setView} />
      </div>

      {/* Content */}
      {view === 'card' ? (
        <div className="content-list">
          {devices.map((device) => {
            const process = processes[device.id]
            let steps: ProcessStep[] | undefined
            if (process) {
              const template = process.type === 'cleanup' ? CLEANUP_STEPS 
                : process.type === 'delete' ? DELETE_STEPS 
                : INSTALL_AGENT_STEPS
              steps = getStepsWithState(template, process.stepIndex)
            }

            return (
              <DeviceCard 
                key={device.id} 
                device={device}
                onCleanup={() => handleCleanup(device.id)}
                onDelete={() => handleDelete(device.id)}
                onInstallAgent={() => handleInstallAgent(device.id)}
                onShowCommand={() => setCommandModalNodeId(device.id)}
                isProcessing={!!process}
                processSteps={steps}
                processType={process?.type}
                lastCleanupAt={device.lastCleanupAt}
              />
            )
          })}
        </div>
      ) : (
        <DevicesTable 
          devices={devices}
          processes={processes}
          onCleanup={handleCleanup}
          onDelete={handleDelete}
          onInstallAgent={handleInstallAgent}
          onShowCommand={(nodeId) => setCommandModalNodeId(nodeId)}
          getStepsWithState={getStepsWithState}
          CLEANUP_STEPS={CLEANUP_STEPS}
          DELETE_STEPS={DELETE_STEPS}
          INSTALL_AGENT_STEPS={INSTALL_AGENT_STEPS}
        />
      )}
    </PageShell>
    
    <AddDeviceModal
      isOpen={isAddDeviceOpen}
      onClose={() => setIsAddDeviceOpen(false)}
      onSuccess={() => {
        refetchDevices()
        showToast('Device added successfully', 'success')
      }}
    />

    <InstallCommandModal
      isOpen={!!commandModalNodeId}
      onClose={() => setCommandModalNodeId(null)}
      nodeId={commandModalNodeId || ''}
    />

    {deleteConfirmId && (
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div className="absolute inset-0 bg-black/80 backdrop-blur-sm" onClick={() => setDeleteConfirmId(null)} />
        <div className="relative card-shell max-w-md">
          <h3 className="text-xl font-bold text-white mb-4">Delete Device</h3>
          <p className="text-gray-300 mb-6">Are you sure you want to delete this device? This action cannot be undone.</p>
          <div className="flex justify-end gap-3">
            <button onClick={() => setDeleteConfirmId(null)} className="px-4 py-2 rounded-lg border border-gray-600 text-gray-300 hover:bg-white/5">
              Cancel
            </button>
            <button onClick={() => confirmDelete(deleteConfirmId)} className="px-4 py-2 rounded-lg bg-red-500/20 border border-red-500 text-red-500 hover:bg-red-500/30">
              Delete
            </button>
          </div>
        </div>
      </div>
    )}

    {toasts.map(toast => (
      <Toast key={toast.id} message={toast.message} type={toast.type} onClose={() => removeToast(toast.id)} />
    ))}
    </>
  )
}
