import { useQuery } from '@tanstack/react-query'
import { Network, Server, Activity, Globe, Hash, Link } from 'lucide-react'
import { api } from '../../lib/api'

export default function NetworkResourcesCard() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['network-stats'],
    queryFn: () => api.getNetworkStats(),
    refetchInterval: 30000,
  })

  if (isLoading) {
    return (
      <div className="card-shell">
        <h3 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
          <Network size={20} className="text-purple-400" />
          Network Resources
        </h3>
        <div className="animate-pulse space-y-3">
          <div className="h-4 bg-gray-700 rounded w-3/4"></div>
          <div className="h-4 bg-gray-700 rounded w-1/2"></div>
        </div>
      </div>
    )
  }

  if (error || !data) {
    return (
      <div className="card-shell">
        <h3 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
          <Network size={20} className="text-purple-400" />
          Network Resources
        </h3>
        <p className="text-red-400 text-sm">Failed to load network stats</p>
      </div>
    )
  }

  const portUsagePercent = (data.portam.used_count / data.portam.total_range) * 100

  return (
    <div className="card-shell">
      <h3 className="text-lg font-semibold text-white mb-4 flex items-center gap-2">
        <Network size={20} className="text-purple-400" />
        Network Resources
      </h3>

      {/* Summary Stats */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-800/50 rounded-lg p-3 text-center">
          <Server size={18} className="text-blue-400 mx-auto mb-1" />
          <div className="text-xl font-bold text-white">{data.summary.total_nodes}</div>
          <div className="text-xs text-gray-400">Nodes</div>
        </div>
        <div className="bg-gray-800/50 rounded-lg p-3 text-center">
          <Activity size={18} className="text-green-400 mx-auto mb-1" />
          <div className="text-xl font-bold text-white">{data.summary.total_tunnels}</div>
          <div className="text-xs text-gray-400">Tunnels</div>
        </div>
        <div className="bg-gray-800/50 rounded-lg p-3 text-center">
          <Globe size={18} className="text-purple-400 mx-auto mb-1" />
          <div className="text-xl font-bold text-white">{data.summary.total_services}</div>
          <div className="text-xs text-gray-400">Services</div>
        </div>
      </div>

      {/* IPAM Section */}
      <div className="mb-6">
        <h4 className="text-sm font-medium text-gray-300 mb-3 flex items-center gap-2">
          <Hash size={14} />
          IP Address Management (IPAM)
        </h4>
        <div className="bg-gray-800/30 rounded-lg p-3 space-y-2">
          <div className="flex justify-between text-sm">
            <span className="text-gray-400">IPv4 Pool</span>
            <span className="text-white font-mono">{data.ipam.ipv4_cidr}</span>
          </div>
          <div className="flex justify-between text-sm">
            <span className="text-gray-400">IPv6 Pool</span>
            <span className="text-white font-mono">{data.ipam.ipv6_cidr}</span>
          </div>
          <div className="flex justify-between text-sm">
            <span className="text-gray-400">Allocated Subnets</span>
            <span className="text-green-400 font-semibold">{data.ipam.allocated_count}</span>
          </div>
        </div>

        {data.ipam.allocations.length > 0 && (
          <div className="mt-3 max-h-32 overflow-y-auto">
            <table className="w-full text-xs">
              <thead className="text-gray-500">
                <tr>
                  <th className="text-left py-1">IP</th>
                  <th className="text-left py-1">Resource</th>
                </tr>
              </thead>
              <tbody className="text-gray-300">
                {data.ipam.allocations.map((alloc, i) => (
                  <tr key={i} className="border-t border-gray-700/50">
                    <td className="py-1 font-mono">{alloc.ip}</td>
                    <td className="py-1">{alloc.resource_name}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* PortAM Section */}
      <div>
        <h4 className="text-sm font-medium text-gray-300 mb-3 flex items-center gap-2">
          <Activity size={14} />
          Port Allocation Manager (PortAM)
        </h4>
        <div className="bg-gray-800/30 rounded-lg p-3 space-y-2">
          <div className="flex justify-between text-sm">
            <span className="text-gray-400">Port Range</span>
            <span className="text-white font-mono">{data.portam.min_port} - {data.portam.max_port}</span>
          </div>
          <div className="flex justify-between text-sm">
            <span className="text-gray-400">Used / Available</span>
            <span className="text-white">
              <span className="text-yellow-400">{data.portam.used_count}</span>
              {' / '}
              <span className="text-green-400">{data.portam.available_count}</span>
            </span>
          </div>
          {/* Progress bar */}
          <div className="mt-2">
            <div className="h-2 bg-gray-700 rounded-full overflow-hidden">
              <div 
                className="h-full bg-gradient-to-r from-purple-500 to-blue-500 transition-all"
                style={{ width: `${Math.min(portUsagePercent, 100)}%` }}
              />
            </div>
            <div className="text-xs text-gray-500 mt-1 text-right">
              {portUsagePercent.toFixed(2)}% used
            </div>
          </div>
        </div>

        {data.portam.allocations.length > 0 && (
          <div className="mt-3 max-h-40 overflow-y-auto">
            <table className="w-full text-xs">
              <thead className="text-gray-500 sticky top-0 bg-gray-900">
                <tr>
                  <th className="text-left py-1">Port</th>
                  <th className="text-left py-1">Node</th>
                  <th className="text-left py-1">Type</th>
                  <th className="text-left py-1">Resource</th>
                </tr>
              </thead>
              <tbody className="text-gray-300">
                {data.portam.allocations.map((alloc, i) => (
                  <tr key={i} className="border-t border-gray-700/50">
                    <td className="py-1 font-mono text-blue-400">{alloc.port}</td>
                    <td className="py-1">{alloc.node_name || `Node #${alloc.node_id}`}</td>
                    <td className="py-1">
                      <span className={`px-1.5 py-0.5 rounded text-xs ${
                        alloc.type === 'tunnel' ? 'bg-green-500/20 text-green-400' : 'bg-purple-500/20 text-purple-400'
                      }`}>
                        {alloc.type}
                      </span>
                    </td>
                    <td className="py-1">{alloc.resource_name}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
