import { useQuery } from '@tanstack/react-query'
import CardShell from '../components/common/CardShell'
import StatusBadge from '../components/common/StatusBadge'
import NetworkGlobe from '../components/common/NetworkGlobe'
import { api } from '../lib/api'
import { Server, Network, AlertTriangle, Bell, Activity } from 'lucide-react'

export default function DashboardPage() {
  const { data: nodes = [], isLoading: nodesLoading } = useQuery({
    queryKey: ['nodes'],
    queryFn: () => api.getNodes(),
  })

  const { data: tunnels = [], isLoading: tunnelsLoading } = useQuery({
    queryKey: ['tunnels'],
    queryFn: () => api.getTunnels(),
  })

  const { data: timeline = [], isLoading: timelineLoading } = useQuery({
    queryKey: ['timeline'],
    queryFn: () => api.getTimeline(),
  })

  const onlineNodes = nodes.filter((n: any) => n.status === 'online').length
  const activeTunnels = tunnels.filter((t: any) => t.status === 'active').length
  const incidents = timeline.filter((e: any) => e.status === 'failed').length

  const globeNodes = nodes.map((node: any) => ({
    id: node.id,
    name: node.name,
    lat: node.geo_data?.latitude || 0,
    lng: node.geo_data?.longitude || 0,
    status: node.status,
    role: node.role,
  }))

  const globeLinks = tunnels.map((tunnel: any) => ({
    source: tunnel.source_node?.name || '',
    target: tunnel.dest_node?.name || '',
    status: tunnel.status,
  }))

  return (
    <div className="dashboard-container">
      {/* Fullscreen Globe Background */}
      <div className="dashboard-globe-wrapper">
        <NetworkGlobe nodes={globeNodes} links={globeLinks} />
      </div>

      {/* Floating Title - Top Center */}
      <div className="dashboard-title-wrapper">
        <h1 className="dashboard-title">
          Dashboard
        </h1>
        <p className="dashboard-subtitle">
          Monitor your entire Netly network at a glance
        </p>
      </div>

      {/* Floating Metrics - Left */}
      <div className="dashboard-metrics-wrapper">
        <CardShell className="dashboard-metric-card">
          <div className="dashboard-metric-icon-wrapper">
            <Server size={24} className="text-neon-a" />
          </div>
          <div className="dashboard-metric-value text-neon">{nodesLoading ? '...' : onlineNodes}</div>
          <div className="dashboard-metric-label">Online Nodes</div>
          <div className="dashboard-progress-track">
            <div className="dashboard-progress-bar dashboard-bar-online"></div>
          </div>
        </CardShell>

        <CardShell className="dashboard-metric-card">
          <div className="dashboard-metric-icon-wrapper">
            <Network size={24} className="text-neon-a" />
          </div>
          <div className="dashboard-metric-value text-neon">{tunnelsLoading ? '...' : activeTunnels}</div>
          <div className="dashboard-metric-label">Active Tunnels</div>
          <div className="dashboard-progress-track">
            <div className="dashboard-progress-bar dashboard-bar-tunnels"></div>
          </div>
        </CardShell>

        <CardShell className="dashboard-metric-card">
          <div className="dashboard-metric-icon-wrapper">
            <AlertTriangle size={24} className="text-neon-a" />
          </div>
          <div className="dashboard-metric-value text-neon">{timelineLoading ? '...' : incidents}</div>
          <div className="dashboard-metric-label">Current Incidents</div>
          <div className="dashboard-progress-track">
            <div className="dashboard-progress-bar dashboard-bar-incidents"></div>
          </div>
        </CardShell>
      </div>

      {/* Floating Recent Incidents - Right */}
      <div className="dashboard-incidents-wrapper">
        <CardShell className="dashboard-metric-card h-auto justify-start">
          <div className="flex items-center gap-3 mb-4">
            <div className="dashboard-metric-icon-wrapper mb-0">
              <Bell size={24} className="text-neon-a" />
            </div>
            <h3 className="text-white font-bold text-lg">Recent Incidents</h3>
          </div>

          <div className="dashboard-incidents-list w-full">
            {timeline.slice(0, 5).map((event: any) => (
              <div key={event.id} className="dashboard-incident-item">
                <div className="dashboard-incident-header">
                  <StatusBadge
                    status={event.status}
                    variant={event.status === 'failed' ? 'error' : event.status === 'pending' ? 'warn' : 'default'}
                  />
                  <span className="dashboard-incident-time">{new Date(event.created_at).toLocaleString()}</span>
                </div>
                <p className="dashboard-incident-desc">{event.message}</p>
              </div>
            ))}
          </div>
        </CardShell>
      </div>

      {/* Bottom - Status Bar */}
      <div className="dashboard-status-wrapper">
        <CardShell className="dashboard-status-card">
          <div className="dashboard-status-content">
            <div className="dashboard-status-row">
              <StatusBadge status="System Optimal" variant="neonA" />
              <span className="dashboard-label-sm">
                <Activity size={16} className="text-neon-a" />
                All systems operational
              </span>
            </div>
            <div className="dashboard-label-xs">
              Last updated: Just now
            </div>
          </div>
        </CardShell>
      </div>
    </div>
  )
}
