import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import CardShell from '../common/CardShell'
import VerticalProcessTimeline from '../common/VerticalProcessTimeline'
import { ProcessStep } from '../../types'
import { api } from '../../lib/api'

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
    name: 'Cloudflare Tunnel',
    subtitle: 'Expose your server via Cloudflare Tunnel',
    enabled: false,
    configured: false,
    expanded: false,
    processing: false,
    status: 'Disabled',
    fields: [
      { key: 'email', label: 'Cloudflare Email', value: '' },
      { key: 'global_key', label: 'Global API Key', value: '', type: 'password' },
      { key: 'account_id', label: 'Account ID', value: '' },
      { key: 'tunnel_name', label: 'Tunnel Name', value: 'netly-tunnel' },
      { key: 'public_url', label: 'Public URL (optional)', value: '', type: 'text' },
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
  const queryClient = useQueryClient()
  const [integrations, setIntegrations] = useState<IntegrationState[]>(INITIAL_INTEGRATIONS)
  
  const { data: settings } = useQuery({
    queryKey: ['settings'],
    queryFn: () => api.getSettings()
  })
  
  const updateSettingsMutation = useMutation({
    mutationFn: (data: Record<string, string>) => api.updateSettings(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] })
    }
  })
  
  useEffect(() => {
    if (settings) {
      setIntegrations(prev => prev.map(int => {
        if (int.id === 'cloudflare') {
          const isConfigured = !!(settings.cloudflare_email && settings.cloudflare_global_key && settings.cloudflare_account_id)
          const isEnabled = settings.cloudflare_enabled === 'true'
          return {
            ...int,
            fields: int.fields.map(field => ({
              ...field,
              value: settings[`cloudflare_${field.key}`] || field.value
            })),
            configured: isConfigured,
            enabled: isEnabled,
            status: isEnabled 
              ? (isConfigured ? 'Active' : 'Enabled, not configured') 
              : (isConfigured ? 'Configured' : 'Not configured')
          }
        }
        return int
      }))
    }
  }, [settings])

  const toggleIntegration = async (id: string) => {
    const integration = integrations.find(i => i.id === id)
    if (!integration) return
    
    const newEnabled = !integration.enabled
    
    // Update UI immediately
    setIntegrations(prev => prev.map(int => {
      if (int.id !== id) return int
      return {
        ...int,
        enabled: newEnabled,
        status: newEnabled 
          ? (int.configured ? 'Active' : 'Enabled, not configured') 
          : (int.configured ? 'Configured' : 'Not configured')
      }
    }))
    
    // Save to backend
    if (id === 'cloudflare') {
      try {
        await updateSettingsMutation.mutateAsync({
          cloudflare_enabled: newEnabled.toString()
        })
      } catch (error) {
        // Revert on error
        setIntegrations(prev => prev.map(int => {
          if (int.id !== id) return int
          return {
            ...int,
            enabled: !newEnabled,
            status: 'Error: Failed to update status'
          }
        }))
      }
    }
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

  const handleTestSave = async (id: string) => {
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
    const runStep = async () => {
      if (currentStepIndex >= INITIAL_TIMELINE_STEPS.length) {
        // Save to backend for Cloudflare
        if (id === 'cloudflare') {
          try {
            const settingsData: Record<string, string> = {}
            integration.fields.forEach(field => {
              settingsData[`cloudflare_${field.key}`] = field.value
            })
            await updateSettingsMutation.mutateAsync(settingsData)
          } catch (error) {
            setIntegrations(prev => prev.map(int => {
              if (int.id !== id) return int
              return {
                ...int,
                processing: false,
                status: 'Error: Failed to save settings',
                steps: int.steps.map((s, idx) => 
                  idx === int.steps.length - 1 ? { ...s, state: 'error' } : s
                )
              }
            }))
            return
          }
        }
        
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
        if (id === 'cloudflare' && currentStepIndex === 2 && currentInt?.fields.find(f => f.key === 'token')?.value === '') {
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
            runStep() // Continue to final step
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
