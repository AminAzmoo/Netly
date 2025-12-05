import { Tunnel } from '../../types'
import StatusBadge from '../common/StatusBadge'
import CardShell from '../common/CardShell'

interface TunnelsTableProps {
  tunnels: Tunnel[]
}

export default function TunnelsTable({ tunnels }: TunnelsTableProps) {
  return (
    <CardShell className="overflow-hidden p-0">
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-left text-sm text-gray-400">
          <thead className="border-b border-gray-800 bg-black/40 text-xs uppercase text-gray-500">
            <tr>
              <th className="px-4 py-3 font-medium">Name</th>
              <th className="px-4 py-3 font-medium">Path</th>
              <th className="px-4 py-3 font-medium">Type</th>
              <th className="px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3 font-medium">Latency</th>
              <th className="px-4 py-3 font-medium">Last Action</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-800">
            {tunnels.map((tunnel) => {
              const statusVariant =
              tunnel.status === 'Live'
                ? 'neonA'
                : tunnel.status === 'Configuring'
                ? 'warn'
                : tunnel.status === 'Error'
                ? 'error'
                : 'default'

              return (
                <tr key={tunnel.id} className="hover:bg-white/5 transition-colors">
                  <td className="px-4 py-3 font-medium text-white">{tunnel.name}</td>
                  <td className="px-4 py-3 font-mono text-xs">{tunnel.path}</td>
                  <td className="px-4 py-3">{tunnel.type}</td>
                  <td className="px-4 py-3">
                    <StatusBadge status={tunnel.status} variant={statusVariant} />
                  </td>
                  <td className="px-4 py-3 text-white">
                    {tunnel.latency > 0 ? `${tunnel.latency}ms` : '-'}
                  </td>
                  <td className="px-4 py-3 text-xs">
                    {tunnel.lastAction ? (
                      <div className="flex flex-col">
                        <span className="text-gray-300">{tunnel.lastAction}</span>
                        <span className="text-gray-600">{tunnel.lastActionTime}</span>
                      </div>
                    ) : (
                      '-'
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </CardShell>
  )
}
