import { useState } from 'react'
import CardShell from '../common/CardShell'
import VerticalProcessTimeline from '../common/VerticalProcessTimeline'
import { ProcessStep } from '../../types'

const INITIAL_STEPS: ProcessStep[] = [
  { id: '1', label: 'Queued', state: 'pending' },
  { id: '2', label: 'Validating ranges', state: 'pending' },
  { id: '3', label: 'Recalculating pools', state: 'pending' },
  { id: '4', label: 'Syncing to nodes', state: 'pending' },
  { id: '5', label: 'Verifying conflicts', state: 'pending' },
  { id: '6', label: 'Done', state: 'pending' },
]

export default function NetworkSettingsCard() {
  const [ipPool, setIpPool] = useState('10.0.0.0/16')
  const [ipv6Pool, setIpv6Pool] = useState('')
  const [reservedBlocks, setReservedBlocks] = useState<string[]>(['10.0.0.1/32', '10.0.1.0/24'])
  const [newBlock, setNewBlock] = useState('')
  const [portRange, setPortRange] = useState('10000-20000')
  const [servicePools] = useState<{ id: string; name: string; range: string }[]>([
    { id: '1', name: 'Web', range: '80-443' },
    { id: '2', name: 'Database', range: '5432-6379' }
  ])
  
  const [steps, setSteps] = useState<ProcessStep[]>(INITIAL_STEPS)
  const [isProcessing, setIsProcessing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)
  const [showTimeline, setShowTimeline] = useState(false)

  const handleAddBlock = () => {
    if (!newBlock) return
    if (reservedBlocks.includes(newBlock)) return
    setReservedBlocks([...reservedBlocks, newBlock])
    setNewBlock('')
  }

  const handleRemoveBlock = (block: string) => {
    setReservedBlocks(reservedBlocks.filter(b => b !== block))
  }

  const handleSave = () => {
    if (isProcessing) return
    
    setIsProcessing(true)
    setSuccess(false)
    setError(null)
    setShowTimeline(true)
    setSteps(INITIAL_STEPS.map(s => ({ ...s, state: 'pending' })))

    // Simulation
    const shouldFail = Math.random() > 0.7
    let currentStepIndex = 0
    
    const runStep = () => {
      // If we reached the end of steps (success case)
      if (currentStepIndex >= INITIAL_STEPS.length) {
        setIsProcessing(false)
        setSuccess(true)
        return
      }

      // Update steps visual state
      setSteps(prev => prev.map((s, idx) => {
        if (idx < currentStepIndex) return { ...s, state: 'done' }
        if (idx === currentStepIndex) return { ...s, state: 'running' }
        return { ...s, state: 'pending' }
      }))

      // Simulate work
      setTimeout(() => {
        // Logic for error simulation at step 5 ("Verifying conflicts") or last step
        if (shouldFail && currentStepIndex === 4) { // Step 5 is index 4
          setSteps(prev => prev.map((s, idx) => {
            if (idx === currentStepIndex) return { ...s, state: 'error' }
             // keep previous as done
            if (idx < currentStepIndex) return { ...s, state: 'done' }
            return { ...s, state: 'pending' }
          }))
          setIsProcessing(false)
          setError('Conflict detected in reserved blocks.')
          return
        }

        currentStepIndex++
        if (currentStepIndex < INITIAL_STEPS.length) {
           runStep()
        } else {
          // Final completion
          setSteps(prev => prev.map(s => ({ ...s, state: 'done' })))
          setIsProcessing(false)
          setSuccess(true)
        }
      }, 800)
    }

    runStep()
  }

  return (
    <CardShell className="card-overflow-hidden">
      <div className="network-settings-content">
        <div>
          <h3 className="settings-card-title">IPAM / PortAM</h3>
          <p className="settings-card-subtitle">Manage network address pools and port allocations.</p>
        </div>

        <div className="settings-grid-2col">
          {/* Left Column: IP Pools */}
          <div className="settings-column-space">
            <h4 className="settings-section-header settings-section-header-blue">Address Pools</h4>
            
            <div>
              <label className="settings-label">IPv4 Pool Range</label>
              <input 
                type="text" 
                value={ipPool}
                onChange={(e) => setIpPool(e.target.value)}
                className="settings-input-base"
              />
            </div>

            <div>
              <label className="settings-label">IPv6 Pool Range (Optional)</label>
              <input 
                type="text" 
                value={ipv6Pool}
                onChange={(e) => setIpv6Pool(e.target.value)}
                placeholder="e.g. 2001:db8::/32"
                className="settings-input-base"
              />
            </div>

             <div>
              <label className="settings-label">Reserved Blocks</label>
              <div className="settings-reserved-list">
                {reservedBlocks.map(block => (
                  <div key={block} className="settings-list-item">
                    <span className="settings-list-text">{block}</span>
                    <button 
                      onClick={() => handleRemoveBlock(block)}
                      className="settings-remove-btn"
                    >
                      &times;
                    </button>
                  </div>
                ))}
              </div>
              <div className="settings-add-block-row">
                <input 
                  type="text" 
                  value={newBlock}
                  onChange={(e) => setNewBlock(e.target.value)}
                  placeholder="CIDR"
                  className="flex-1-input"
                />
                <button 
                  onClick={handleAddBlock}
                  className="settings-add-btn"
                >
                  Add
                </button>
              </div>
            </div>
          </div>

          {/* Right Column: Ports */}
          <div className="settings-column-space">
            <h4 className="settings-section-header settings-section-header-purple">Port Management</h4>
            
            <div>
              <label className="settings-label">Global Port Range</label>
              <input 
                type="text" 
                value={portRange}
                onChange={(e) => setPortRange(e.target.value)}
                className="settings-input-base"
              />
            </div>

            <div>
               <label className="settings-label">Service Pools</label>
               <div className="settings-reserved-list">
                 {servicePools.map(pool => (
                   <div key={pool.id} className="settings-list-item">
                     <span className="settings-list-text">{pool.name}: {pool.range}</span>
                     <button className="settings-remove-btn">&times;</button>
                   </div>
                 ))}
               </div>
               <button className="settings-add-btn w-full">
                 + Add Service Pool
               </button>
            </div>
          </div>
        </div>

        <div className="mt-3">
          <button 
            onClick={handleSave}
            disabled={isProcessing}
            className="settings-btn w-full"
          >
            {isProcessing ? 'Applying Changes...' : 'Apply Network Changes'}
          </button>

          {error && (
            <div className="error-msg-box">
              <span className="w-1.5 h-1.5 rounded-full bg-red-500" />
              {error}
            </div>
          )}

          {success && (
            <div className="success-msg-box">
               <span className="w-1.5 h-1.5 rounded-full bg-green-400" />
               Network settings applied successfully
            </div>
          )}
        </div>
      </div>

      {showTimeline && (
        <div className="px-6 pb-6">
           <VerticalProcessTimeline 
             steps={steps} 
             variant="horizontal"
           />
        </div>
      )}
    </CardShell>
  )
}
