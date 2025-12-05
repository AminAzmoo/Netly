import { Service, ProcessStep } from '../../types'
import CardShell from '../common/CardShell'
import StatusBadge from '../common/StatusBadge'
import VerticalProcessTimeline from '../common/VerticalProcessTimeline'
import { Activity, LogIn, LogOut, Users, Network, Edit2, Trash2, Save, X } from 'lucide-react'

interface ServiceCardProps {
  service: Service
  onEdit?: () => void
  onDelete?: () => void
  isEditing?: boolean
  isProcessing?: boolean
  processSteps?: ProcessStep[]
  processType?: 'edit' | 'delete'
  onSave?: () => void
  onCancel?: () => void
  onDraftChange?: (draft: Service) => void
}

export default function ServiceCard({ 
  service, 
  onEdit, 
  onDelete, 
  isEditing, 
  isProcessing, 
  processSteps, 
  processType,
  onSave,
  onCancel,
  onDraftChange
}: ServiceCardProps) {
  const statusVariant =
    service.status === 'Ready' ? 'neonA' : service.status === 'Error' ? 'error' : 'default'

  return (
    <CardShell hover className="card-relative overflow-hidden">
      <div className="entity-card-header relative z-10">
        <div className="entity-header-left w-full">
          <div className="flex justify-between items-start mb-4">
            <div className="entity-title-row flex-1">
              <Activity size={20} className="text-neon-a icon-mr-2" />
              {isEditing ? (
                <input 
                  type="text" 
                  value={service.name} 
                  onChange={(e) => onDraftChange?.({ ...service, name: e.target.value })}
                  className="bg-white/5 border border-white/10 rounded px-2 py-1 text-sm text-white focus:border-neon-a outline-none w-48"
                />
              ) : (
                <h3 className="entity-title">{service.name}</h3>
              )}
              <StatusBadge status={service.protocol} variant="default" />
              <StatusBadge status={service.status} variant={statusVariant} />
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
                    title="Edit Service"
                  >
                    <Edit2 size={16} className={`text-gray-400 group-hover:text-neon-a ${isProcessing && processType === 'edit' ? 'animate-pulse text-neon-a' : ''}`} />
                  </button>
                  <button 
                    onClick={(e) => { e.stopPropagation(); onDelete?.() }}
                    disabled={isProcessing}
                    className="p-2 hover:bg-red-500/20 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed group"
                    title="Delete Service"
                  >
                    <Trash2 size={16} className={`text-gray-400 group-hover:text-red-500 ${isProcessing && processType === 'delete' ? 'text-red-500' : ''}`} />
                  </button>
                </>
              )}
            </div>
          </div>

          <div className="entity-info-container">
            <div className="entity-info-row">
              <LogIn size={14} className="text-muted icon-mr-1" />
              <span className="entity-info-label">Entry:</span>
              <span>{service.entryNode}</span>
            </div>
            <div className="entity-info-row">
              <LogOut size={14} className="text-muted icon-mr-1" />
              <span className="entity-info-label">Exit:</span>
              <span>{service.exitNode}</span>
            </div>
            <div className="entity-stats-row">
              <div className="entity-info-row">
                <Users size={14} className="text-muted icon-mr-1" />
                <span className="entity-info-label">Users:</span>
                <span className="text-neon">{service.users}</span>
              </div>
              <div className="entity-info-row">
                <Network size={14} className="text-muted icon-mr-1" />
                <span className="entity-info-label">Traffic:</span>
                <span className="text-neon">{service.traffic}</span>
              </div>
            </div>
          </div>

          {service.lastAction && (
            <div className="entity-last-action flex-center">
              <Activity size={14} className="text-muted icon-mr-2" />
              <span>
                Last action: <span className="text-muted">{service.lastAction}</span> {service.lastActionTime}
              </span>
            </div>
          )}
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
