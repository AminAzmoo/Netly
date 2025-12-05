import { useState } from 'react'
import CardShell from '../common/CardShell'
import VerticalProcessTimeline from '../common/VerticalProcessTimeline'
import { ProcessStep } from '../../types'

interface IntegrationField {
  key: string
  label: string
  value: string
  type?: 'text' | 'password'
}

interface IntegrationState {
  id: string
  name: string
  subtitle: string
  enabled: boolean
  configured: boolean
  expanded: boolean
  processing: boolean
  status: string
  fields: IntegrationField[]
  steps: ProcessStep[]
}

const INITIAL_TIMELINE_STEPS: ProcessStep[] = [
  { id: '1', label: 'Queued', state: 'pending' },
  { id: '2', label: 'Validating', state: 'pending' },
  { id: '3', label: 'Pinging', state: 'pending' },
  { id: '4', label: 'Saving', state: 'pending' },
  { id: '5', label: 'Done', state: 'pending' },
]

const INITIAL_INTEGRATIONS: IntegrationState[] = [
  {
    id: 'prometheus',
    name: 'Prometheus Metrics',
    subtitle: 'Export metrics to Prometheus',
    enabled: false,
    configured: false,
    expanded: false,
    processing: false,
    status: 'Disabled',
    fields: [
      { key: 'endpoint', label: 'Endpoint URL', value: 'http://localhost:9090' },
      { key: 'token', label: 'Auth Token', value: '', type: 'password' },
      { key: 'path', label: 'Metrics Path', value: '/metrics' },
    ],
    steps: INITIAL_TIMELINE_STEPS
  },
  {
    id: 'cloudflare',
    name: 'Cloudflare',
    subtitle: 'Sync DNS and Proxy settings',
    enabled: false,
    configured: false,
    expanded: false,
    processing: false,
    status: 'Disabled',
    fields: [
      { key: 'api_token', label: 'API Token', value: '', type: 'password' },
      { key: 'account_id', label: 'Account ID', value: '' },
      { key: 'zone_id', label: 'Zone ID', value: '' },
      { key: 'domain', label: 'Domain Pattern', value: '*.example.com' },
    ],
    steps: INITIAL_TIMELINE_STEPS
  },
  {
    id: 'firebase',
    name: 'Firebase Auth',
    subtitle: 'External authentication provider',
    enabled: false,
    configured: false,
    expanded: false,
    processing: false,
    status: 'Disabled',
    fields: [
      { key: 'project_id', label: 'Project ID', value: '' },
      { key: 'api_key', label: 'API Key', value: '', type: 'password' },
      { key: 'auth_domain', label: 'Auth Domain', value: '' },
    ],
    steps: INITIAL_TIMELINE_STEPS
  }
]

export default function IntegrationsCard() {
  const [integrations, setIntegrations] = useState<IntegrationState[]>(INITIAL_INTEGRATIONS)

  const toggleIntegration = (id: string) => {
    setIntegrations(prev => prev.map(int => {
      if (int.id !== id) return int
      const newEnabled = !int.enabled
      return {
        ...int,
        enabled: newEnabled,
        status: newEnabled 
          ? (int.configured ? 'Active' : 'Enabled, not configured') 
          : 'Disabled'
      }
    }))
  }

  const toggleExpand = (id: string) => {
    setIntegrations(prev => prev.map(int => ({
      ...int,
      expanded: int.id === id ? !int.expanded : int.expanded
    })))
  }

  const updateField = (id: string, key: string, val: string) => {
    setIntegrations(prev => prev.map(int => {
      if (int.id !== id) return int
      return {
        ...int,
        fields: int.fields.map(f => f.key === key ? { ...f, value: val } : f)
      }
    }))
  }

  const handleTestSave = (id: string) => {
    const integration = integrations.find(i => i.id === id)
    if (!integration || integration.processing) return

    // Start processing
    setIntegrations(prev => prev.map(int => {
      if (int.id !== id) return int
      return {
        ...int,
        processing: true,
        steps: INITIAL_TIMELINE_STEPS.map(s => ({ ...s, state: 'pending' }))
      }
    }))

    let currentStepIndex = 0
    const runStep = () => {
      if (currentStepIndex >= INITIAL_TIMELINE_STEPS.length) {
        // Success
        setIntegrations(prev => prev.map(int => {
            if (int.id !== id) return int
            return {
                ...int,
                processing: false,
                configured: true,
                status: `${int.name}: Connected and saved just now`,
                steps: int.steps.map(s => ({ ...s, state: 'done' }))
            }
        }))
        return
      }

      setIntegrations(prev => prev.map(int => {
        if (int.id !== id) return int
        return {
            ...int,
            steps: int.steps.map((s, idx) => {
                if (idx < currentStepIndex) return { ...s, state: 'done' }
                if (idx === currentStepIndex) return { ...s, state: 'running' }
                return { ...s, state: 'pending' }
            })
        }
      }))

      setTimeout(() => {
        // Error simulation (randomly fail Cloudflare if empty token)
        const currentInt = integrations.find(i => i.id === id)
        if (id === 'cloudflare' && currentStepIndex === 2 && currentInt?.fields.find(f => f.key === 'api_token')?.value === '') {
             setIntegrations(prev => prev.map(int => {
                if (int.id !== id) return int
                return {
                    ...int,
                    processing: false,
                    status: 'Error: Invalid API Token',
                    steps: int.steps.map((s, idx) => {
                         if (idx === currentStepIndex) return { ...s, state: 'error' }
                         if (idx < currentStepIndex) return { ...s, state: 'done' }
                         return { ...s, state: 'pending' }
                    })
                }
            }))
            return
        }

        currentStepIndex++
        if (currentStepIndex < INITIAL_TIMELINE_STEPS.length) {
            runStep()
        } else {
            // Final completion logic repeated here for safety in timeout
            setIntegrations(prev => prev.map(int => {
                if (int.id !== id) return int
                return {
                    ...int,
                    processing: false,
                    configured: true,
                    status: `${int.name}: Connected and saved just now`,
                    steps: int.steps.map(s => ({ ...s, state: 'done' }))
                }
            }))
        }
      }, 800)
    }

    runStep()
  }

  return (
    <CardShell>
      <div className="p-2">
        <h3 className="settings-card-title">Integrations</h3>
        
        <div className="integration-card-list">
          {integrations.map((integration) => (
            <div key={integration.id} className="integration-item">
              {/* Row Header */}
              <div className="integration-header">
                <div className="flex-1">
                    <div className="flex-center gap-3">
                        <h4 className="integration-name">{integration.name}</h4>
                        <span className={`badge ${
                            integration.enabled 
                                ? 'badge-active' 
                                : 'badge-inactive'
                        }`}>
                            {integration.enabled ? 'ON' : 'OFF'}
                        </span>
                    </div>
                    <p className="integration-subtitle">{integration.subtitle}</p>
                    <p className={`integration-status ${integration.status.includes('Error') ? 'integration-status-error' : ''}`}>
                        {integration.status}
                    </p>
                </div>

                <div className="flex-center gap-4">
                   {/* Toggle */}
                   <button 
                     onClick={() => toggleIntegration(integration.id)}
                     className={`toggle-switch ${
                        integration.enabled ? 'toggle-active' : 'toggle-inactive'
                     }`}
                   >
                     <div className={`toggle-dot ${
                        integration.enabled ? 'toggle-dot-active' : 'toggle-dot-inactive'
                     }`} />
                   </button>

                   <button 
                     onClick={() => toggleExpand(integration.id)}
                     className="text-link"
                   >
                     {integration.expanded ? 'Close' : 'Configure'}
                   </button>
                </div>
              </div>

              {/* Drawer */}
              {integration.expanded && (
                <div className="integration-drawer">
                   <div className="drawer-grid">
                      {integration.fields.map(field => (
                          <div key={field.key}>
                              <label className="settings-label">{field.label}</label>
                              <input 
                                type={field.type || 'text'}
                                value={field.value}
                                onChange={(e) => updateField(integration.id, field.key, e.target.value)}
                                className="settings-input"
                              />
                          </div>
                      ))}
                   </div>

                   <div className="flex-between border-top-separator">
                       <div className="flex-1">
                           {integration.processing && (
                               <VerticalProcessTimeline 
                                 steps={integration.steps} 
                                 variant="horizontal" 
                                 disableSeparator
                                 className="pr-4"
                               />
                           )}
                       </div>
                       <button 
                         onClick={() => handleTestSave(integration.id)}
                         disabled={integration.processing}
                         className="settings-btn-secondary"
                       >
                         {integration.processing ? 'Testing...' : 'Test & Save'}
                       </button>
                   </div>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>
    </CardShell>
  )
}
