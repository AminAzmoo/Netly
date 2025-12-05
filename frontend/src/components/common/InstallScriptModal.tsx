import { useState } from 'react'
import { Copy, Check, X } from 'lucide-react'
import CardShell from './CardShell'

interface InstallScriptModalProps {
  isOpen: boolean
  onClose: () => void
  scriptUrl: string
}

export default function InstallScriptModal({ isOpen, onClose, scriptUrl }: InstallScriptModalProps) {
  const [copied, setCopied] = useState(false)

  if (!isOpen) return null

  const installCommand = `curl -fsSL ${scriptUrl} | sudo bash`

  const handleCopy = () => {
    navigator.clipboard.writeText(installCommand)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm">
      <CardShell className="w-full max-w-2xl mx-4">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-2xl font-bold text-white">Install Agent</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-white">
            <X size={24} />
          </button>
        </div>

        <div className="space-y-4">
          <p className="text-gray-300">
            Run this command on your server to install the Netly agent:
          </p>

          <div className="relative">
            <pre className="bg-black/50 border border-neon-a/30 rounded-lg p-4 text-neon-a font-mono text-sm overflow-x-auto">
              {installCommand}
            </pre>
            <button
              onClick={handleCopy}
              className="absolute top-2 right-2 p-2 bg-neon-a/20 hover:bg-neon-a/30 rounded-lg transition-colors"
            >
              {copied ? <Check size={20} className="text-neon-a" /> : <Copy size={20} className="text-neon-a" />}
            </button>
          </div>

          <div className="bg-neon-a/10 border border-neon-a/30 rounded-lg p-4">
            <h3 className="text-white font-semibold mb-2">Requirements:</h3>
            <ul className="text-gray-300 text-sm space-y-1">
              <li>• Root or sudo access</li>
              <li>• Ubuntu/Debian/CentOS/RHEL/Fedora</li>
              <li>• Internet connection</li>
            </ul>
          </div>

          <button onClick={onClose} className="btn-primary-glow w-full">
            Close
          </button>
        </div>
      </CardShell>
    </div>
  )
}
