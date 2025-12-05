import { useState, FormEvent } from 'react'
import { X, AlertCircle } from 'lucide-react'
import { api } from '../../lib/api'

interface AddDeviceModalProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: () => void
}

export default function AddDeviceModal({ isOpen, onClose, onSuccess }: AddDeviceModalProps) {
  const [name, setName] = useState('')
  const [role, setRole] = useState<'entry' | 'exit'>('entry')
  const [ip, setIp] = useState('')
  const [sshPort, setSshPort] = useState('22')
  const [username, setUsername] = useState('')
  const [authMethod, setAuthMethod] = useState<'password' | 'sshKey'>('password')
  const [password, setPassword] = useState('')
  const [sshKey, setSshKey] = useState('')
  const [subdomain, setSubdomain] = useState('')
  const [enableCloudflareProxy, setEnableCloudflareProxy] = useState(false)
  const [enableWarpOutbound, setEnableWarpOutbound] = useState(false)
  const [tags, setTags] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState('')

  const [errors, setErrors] = useState<Record<string, string>>({})

  const validate = () => {
    const newErrors: Record<string, string> = {}
    
    if (!name.trim()) newErrors.name = 'Name is required'
    if (!ip.trim()) newErrors.ip = 'IP address is required'
    else if (!/^(\d{1,3}\.){3}\d{1,3}$/.test(ip)) newErrors.ip = 'Invalid IPv4 address'
    
    const port = parseInt(sshPort)
    if (!sshPort || isNaN(port) || port < 1 || port > 65535) {
      newErrors.sshPort = 'Port must be between 1 and 65535'
    }
    
    if (!username.trim()) newErrors.username = 'Username is required'
    
    if (authMethod === 'password' && !password) {
      newErrors.password = 'Password is required'
    }
    if (authMethod === 'sshKey' && !sshKey.trim()) {
      newErrors.sshKey = 'SSH key is required'
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
        role,
        ip,
        ssh_port: parseInt(sshPort),
        username,
      }

      if (authMethod === 'password') {
        payload.password = password
      } else {
        payload.private_key = sshKey
      }

      if (subdomain.trim()) {
        payload.subdomain = subdomain.trim()
      }

      if (tags.trim()) {
        payload.tags = tags.split(',').map(t => t.trim()).filter(Boolean)
      }

      await api.createNode(payload)
      
      // Reset form
      setName('')
      setRole('entry')
      setIp('')
      setSshPort('22')
      setUsername('')
      setPassword('')
      setSshKey('')
      setSubdomain('')
      setEnableCloudflareProxy(false)
      setEnableWarpOutbound(false)
      setTags('')
      setErrors({})
      
      onSuccess()
      onClose()
    } catch (err: any) {
      setError(err.message || 'Failed to create device')
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
          <h2 className="text-2xl font-bold text-white">Add Device</h2>
          <button onClick={onClose} className="p-2 hover:bg-white/10 rounded-lg transition-colors">
            <X size={20} className="text-gray-400" />
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
            <div>
              <label className="settings-label">Name *</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                disabled={isSubmitting}
                className={`settings-input ${errors.name ? 'border-red-500' : ''}`}
              />
              {errors.name && <p className="text-red-500 text-xs mt-1">{errors.name}</p>}
            </div>

            <div>
              <label className="settings-label">Role *</label>
              <select
                value={role}
                onChange={(e) => setRole(e.target.value as 'entry' | 'exit')}
                disabled={isSubmitting}
                className="settings-input"
              >
                <option value="entry">Entry Node – Iran / inbound</option>
                <option value="exit">Exit Node – abroad / outbound</option>
              </select>
            </div>

            <div>
              <label className="settings-label">IP Address *</label>
              <input
                type="text"
                value={ip}
                onChange={(e) => setIp(e.target.value)}
                disabled={isSubmitting}
                placeholder="192.168.1.1"
                className={`settings-input ${errors.ip ? 'border-red-500' : ''}`}
              />
              {errors.ip && <p className="text-red-500 text-xs mt-1">{errors.ip}</p>}
            </div>

            <div>
              <label className="settings-label">SSH Port *</label>
              <input
                type="number"
                value={sshPort}
                onChange={(e) => setSshPort(e.target.value)}
                disabled={isSubmitting}
                min="1"
                max="65535"
                className={`settings-input ${errors.sshPort ? 'border-red-500' : ''}`}
              />
              {errors.sshPort && <p className="text-red-500 text-xs mt-1">{errors.sshPort}</p>}
            </div>

            <div>
              <label className="settings-label">Subdomain</label>
              <input
                type="text"
                value={subdomain}
                onChange={(e) => setSubdomain(e.target.value)}
                disabled={isSubmitting}
                placeholder="node1"
                className="settings-input"
              />
            </div>

            <div>
              <label className="settings-label">Username *</label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                disabled={isSubmitting}
                className={`settings-input ${errors.username ? 'border-red-500' : ''}`}
              />
              {errors.username && <p className="text-red-500 text-xs mt-1">{errors.username}</p>}
            </div>

            <div>
              <label className="settings-label">Authentication Method *</label>
              <div className="flex gap-4 mt-2">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    value="password"
                    checked={authMethod === 'password'}
                    onChange={(e) => setAuthMethod(e.target.value as 'password')}
                    disabled={isSubmitting}
                    className="accent-purple-600"
                  />
                  <span className="text-sm text-gray-300">Password</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    value="sshKey"
                    checked={authMethod === 'sshKey'}
                    onChange={(e) => setAuthMethod(e.target.value as 'sshKey')}
                    disabled={isSubmitting}
                    className="accent-purple-600"
                  />
                  <span className="text-sm text-gray-300">SSH Key</span>
                </label>
              </div>
            </div>
          </div>

          {authMethod === 'password' && (
            <div className="mb-4">
              <label className="settings-label">Password *</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                disabled={isSubmitting}
                className={`settings-input ${errors.password ? 'border-red-500' : ''}`}
              />
              {errors.password && <p className="text-red-500 text-xs mt-1">{errors.password}</p>}
            </div>
          )}

          {authMethod === 'sshKey' && (
            <div className="mb-4">
              <label className="settings-label">SSH Key *</label>
              <textarea
                value={sshKey}
                onChange={(e) => setSshKey(e.target.value)}
                disabled={isSubmitting}
                rows={4}
                placeholder="-----BEGIN OPENSSH PRIVATE KEY-----"
                className={`settings-input font-mono text-xs ${errors.sshKey ? 'border-red-500' : ''}`}
              />
              {errors.sshKey && <p className="text-red-500 text-xs mt-1">{errors.sshKey}</p>}
            </div>
          )}

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={enableCloudflareProxy}
                onChange={(e) => setEnableCloudflareProxy(e.target.checked)}
                disabled={isSubmitting}
                className="w-4 h-4 accent-purple-600"
              />
              <span className="text-sm text-gray-300">Enable Cloudflare Proxy</span>
            </label>

            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={enableWarpOutbound}
                onChange={(e) => setEnableWarpOutbound(e.target.checked)}
                disabled={isSubmitting}
                className="w-4 h-4 accent-purple-600"
              />
              <span className="text-sm text-gray-300">Enable WARP Outbound</span>
            </label>
          </div>

          <div className="mb-6">
            <label className="settings-label">Tags (comma-separated)</label>
            <input
              type="text"
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              disabled={isSubmitting}
              placeholder="production, iran, high-priority"
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
              {isSubmitting ? 'Saving...' : 'Save Device'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
