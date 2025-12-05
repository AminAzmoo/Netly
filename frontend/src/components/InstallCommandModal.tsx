import { useEffect, useState } from 'react'
import { Copy, AlertTriangle, RefreshCw } from 'lucide-react'
import { api } from '../lib/api'

interface InstallCommandModalProps {
  isOpen: boolean
  onClose: () => void
  nodeId: string
}

export default function InstallCommandModal({ isOpen, onClose, nodeId }: InstallCommandModalProps) {
  const [command, setCommand] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string>('')
  const [copied, setCopied] = useState(false)

  const fetchCommand = async () => {
    setLoading(true)
    setError('')
    try {
      const response = await api.request<{ command: string }>(`/nodes/${nodeId}/command`)
      setCommand(response.command)
    } catch (err: any) {
      setError(err.message === 'HTTP 503' ? 'Tunnel not ready. Please start Cloudflare Tunnel first.' : 'Failed to fetch install command')
    } finally {
      setLoading(false)
    }
  }

  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(command)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy:', err)
    }
  }

  useEffect(() => {
    if (isOpen && nodeId) {
      fetchCommand()
    }
  }, [isOpen, nodeId])

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/80 backdrop-blur-sm" onClick={onClose} />
      <div className="relative card-shell max-w-2xl w-full">
        <div className="flex items-center justify-between mb-6">
          <h3 className="text-xl font-bold text-cyan-400">Manual Agent Installation</h3>
          <button 
            onClick={onClose}
            className="text-gray-400 hover:text-white transition-colors"
          >
            ✕
          </button>
        </div>

        {loading && (
          <div className="space-y-3">
            <div className="h-4 w-3/4 rounded bg-gray-800 animate-pulse" />
            <div className="h-20 w-full rounded bg-gray-800 animate-pulse" />
          </div>
        )}

        {error && (
          <div className="space-y-4">
            <p className="text-red-400 text-sm flex items-center gap-2">
              <AlertTriangle size={16} />
              {error}
            </p>
            <button 
              onClick={fetchCommand}
              className="flex items-center gap-2 px-4 py-2 bg-yellow-500/20 border border-yellow-500 text-yellow-500 rounded-lg hover:bg-yellow-500/30 transition-colors"
            >
              <RefreshCw size={16} />
              Retry
            </button>
          </div>
        )}

        {!loading && !error && command && (
          <div className="space-y-4">
            <div className="flex items-center gap-2 p-3 bg-yellow-500/10 border border-yellow-500/30 rounded-lg">
              <AlertTriangle size={16} className="text-yellow-500 flex-shrink-0" />
              <p className="text-yellow-400 text-sm">
                Run this command on your target server (Root access required)
              </p>
            </div>
            
            <div className="relative">
              <pre className="bg-gray-800/50 border border-yellow-500/30 rounded-lg p-4 text-xs text-gray-200 font-mono overflow-x-auto">
                {command}
              </pre>
              <button
                onClick={copyToClipboard}
                className="absolute top-2 right-2 p-2 bg-gray-700/80 hover:bg-gray-600/80 rounded-lg transition-colors"
                title="Copy to clipboard"
              >
                <Copy size={14} className={copied ? 'text-green-400' : 'text-gray-400'} />
              </button>
              {copied && (
                <div className="absolute top-2 right-12 px-2 py-1 bg-green-500/20 border border-green-500 text-green-400 text-xs rounded">
                  Copied!
                </div>
              )}
            </div>
          </div>
        )}

        <div className="flex justify-end mt-6">
          <button 
            onClick={onClose}
            className="px-4 py-2 rounded-lg border border-gray-600 text-gray-300 hover:bg-white/5 transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  )
}
