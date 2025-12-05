import PageShell from '../components/layout/PageShell'
import GeneralSettingsCard from '../components/settings/GeneralSettingsCard'
import NetworkSettingsCard from '../components/settings/NetworkSettingsCard'
import IntegrationsCard from '../components/settings/IntegrationsCard'

export default function SettingsPage() {
  return (
    <PageShell
      title="Settings"
      subtitle="Configure system preferences and integrations"
    >
      <div className="settings-page-content">
        <GeneralSettingsCard />
        <NetworkSettingsCard />
        <IntegrationsCard />
        
        {/* Placeholder Security Card */}
        <div className="settings-security-card">
            <h3 className="settings-card-title">Security</h3>
            <p className="settings-card-subtitle">Security settings are managed by the organization administrator.</p>
        </div>
      </div>
    </PageShell>
  )
}
