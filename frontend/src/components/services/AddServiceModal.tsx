import { useState, FormEvent } from 'react'
import { X, AlertCircle } from 'lucide-react'
import { api } from '../../lib/api'

interface Node {
  id: string
  name: string
  role: 'entry' | 'exit'
  ip: string
}

interface Tunnel {
  id: string
  name: string
}

interface AddServiceModalProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: () => void
  nodes: Node[]
  tunnels: Tunnel[]
}

export default function AddServiceModal({ isOpen, onClose, onSuccess, nodes, tunnels }: AddServiceModalProps) {
  const [name, setName] = useState('')
  const [protocol, setProtocol] = useState<'vless-reality' | 'hysteria2' | 'wireguard' | 'openconnect' | 'l2tp-ipsec'>('vless-reality')
  const [bindType, setBindType] = useState<'direct-node' | 'via-tunnel'>('direct-node')
  const [targetNodeId, setTargetNodeId] = useState('')
  const [tunnelId, setTunnelId] = useState('')
  const [publicPort, setPublicPort] = useState('443')
  const [enableWarpSanitizer, setEnableWarpSanitizer] = useState(false)
  const [clientConfigTemplate, setClientConfigTemplate] = useState('')
  const [notes, setNotes] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [errors, setErrors] = useState<Record<string, string>>({})

  const entryNodes = nodes.filter(n => n.role === 'entry')

  const validate = () => {
    const newErrors: Record<string, string> = {}
    
    if (!name.trim()) newErrors.name = 'Name is required'
    if (!clientConfigTemplate.trim()) newErrors.clientConfigTemplate = 'Client config template is required'
    
    if (bindType === 'direct-node' && !targetNodeId) {
      newErrors.targetNodeId = 'Target node is required'
    }
    if (bindType === 'via-tunnel' && !tunnelId) {
      newErrors.tunnelId = 'Tunnel is required'
    }
    
    const portNum = parseInt(publicPort)
    if (!publicPort || isNaN(portNum) || portNum < 1 || portNum > 65535) {
      newErrors.publicPort = 'Port must be between 1 and 65535'
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
      const payload: any = {
        name,
        protocol,
        node_id: bindType === 'direct-node' ? parseInt(targetNodeId) : parseInt(tunnelId),
        listen_port: parseInt(publicPort),
        routing_mode: bindType === 'direct-node' ? 'direct' : 'tunnel',
        config: {
          clientConfigTemplate,
          enableWarpSanitizer,
          notes: notes.trim() || undefined,
        },
      }

      await api.createService(payload)
      
      // Reset form
      setName('')
      setProtocol('vless-reality')
      setBindType('direct-node')
      setTargetNodeId('')
      setTunnelId('')
      setPublicPort('443')
      setEnableWarpSanitizer(false)
      setClientConfigTemplate('')
      setNotes('')
      setErrors({})
      
      onSuccess()
      onClose()
    } catch (err: any) {
      setError(err.message || 'Failed to create service')
    } finally {
      setIsSubmitting(false)
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/80 backdrop-blur-sm" onClick={onClose} />
      
      <div className="relative w-full max-w-3xl max-h-[90vh] overflow-y-auto card-shell">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-2xl font-bold text-white">Add Service</h2>
          <button onClick={onClose} className="p-2 hover:bg-white/10 rounded-lg transition-colors">
            <X size={20} className="text-gray-400" />
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
            <div>
              <label className="settings-label">Service Name *</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                disabled={isSubmitting}
                placeholder="VLESS-Reality-Tehran"
                className={`settings-input ${errors.name ? 'border-red-500' : ''}`}
              />
              {errors.name && <p className="text-red-500 text-xs mt-1">{errors.name}</p>}
            </div>

            <div>
              <label className="settings-label">Protocol *</label>
              <select
                value={protocol}
                onChange={(e) => setProtocol(e.target.value as any)}
                disabled={isSubmitting}
                className="settings-input"
              >
                <option value="vless-reality">VLESS Reality</option>
                <option value="hysteria2">Hysteria2</option>
                <option value="wireguard">WireGuard</option>
                <option value="openconnect">OpenConnect</option>
                <option value="l2tp-ipsec">L2TP/IPsec</option>
              </select>
            </div>

            <div className="md:col-span-2">
              <label className="settings-label">Bind Type *</label>
              <div className="flex gap-4 mt-2">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    value="direct-node"
                    checked={bindType === 'direct-node'}
                    onChange={(e) => setBindType(e.target.value as 'direct-node')}
                    disabled={isSubmitting}
                    className="accent-purple-600"
                  />
                  <span className="text-sm text-gray-300">Direct Node</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    value="via-tunnel"
                    checked={bindType === 'via-tunnel'}
                    onChange={(e) => setBindType(e.target.value as 'via-tunnel')}
                    disabled={isSubmitting}
                    className="accent-purple-600"
                  />
                  <span className="text-sm text-gray-300">Via Tunnel</span>
                </label>
              </div>
            </div>

            {bindType === 'direct-node' && (
              <div className="md:col-span-2">
                <label className="settings-label">Target Node *</label>
                <select
                  value={targetNodeId}
                  onChange={(e) => setTargetNodeId(e.target.value)}
                  disabled={isSubmitting}
                  className={`settings-input ${errors.targetNodeId ? 'border-red-500' : ''}`}
                >
                  <option value="">Select node...</option>
                  {entryNodes.map(node => (
                    <option key={node.id} value={node.id}>
                      {node.name} – {node.ip}
                    </option>
                  ))}
                </select>
                {errors.targetNodeId && <p className="text-red-500 text-xs mt-1">{errors.targetNodeId}</p>}
              </div>
            )}

            {bindType === 'via-tunnel' && (
              <div className="md:col-span-2">
                <label className="settings-label">Tunnel *</label>
                <select
                  value={tunnelId}
                  onChange={(e) => setTunnelId(e.target.value)}
                  disabled={isSubmitting}
                  className={`settings-input ${errors.tunnelId ? 'border-red-500' : ''}`}
                >
                  <option value="">Select tunnel...</option>
                  {tunnels.map(tunnel => (
                    <option key={tunnel.id} value={tunnel.id}>
                      {tunnel.name}
                    </option>
                  ))}
                </select>
                {errors.tunnelId && <p className="text-red-500 text-xs mt-1">{errors.tunnelId}</p>}
              </div>
            )}

            <div>
              <label className="settings-label">Public Port *</label>
              <input
                type="number"
                value={publicPort}
                onChange={(e) => setPublicPort(e.target.value)}
                disabled={isSubmitting}
                min="1"
                max="65535"
                className={`settings-input ${errors.publicPort ? 'border-red-500' : ''}`}
              />
              {errors.publicPort && <p className="text-red-500 text-xs mt-1">{errors.publicPort}</p>}
            </div>

            <div className="flex items-end">
              <label className="flex items-center gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  checked={enableWarpSanitizer}
                  onChange={(e) => setEnableWarpSanitizer(e.target.checked)}
                  disabled={isSubmitting}
                  className="w-4 h-4 accent-purple-600"
                />
                <span className="text-sm text-gray-300">Enable WARP Sanitizer</span>
              </label>
            </div>
          </div>

          <div className="mb-4">
            <label className="settings-label">Client Config Template *</label>
            <textarea
              value={clientConfigTemplate}
              onChange={(e) => setClientConfigTemplate(e.target.value)}
              disabled={isSubmitting}
              rows={6}
              placeholder={`[Interface]
PrivateKey = {{PRIVATE_KEY}}
Address = {{CLIENT_IP}}

[Peer]
PublicKey = {{SERVER_PUBLIC_KEY}}
Endpoint = {{SERVER_IP}}:{{PORT}}`}
              className={`settings-input font-mono text-xs ${errors.clientConfigTemplate ? 'border-red-500' : ''}`}
            />
            {errors.clientConfigTemplate && <p className="text-red-500 text-xs mt-1">{errors.clientConfigTemplate}</p>}
          </div>

          <div className="mb-6">
            <label className="settings-label">Notes</label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              disabled={isSubmitting}
              rows={3}
              placeholder="Additional notes..."
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
              {isSubmitting ? 'Creating...' : 'Create Service'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
