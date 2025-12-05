import { useState, FormEvent } from 'react'
import { X, AlertCircle } from 'lucide-react'
import { api } from '../../lib/api'

interface Node {
  id: string
  name: string
  role: 'entry' | 'exit'
  ip: string
}

interface AddTunnelModalProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: () => void
  nodes: Node[]
}

export default function AddTunnelModal({ isOpen, onClose, onSuccess, nodes }: AddTunnelModalProps) {
  const [name, setName] = useState('')
  const [entryNodeId, setEntryNodeId] = useState('')
  const [exitNodeId, setExitNodeId] = useState('')
  const [protocol, setProtocol] = useState<'wireguard' | 'hysteria2' | 'vless'>('wireguard')
  const [port, setPort] = useState('51820')
  const [enableObfuscation, setEnableObfuscation] = useState(false)
  const [notes, setNotes] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [errors, setErrors] = useState<Record<string, string>>({})

  const entryNodes = nodes.filter(n => n.role === 'entry')
  const exitNodes = nodes.filter(n => n.role === 'exit')

  const validate = () => {
    const newErrors: Record<string, string> = {}
    
    if (!name.trim()) newErrors.name = 'Name is required'
    if (!entryNodeId) newErrors.entryNodeId = 'Entry node is required'
    if (!exitNodeId) newErrors.exitNodeId = 'Exit node is required'
    
    if (entryNodeId && exitNodeId && entryNodeId === exitNodeId) {
      newErrors.exitNodeId = 'Entry and exit nodes must be different'
    }
    
    const portNum = parseInt(port)
    if (!port || isNaN(portNum) || portNum < 1 || portNum > 65535) {
      newErrors.port = 'Port must be between 1 and 65535'
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    
    if (!validate()) return

    setIsSubmitting(true)
    try {
      const payload = {
        name,
        protocol,
        source_node_id: parseInt(entryNodeId),
        dest_node_id: parseInt(exitNodeId),
        source_port: parseInt(port),
        dest_port: parseInt(port),
        enableObfuscation,
        notes: notes.trim() || undefined,
      }

      await api.createTunnel(payload)
      
      // Reset form
      setName('')
      setEntryNodeId('')
      setExitNodeId('')
      setProtocol('wireguard')
      setPort('51820')
      setEnableObfuscation(false)
      setNotes('')
      setErrors({})
      
      onSuccess()
      onClose()
    } catch (err: any) {
      setError(err.message || 'Failed to create tunnel')
    } finally {
      setIsSubmitting(false)
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/80 backdrop-blur-sm" onClick={onClose} />
      
      <div className="relative w-full max-w-2xl max-h-[90vh] overflow-y-auto card-shell">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-2xl font-bold text-white">Add Tunnel</h2>
          <button onClick={onClose} className="p-2 hover:bg-white/10 rounded-lg transition-colors">
            <X size={20} className="text-gray-400" />
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
            <div className="md:col-span-2">
              <label className="settings-label">Tunnel Name *</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                disabled={isSubmitting}
                placeholder="Tehran-Frankfurt-WG"
                className={`settings-input ${errors.name ? 'border-red-500' : ''}`}
              />
              {errors.name && <p className="text-red-500 text-xs mt-1">{errors.name}</p>}
            </div>

            <div>
              <label className="settings-label">Entry Node *</label>
              <select
                value={entryNodeId}
                onChange={(e) => setEntryNodeId(e.target.value)}
                disabled={isSubmitting}
                className={`settings-input ${errors.entryNodeId ? 'border-red-500' : ''}`}
              >
                <option value="">Select entry node...</option>
                {entryNodes.map(node => (
                  <option key={node.id} value={node.id}>
                    {node.name} – {node.ip}
                  </option>
                ))}
              </select>
              {errors.entryNodeId && <p className="text-red-500 text-xs mt-1">{errors.entryNodeId}</p>}
            </div>

            <div>
              <label className="settings-label">Exit Node *</label>
              <select
                value={exitNodeId}
                onChange={(e) => setExitNodeId(e.target.value)}
                disabled={isSubmitting}
                className={`settings-input ${errors.exitNodeId ? 'border-red-500' : ''}`}
              >
                <option value="">Select exit node...</option>
                {exitNodes.map(node => (
                  <option key={node.id} value={node.id}>
                    {node.name} – {node.ip}
                  </option>
                ))}
              </select>
              {errors.exitNodeId && <p className="text-red-500 text-xs mt-1">{errors.exitNodeId}</p>}
            </div>

            <div>
              <label className="settings-label">Protocol *</label>
              <select
                value={protocol}
                onChange={(e) => setProtocol(e.target.value as any)}
                disabled={isSubmitting}
                className="settings-input"
              >
                <option value="wireguard">WireGuard</option>
                <option value="hysteria2">Hysteria2</option>
                <option value="vless">VLESS</option>
              </select>
            </div>

            <div>
              <label className="settings-label">Port *</label>
              <input
                type="number"
                value={port}
                onChange={(e) => setPort(e.target.value)}
                disabled={isSubmitting}
                min="1"
                max="65535"
                className={`settings-input ${errors.port ? 'border-red-500' : ''}`}
              />
              {errors.port && <p className="text-red-500 text-xs mt-1">{errors.port}</p>}
            </div>
          </div>

          <div className="mb-4">
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={enableObfuscation}
                onChange={(e) => setEnableObfuscation(e.target.checked)}
                disabled={isSubmitting}
                className="w-4 h-4 accent-purple-600"
              />
              <span className="text-sm text-gray-300">Enable Obfuscation</span>
            </label>
          </div>

          <div className="mb-6">
            <label className="settings-label">Notes</label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              disabled={isSubmitting}
              rows={3}
              placeholder="Additional configuration notes..."
              className="settings-input"
            />
          </div>

          {error && (
            <div className="error-msg-box mb-4">
              <AlertCircle size={16} />
              <span>{error}</span>
            </div>
          )}

          <div className="flex justify-end gap-3">
            <button
              type="button"
              onClick={onClose}
              disabled={isSubmitting}
              className="px-4 py-2 rounded-lg border border-gray-600 text-gray-300 hover:bg-white/5 transition-colors disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="btn-primary-glow disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isSubmitting ? 'Creating...' : 'Create Tunnel'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
