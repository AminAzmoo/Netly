import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import CardShell from '../common/CardShell'
import { api } from '../../lib/api'

export default function SystemCard() {
  const [isClearing, setIsClearing] = useState(false)

  const clearLogsMutation = useMutation({
    mutationFn: () => fetch('http://localhost:8081/api/v1/settings/logs', {
      method: 'DELETE',
      headers: {
        'X-Admin-Token': 'change-me-admin'
      }
    }).then(res => res.json()),
    onSuccess: () => {
      alert('Logs cleared successfully')
    },
    onError: () => {
      alert('Failed to clear logs')
    }
  })

  const handleClearLogs = () => {
    if (confirm('Are you sure you want to clear all logs?')) {
      setIsClearing(true)
      clearLogsMutation.mutate()
      setTimeout(() => setIsClearing(false), 2000)
    }
  }

  return (
    <CardShell className="system-card">
      <div className="flex-col-between-full">
        <div>
          <h3 className="settings-card-title">System</h3>
          <div className="settings-form-grid">
            <div>
              <label className="settings-label">Log Management</label>
              <p className="text-xs text-gray-500 mb-3">Clear all system logs and error files</p>
              <button 
                onClick={handleClearLogs}
                disabled={isClearing}
                className="px-4 py-2 bg-red-500/20 border border-red-500 text-red-500 rounded-lg hover:bg-red-500/30 disabled:opacity-50"
              >
                {isClearing ? 'Clearing...' : 'Clear Logs'}
              </button>
            </div>
          </div>
        </div>
      </div>
    </CardShell>
  )
}