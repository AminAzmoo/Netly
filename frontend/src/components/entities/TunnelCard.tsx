import { Tunnel, ProcessStep } from '../../types'
import CardShell from '../common/CardShell'
import StatusBadge from '../common/StatusBadge'
import VerticalProcessTimeline from '../common/VerticalProcessTimeline'
import { Network, ArrowRightLeft, Activity, Edit2, Trash2, Save, X } from 'lucide-react'

interface TunnelCardProps {
  tunnel: Tunnel
  onEdit?: () => void
  onDelete?: () => void
  isEditing?: boolean
  isProcessing?: boolean
  processSteps?: ProcessStep[]
  processType?: 'edit' | 'delete'
  onSave?: () => void
  onCancel?: () => void
  onDraftChange?: (draft: Tunnel) => void
}

export default function TunnelCard({ 
  tunnel, 
  onEdit, 
  onDelete, 
  isEditing, 
  isProcessing, 
  processSteps, 
  processType,
  onSave,
  onCancel,
  onDraftChange
}: TunnelCardProps) {
  const statusVariant =
    tunnel.status === 'Live' ? 'neonA' : tunnel.status === 'Error' ? 'error' : 'default'

  return (
    <CardShell hover className="card-relative overflow-hidden">
      <div className="entity-card-header relative z-10">
        <div className="entity-header-left w-full">
          <div className="flex justify-between items-start mb-4">
            <div className="entity-title-row flex-1">
              <Network size={20} className="text-neon-a icon-mr-2" />
              {isEditing ? (
                <input 
                  type="text" 
                  value={tunnel.name} 
                  onChange={(e) => onDraftChange?.({ ...tunnel, name: e.target.value })}
                  className="bg-white/5 border border-white/10 rounded px-2 py-1 text-sm text-white focus:border-neon-a outline-none w-48"
                />
              ) : (
                <h3 className="entity-title">{tunnel.name}</h3>
              )}
              <StatusBadge status={tunnel.type} variant="default" />
              <StatusBadge status={tunnel.status} variant={statusVariant} />
            </div>

            <div className="flex items-center gap-2">
              {isEditing ? (
                <>
                  <button 
                    onClick={onSave}
                    className="p-2 hover:bg-green-500/20 rounded-lg transition-colors group"
                    title="Save Changes"
                  >
                    <Save size={16} className="text-green-500" />
                  </button>
                  <button 
                    onClick={onCancel}
                    className="p-2 hover:bg-red-500/20 rounded-lg transition-colors group"
                    title="Cancel Edit"
                  >
                    <X size={16} className="text-red-500" />
                  </button>
                </>
              ) : (
                <>
                  <button 
                    onClick={(e) => { e.stopPropagation(); onEdit?.() }}
                    disabled={isProcessing}
                    className="p-2 hover:bg-white/10 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed group"
                    title="Edit Tunnel"
                  >
                    <Edit2 size={16} className={`text-gray-400 group-hover:text-neon-a ${isProcessing && processType === 'edit' ? 'animate-pulse text-neon-a' : ''}`} />
                  </button>
                  <button 
                    onClick={(e) => { e.stopPropagation(); onDelete?.() }}
                    disabled={isProcessing}
                    className="p-2 hover:bg-red-500/20 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed group"
                    title="Delete Tunnel"
                  >
                    <Trash2 size={16} className={`text-gray-400 group-hover:text-red-500 ${isProcessing && processType === 'delete' ? 'text-red-500' : ''}`} />
                  </button>
                </>
              )}
            </div>
          </div>

          <div className="entity-info-container">
            <div className="entity-info-row w-full">
              <ArrowRightLeft size={14} className="text-muted icon-mr-1" />
              <span className="entity-info-label whitespace-nowrap">Path:</span>
              {isEditing ? (
                 <input 
                  type="text" 
                  value={tunnel.path} 
                  onChange={(e) => onDraftChange?.({ ...tunnel, path: e.target.value })}
                  className="bg-white/5 border border-white/10 rounded px-2 py-1 text-xs text-white focus:border-neon-a outline-none w-full ml-2"
                />
              ) : (
                <span>{tunnel.path}</span>
              )}
            </div>
            {tunnel.status === 'Live' && !isEditing && (
              <div className="entity-info-row">
                <Activity size={14} className="text-muted icon-mr-1" />
                <span className="entity-info-label">Latency:</span>
                <span className="text-neon">{tunnel.latency}ms</span>
              </div>
            )}
          </div>

          <div className="flex justify-between items-center mt-4">
            {tunnel.lastAction && (
              <div className="entity-last-action flex-center">
                <Activity size={14} className="text-muted icon-mr-2" />
                <span>
                  Last action: <span className="text-muted">{tunnel.lastAction}</span> {tunnel.lastActionTime}
                </span>
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
