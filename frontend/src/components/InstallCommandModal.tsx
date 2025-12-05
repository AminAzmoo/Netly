import { useEffect, useState } from 'react'
import { AlertTriangle, Copy } from 'lucide-react'

interface InstallCommandModalProps {
  nodeId: string | null
  isOpen: boolean
  onClose: () => void
}

interface CommandResponse {
  command: string
  api_url: string
  token: string
}

export default function InstallCommandModal({ nodeId, isOpen, onClose }: InstallCommandModalProps) {
  const [command, setCommand] = useState<CommandResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const fetchCommand = () => {
    if (!nodeId) return
    
    setLoading(true)
    setError(null)
    
    fetch(`http://localhost:8081/api/v1/nodes/${nodeId}/command`, {
      headers: {
        'X-Admin-Token': 'change-me-admin'
      }
    })
      .then(res => {
        if (!res.ok) {
          throw new Error(`HTTP ${res.status}: ${res.statusText}`)
        }
        return res.json()
      })
      .then(data => {
        if (data.error) {
          setError(data.error)
        } else {
          setCommand(data)
        }
      })
      .catch(err => setError(err.message))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    if (isOpen && nodeId) {
      fetchCommand()
    }
  }, [isOpen, nodeId])

  const handleClose = () => {
    setCommand(null)
    setError(null)
    setCopied(false)
    onClose()
  }

  const copyToClipboard = () => {
    if (command) {
      navigator.clipboard.writeText(command.command)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/80 backdrop-blur-sm" onClick={handleClose} />
      <div className="relative card-shell max-w-sm w-full">
        <h3 className="text-lg font-bold text-white mb-3">Manual Installation</h3>
        
        <div className="flex items-center gap-2 p-2 bg-yellow-500/20 rounded border border-yellow-500/50 mb-3">
          <AlertTriangle className="w-5 h-5 text-yellow-500" />
          <p className="text-sm text-yellow-200">
            Run this command on your server (Root access required):
          </p>
        </div>

        {loading && (
          <div className="space-y-3">
            <div className="h-4 bg-gray-700 rounded animate-pulse" />
            <div className="h-20 bg-gray-700 rounded animate-pulse" />
          </div>
        )}

        {error && (
          <div className="p-3 bg-red-500/20 rounded-lg border border-red-500/50 mb-4">
            <p className="text-sm text-red-200 mb-3">Error: {error}</p>
            <button 
              onClick={fetchCommand}
              className="px-3 py-1 bg-blue-500/20 border border-blue-500 text-blue-400 rounded hover:bg-blue-500/30 text-sm"
            >
              Retry
            </button>
          </div>
        )}

        {command && !loading && (
          <div className="space-y-4">
            <div>
              <p className="text-sm font-medium mb-2 text-gray-300">Installation Command:</p>
              <div className="relative w-full">
                <pre className="bg-gray-900 p-2 rounded text-xs text-green-400 overflow-x-auto border border-gray-700 max-w-full break-all whitespace-pre-wrap">
                  {command.command}
                </pre>
                <button 
                  onClick={copyToClipboard}
                  className="absolute top-2 right-2 p-1.5 bg-gray-800 hover:bg-gray-700 rounded transition-colors"
                  title="Copy to clipboard"
                >
                  <Copy size={12} className={copied ? 'text-green-500' : 'text-gray-400'} />
                </button>
              </div>
              {copied && <p className="text-xs text-green-500 mt-1">Copied to clipboard!</p>}
            </div>
            
            <div className="grid grid-cols-2 gap-4 text-xs text-gray-400">
              <div>
                <span className="font-medium">API URL:</span> {command.api_url}
              </div>
              <div>
                <span className="font-medium">Token:</span> {command.token}
              </div>
            </div>
          </div>
        )}

        <div className="flex justify-end mt-4">
          <button onClick={handleClose} className="px-4 py-2 rounded-lg border border-gray-600 text-gray-300 hover:bg-white/5">
            Close
          </button>
        </div>
      </div>
    </div>
  )
}