import { useState, FormEvent, useEffect } from 'react'
import { X, AlertCircle } from 'lucide-react'
import { api } from '../../lib/api'
import { Device } from '../../types'

interface DeviceModalProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: () => void
  initialData?: Device
}

export default function DeviceModal({ isOpen, onClose, onSuccess, initialData }: DeviceModalProps) {
  const [name, setName] = useState('')
  const [role, setRole] = useState<'entry' | 'exit'>('entry')
  const [ip, setIp] = useState('')
  const [sshPort, setSshPort] = useState('22')
  const [username, setUsername] = useState('')
  const [authMethod, setAuthMethod] = useState<'password' | 'sshKey'>('password')
  const [password, setPassword] = useState('')
  const [sshKey, setSshKey] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [errors, setErrors] = useState<Record<string, string>>({})

  const isEditing = !!initialData

  useEffect(() => {
    if (initialData) {
      setName(initialData.name)
      setRole(initialData.role.toLowerCase() as 'entry' | 'exit')
      setIp(initialData.ip)
      setSshPort('22') // Default, we don't have this in Device type
      setUsername('') // We don't have this in Device type
      setPassword('')
      setSshKey('')
    } else {
      setName('')
      setRole('entry')
      setIp('')
      setSshPort('22')
      setUsername('')
      setPassword('')
      setSshKey('')
    }
    setErrors({})
    setError('')
  }, [initialData, isOpen])

  const validate = () => {
    const newErrors: Record<string, string> = {}
    
    if (!name.trim()) newErrors.name = 'Name is required'
    if (!isEditing && !ip.trim()) newErrors.ip = 'IP address is required'
    else if (!isEditing && !/^(\d{1,3}\.){3}\d{1,3}$/.test(ip)) newErrors.ip = 'Invalid IPv4 address'
    
    const port = parseInt(sshPort)
    if (!sshPort || isNaN(port) || port < 1 || port > 65535) {
      newErrors.sshPort = 'Port must be between 1 and 65535'
    }
    
    if (!username.trim()) newErrors.username = 'Username is required'
    
    if (!isEditing) {
      if (authMethod === 'password' && !password) {
        newErrors.password = 'Password is required'
      }
      if (authMethod === 'sshKey' && !sshKey.trim()) {
        newErrors.sshKey = 'SSH key is required'
      }
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
        ssh_port: parseInt(sshPort),
        username,
      }

      if (!isEditing) {
        payload.ip = ip
        if (authMethod === 'password') {
          payload.password = password
        } else {
          payload.private_key = sshKey
        }
      } else {
        // For editing, only include auth if provided
        if (password) payload.password = password
        if (sshKey) payload.private_key = sshKey
      }

      if (isEditing) {
        await api.updateNode(initialData.id, payload)
      } else {
        await api.createNode(payload)
      }
      
      onSuccess()
      onClose()
    } catch (err: any) {
      setError(err.message || `Failed to ${isEditing ? 'update' : 'create'} device`)
    } finally {
      setIsSubmitting(false)
    }
  }

  if (!isOpen) return null

  return (
    <div className="device-modal-overlay">
      <div className="device-modal-backdrop" onClick={onClose} />
      
      <div className="device-modal-container">
        <div className="device-modal-header">
          <h2 className="device-modal-title">
            {isEditing ? 'Edit Device' : 'Add Device'}
          </h2>
          <button onClick={onClose} className="device-modal-close-button">
            <X size={20} className="text-gray-400" />
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="device-modal-form-grid">
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
                <option value="entry">Entry Node</option>
                <option value="exit">Exit Node</option>
              </select>
            </div>

            <div>
              <label className="settings-label">IP Address *</label>
              <input
                type="text"
                value={ip}
                onChange={(e) => setIp(e.target.value)}
                disabled={isSubmitting || isEditing}
                placeholder="192.168.1.1"
                className={`settings-input ${errors.ip ? 'border-red-500' : ''} ${isEditing ? 'device-modal-ip-disabled' : ''}`}
              />
              {errors.ip && <p className="text-red-500 text-xs mt-1">{errors.ip}</p>}
              {isEditing && <p className="text-xs text-gray-500 mt-1">IP cannot be changed</p>}
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

            <div className="device-modal-form-col-span-2">
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
              <label className="settings-label">Authentication Method</label>
              <div className="device-modal-auth-method-container">
                <label className="device-modal-auth-option">
                  <input
                    type="radio"
                    value="password"
                    checked={authMethod === 'password'}
                    onChange={(e) => setAuthMethod(e.target.value as 'password')}
                    disabled={isSubmitting}
                    className="device-modal-radio-input"
                  />
                  <span className="device-modal-text-sm">Password</span>
                </label>
                <label className="device-modal-auth-option">
                  <input
                    type="radio"
                    value="sshKey"
                    checked={authMethod === 'sshKey'}
                    onChange={(e) => setAuthMethod(e.target.value as 'sshKey')}
                    disabled={isSubmitting}
                    className="device-modal-radio-input"
                  />
                  <span className="device-modal-text-sm">SSH Key</span>
                </label>
              </div>
            </div>
          </div>

          {authMethod === 'password' && (
            <div className="mb-4">
              <label className="settings-label">
                Password {isEditing ? '' : '*'}
              </label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                disabled={isSubmitting}
                placeholder={isEditing ? "Leave empty to keep current" : ""}
                className={`settings-input ${errors.password ? 'border-red-500' : ''}`}
              />
              {errors.password && <p className="text-red-500 text-xs mt-1">{errors.password}</p>}
            </div>
          )}

          {authMethod === 'sshKey' && (
            <div className="mb-4">
              <label className="settings-label">
                SSH Key {isEditing ? '' : '*'}
              </label>
              <textarea
                value={sshKey}
                onChange={(e) => setSshKey(e.target.value)}
                disabled={isSubmitting}
                rows={4}
                placeholder={isEditing ? "Leave empty to keep current" : "-----BEGIN OPENSSH PRIVATE KEY-----"}
                className="device-modal-textarea"
              />
              {errors.sshKey && <p className="text-red-500 text-xs mt-1">{errors.sshKey}</p>}
            </div>
          )}

          {error && (
            <div className="error-msg-box mb-4">
              <AlertCircle size={16} />
              <span>{error}</span>
            </div>
          )}

          <div className="device-modal-buttons-container">
            <button
              type="button"
              onClick={onClose}
              disabled={isSubmitting}
              className="device-modal-cancel-button"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="device-modal-submit-button"
            >
              {isSubmitting ? 'Saving...' : (isEditing ? 'Save Changes' : 'Save Device')}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
