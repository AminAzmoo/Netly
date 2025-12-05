import { Service, ProcessStep } from '../../types'
import StatusBadge from '../common/StatusBadge'
import CardShell from '../common/CardShell'

interface ActiveProcess {
  type: 'edit' | 'delete'
  stepIndex: number
}

interface ServicesTableProps {
  services: Service[]
  processes: Record<string, ActiveProcess>
  editingId: string | null
  editDraft: Service | null
  onEdit: (service: Service) => void
  onDelete: (serviceId: string) => void
  onSave: (serviceId: string) => void
  onCancel: () => void
  onDraftChange: (draft: Service | null) => void
  getStepsWithState: (template: ProcessStep[], currentIndex: number) => ProcessStep[]
  EDIT_STEPS: ProcessStep[]
  DELETE_STEPS: ProcessStep[]
}

export default function ServicesTable({ 
  services,
  processes,
  editingId,
  editDraft,
  onEdit,
  onDelete,
  onSave,
  onCancel,
  onDraftChange,
  getStepsWithState,
  EDIT_STEPS,
  DELETE_STEPS
}: ServicesTableProps) {
  return (
    <CardShell className="overflow-hidden p-0">
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-left text-sm text-gray-400">
          <thead className="border-b border-gray-800 bg-black/40 text-xs uppercase text-gray-500">
            <tr>
              <th className="px-4 py-3 font-medium">Name</th>
              <th className="px-4 py-3 font-medium">Protocol</th>
              <th className="px-4 py-3 font-medium">Route</th>
              <th className="px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3 font-medium">Users</th>
              <th className="px-4 py-3 font-medium">Traffic</th>
              <th className="px-4 py-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-800">
            {services.map((service) => {
              const process = processes[service.id]
              const isEditing = editingId === service.id
              const displayService = isEditing && editDraft ? editDraft : service
              
              const statusVariant =
              displayService.status === 'Ready'
                ? 'neonA'
                : displayService.status === 'Configuring'
                ? 'warn'
                : displayService.status === 'Error'
                ? 'error'
                : 'default'

              return (
                <tr key={service.id} className="hover:bg-white/5 transition-colors">
                  <td className="px-4 py-3 font-medium text-white">{displayService.name}</td>
                  <td className="px-4 py-3">
                      <span className="bg-white/5 px-2 py-1 rounded text-xs font-mono border border-white/10">
                          {displayService.protocol}
                      </span>
                  </td>
                  <td className="px-4 py-3 text-xs font-mono">
                    <div className="flex flex-col">
                      <span className="text-gray-300">{displayService.entryNode}</span>
                      <span className="text-gray-600">↓</span>
                      <span className="text-gray-300">{displayService.exitNode}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    {process ? (
                      <span className="text-xs text-yellow-400">Processing...</span>
                    ) : (
                      <StatusBadge status={displayService.status} variant={statusVariant} />
                    )}
                  </td>
                  <td className="px-4 py-3 text-white">{displayService.users}</td>
                  <td className="px-4 py-3 text-white">{displayService.traffic}</td>
                  <td className="px-4 py-3">
                    {isEditing ? (
                      <div className="flex gap-2">
                        <button onClick={() => onSave(service.id)} className="text-xs text-green-400 hover:text-green-300">Save</button>
                        <button onClick={onCancel} className="text-xs text-gray-400 hover:text-gray-300">Cancel</button>
                      </div>
                    ) : process ? (
                      <span className="text-xs text-gray-500">—</span>
                    ) : (
                      <div className="flex gap-2">
                        <button onClick={() => onEdit(service)} className="text-xs text-blue-400 hover:text-blue-300">Edit</button>
                        <button onClick={() => onDelete(service.id)} className="text-xs text-red-400 hover:text-red-300">Delete</button>
                      </div>
                    )}
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
