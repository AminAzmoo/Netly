import { Link, useLocation } from 'react-router-dom'
import { 
  LayoutDashboard, 
  Server, 
  Network, 
  Activity, 
  Clock, 
  Settings, 
  Moon, 
  User 
} from 'lucide-react'

const tabs = [
  { name: 'Dashboard', path: '/dashboard', icon: LayoutDashboard },
  { name: 'Devices', path: '/devices', icon: Server },
  { name: 'Tunnels', path: '/tunnels', icon: Network },
  { name: 'Services', path: '/services', icon: Activity },
  { name: 'Timeline', path: '/timeline', icon: Clock },
  { name: 'Settings', path: '/settings', icon: Settings },
]

export default function HeaderDock() {
  const location = useLocation()

  return (
    <header className="header-dock">
      <div className="header-content">
        {/* Logo */}
        <div className="logo-container">
          <div className="logo-icon">
            <span className="logo-symbol">N</span>
          </div>
          <span className="logo-text">Netly</span>
        </div>

        {/* Tabs */}
        <nav className="nav-container">
          {tabs.map((tab) => {
            const isActive = location.pathname === tab.path
            const Icon = tab.icon
            return (
              <Link
                key={tab.path}
                to={tab.path}
                className={`nav-link ${
                  isActive
                    ? 'nav-link-active'
                    : 'nav-link-inactive'
                }`}
              >
                <Icon size={18} className={`${isActive ? "text-neon-a" : "text-muted"} icon-mr-2`} />
                {tab.name}
              </Link>
            )
          })}
        </nav>

        {/* Right side */}
        <div className="header-right-actions">
          <button className="header-icon-btn">
            <Moon size={20} />
          </button>
          <div className="user-avatar">
            <User size={20} className="text-muted" />
          </div>
        </div>
      </div>
    </header>
  )
}
