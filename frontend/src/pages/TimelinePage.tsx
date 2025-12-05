import { useQuery } from '@tanstack/react-query'
import PageShell from '../components/layout/PageShell'
import CardShell from '../components/common/CardShell'
import StatusBadge from '../components/common/StatusBadge'
import { api } from '../lib/api'

export default function TimelinePage() {
  const { data: events = [], isLoading } = useQuery({
    queryKey: ['timeline'],
    queryFn: () => api.getTimeline(),
  })

  return (
    <PageShell title="Timeline" subtitle="View all system events and activity logs">
      <div className="timeline-grid">
        {/* Left - Filters */}
        <div className="timeline-filters-col">
          <CardShell>
            <h3 className="timeline-filter-title">Filters</h3>
            <div className="timeline-filter-group">
              <div>
                <label className="timeline-filter-label">Severity</label>
                <select className="timeline-filter-select">
                  <option>All</option>
                  <option>INFO</option>
                  <option>WARN</option>
                  <option>ERROR</option>
                </select>
              </div>
              <div>
                <label className="timeline-filter-label">Entity Type</label>
                <select className="timeline-filter-select">
                  <option>All</option>
                  <option>Device</option>
                  <option>Tunnel</option>
                  <option>Service</option>
                </select>
              </div>
              <div>
                <label className="timeline-filter-label">Time Range</label>
                <select className="timeline-filter-select">
                  <option>Last Hour</option>
                  <option>Last 24 Hours</option>
                  <option>Last 7 Days</option>
                  <option>Last 30 Days</option>
                </select>
              </div>
            </div>
          </CardShell>
        </div>

        {/* Right - Events */}
        <div className="timeline-events-col">
          <div className="timeline-events-list">
            {isLoading ? (
              <CardShell><p className="text-white">Loading events...</p></CardShell>
            ) : events.length === 0 ? (
              <CardShell><p className="text-white">No events found</p></CardShell>
            ) : events.map((event: any) => (
              <CardShell key={event.id} hover>
                <div className="timeline-event-content">
                  <div className="timeline-event-main">
                    <div className="timeline-event-header">
                      <StatusBadge
                        status={event.status}
                        variant={
                          event.status === 'failed' ? 'error' : event.status === 'pending' ? 'warn' : 'default'
                        }
                      />
                      <span className="timeline-event-time">{new Date(event.created_at).toLocaleString()}</span>
                    </div>
                    <h3 className="timeline-event-title">{event.type}</h3>
                    <p className="timeline-event-desc">{event.message}</p>
                    {event.resource_type && (
                      <div className="timeline-event-meta">
                        Related: {event.resource_type} #{event.resource_id}
                      </div>
                    )}
                  </div>
                </div>
              </CardShell>
            ))}
          </div>
        </div>
      </div>
    </PageShell>
  )
}
