import { useState, useEffect, useRef } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'
import CardShell from '../common/CardShell'
import { ProcessStep } from '../../types'
import VerticalProcessTimeline from '../common/VerticalProcessTimeline'
import { api } from '../../lib/api'

const INITIAL_STEPS: ProcessStep[] = [
  { id: '1', label: 'Queued', state: 'pending' },
  { id: '2', label: 'Validating values', state: 'pending' },
  { id: '3', label: 'Writing config', state: 'pending' },
  { id: '4', label: 'Reloading cache', state: 'pending' },
  { id: '5', label: 'Done', state: 'pending' },
]

export default function GeneralSettingsCard() {
  const [steps, setSteps] = useState<ProcessStep[]>(INITIAL_STEPS)
  const [isProcessing, setIsProcessing] = useState(false)
  const [lastUpdate, setLastUpdate] = useState<string | null>(null)
  const [showTimeline, setShowTimeline] = useState(false)
  const { data: settings } = useQuery({
    queryKey: ['generalSettings'],
    queryFn: () => api.getGeneralSettings(),
  })

  const [formData, setFormData] = useState({
    systemName: '',
    adminEmail: '',
    publicUrl: '',
    environment: 'Production'
  })

  useEffect(() => {
    if (settings) {
      setFormData({
        systemName: settings.systemName || '',
        adminEmail: settings.adminEmail || '',
        publicUrl: settings.publicUrl || '',
        environment: settings.environment || 'Production'
      })
    }
  }, [settings])
  const timeoutsRef = useRef<number[]>([])

  useEffect(() => {
    return () => {
      timeoutsRef.current.forEach(clearTimeout)
    }
  }, [])

  const updateMutation = useMutation({
    mutationFn: (data: typeof formData) => api.updateGeneralSettings(data),
    onSuccess: () => {
      setLastUpdate('Just now')
    },
  })

  const handleSave = () => {
    if (isProcessing) return
    
    timeoutsRef.current.forEach(clearTimeout)
    timeoutsRef.current = []
    
    setIsProcessing(true)
    setShowTimeline(true)
    setSteps(INITIAL_STEPS.map(s => ({ ...s, state: 'pending' })))

    let currentStepIndex = 0
    
    const runStep = () => {
        if (currentStepIndex >= INITIAL_STEPS.length) {
            updateMutation.mutate(formData)
            setIsProcessing(false)
            return
        }

        setSteps(prev => prev.map((s, idx) => {
            if (idx < currentStepIndex) return { ...s, state: 'done' }
            if (idx === currentStepIndex) return { ...s, state: 'running' }
            return { ...s, state: 'pending' }
        }))

        const timeout = setTimeout(() => {
            currentStepIndex++
            if (currentStepIndex < INITIAL_STEPS.length) {
                 runStep()
            } else {
                setSteps(prev => prev.map(s => ({ ...s, state: 'done' })))
                setIsProcessing(false)
                timeoutsRef.current = []
            }
        }, 800) as unknown as number
        
        timeoutsRef.current.push(timeout)
    }

    runStep()
  }

  return (
    <CardShell className="general-settings-card">
      <div className="flex-col-between-full">
        <div>
            <h3 className="settings-card-title">General</h3>
            <div className="settings-form-grid">
                <div>
                    <label htmlFor="systemName" className="settings-label">System Name</label>
                    <input 
                        id="systemName"
                        type="text" 
                        value={formData.systemName} 
                        onChange={(e) => setFormData(prev => ({ ...prev, systemName: e.target.value }))}
                        className="settings-input" 
                    />
                </div>
                <div>
                    <label htmlFor="adminEmail" className="settings-label">Admin Email</label>
                    <input 
                        id="adminEmail"
                        type="email" 
                        value={formData.adminEmail} 
                        onChange={(e) => setFormData(prev => ({ ...prev, adminEmail: e.target.value }))}
                        className="settings-input" 
                    />
                </div>
                <div>
                    <label htmlFor="publicUrl" className="settings-label">Public URL</label>
                    <input 
                        id="publicUrl"
                        type="text" 
                        value={formData.publicUrl} 
                        onChange={(e) => setFormData(prev => ({ ...prev, publicUrl: e.target.value }))}
                        className="settings-input"
                        placeholder="https://netly.yourdomain.com" 
                    />
                    <p className="text-xs text-gray-500 mt-1">Used for agent installation script</p>
                </div>
                <div>
                    <label htmlFor="environment" className="settings-label">Environment</label>
                    <select 
                        id="environment"
                        value={formData.environment}
                        onChange={(e) => setFormData(prev => ({ ...prev, environment: e.target.value }))}
                        className="settings-input"
                    >
                        <option>Production</option>
                        <option>Staging</option>
                        <option>Development</option>
                    </select>
                </div>
            </div>
        </div>

        <div className="mt-6 max-w-md flex flex-col gap-3">
            <span className="settings-last-update text-right">
                {lastUpdate ? `Last updated: ${lastUpdate}` : 'Unsaved changes'}
            </span>
            <button 
                onClick={handleSave} 
                disabled={isProcessing}
                className="settings-btn w-full"
            >
                {isProcessing ? 'Saving...' : 'Save General'}
            </button>
        </div>
      </div>

      {showTimeline && (
        <div className="mt-4">
          <VerticalProcessTimeline 
            steps={steps} 
            variant="horizontal"
          />
        </div>
      )}
    </CardShell>
  )
}
