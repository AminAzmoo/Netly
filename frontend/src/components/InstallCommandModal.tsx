import { useEffect, useState } from 'react'
import { Modal, ModalContent, ModalHeader, ModalBody, ModalFooter, Button, Snippet, Skeleton } from '@heroui/react'
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

  const fetchCommand = async () => {
    setLoading(true)
    setError('')
    try {
      const response = await api.get(`/nodes/${nodeId}/command`)
      setCommand(response.data.command)
    } catch (err: any) {
      setError(err.response?.status === 503 ? 'Tunnel not ready. Please start Cloudflare Tunnel first.' : 'Failed to fetch install command')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (isOpen && nodeId) {
      fetchCommand()
    }
  }, [isOpen, nodeId])

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="2xl" backdrop="blur">
      <ModalContent className="bg-gray-900 border border-cyan-500/20">
        <ModalHeader className="text-cyan-400">Install Agent</ModalHeader>
        <ModalBody>
          {loading && (
            <div className="space-y-3">
              <Skeleton className="h-4 w-3/4 rounded bg-gray-800" />
              <Skeleton className="h-20 w-full rounded bg-gray-800" />
            </div>
          )}

          {error && (
            <div className="space-y-4">
              <p className="text-red-400 text-sm">{error}</p>
              <Button color="warning" variant="flat" onClick={fetchCommand}>
                Retry
              </Button>
            </div>
          )}

          {!loading && !error && command && (
            <div className="space-y-4">
              <p className="text-yellow-400 text-sm flex items-center gap-2">
                <span className="text-lg">⚠️</span>
                Run this command on your target server (Root access required)
              </p>
              <Snippet 
                symbol="" 
                color="warning" 
                className="w-full"
                classNames={{
                  base: "bg-gray-800/50 border border-yellow-500/30",
                  pre: "text-xs text-gray-200 font-mono"
                }}
              >
                {command}
              </Snippet>
            </div>
          )}
        </ModalBody>
        <ModalFooter>
          <Button color="default" variant="light" onClick={onClose}>
            Close
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  )
}
