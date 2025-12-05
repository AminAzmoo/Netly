export type ProcessStepState = 'pending' | 'running' | 'done' | 'error'

export interface ProcessStep {
  id: string
  label: string
  icon?: string
  state: ProcessStepState
}

export type DeviceRole = 'Entry' | 'Exit' | 'Hybrid' | 'Internal'
export type DeviceStatus = 'Pending' | 'Installing' | 'Online' | 'Degraded' | 'Offline'

export interface Device {
  id: string
  name: string
  role: DeviceRole
  ip: string
  location: string
  status: DeviceStatus
  cpu: number
  ram: number
  lastAction?: string
  lastActionTime?: string
  processSteps?: ProcessStep[]
}

export type TunnelType = 'Single-hop' | 'Multi-hop'
export type TunnelStatus = 'Queued' | 'Planning' | 'Configuring' | 'Live' | 'Error'

export interface Tunnel {
  id: string
  name: string
  path: string
  type: TunnelType
  status: TunnelStatus
  latency: number
  lastAction?: string
  lastActionTime?: string
  processSteps?: ProcessStep[]
}

export type ServiceProtocol = 'HTTP' | 'HTTPS' | 'TCP' | 'UDP'
export type ServiceStatus = 'Queued' | 'Configuring' | 'Ready' | 'Error'

export interface Service {
  id: string
  name: string
  protocol: ServiceProtocol
  entryNode: string
  exitNode: string
  users: number
  traffic: string
  status: ServiceStatus
  lastAction?: string
  lastActionTime?: string
  processSteps?: ProcessStep[]
}

export type EventSeverity = 'INFO' | 'WARN' | 'ERROR'

export interface TimelineEvent {
  id: string
  time: string
  title: string
  description: string
  severity: EventSeverity
  entityType?: string
  entityId?: string
}
