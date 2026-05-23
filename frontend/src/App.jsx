import { useState } from 'react'
import StatsBar from './components/StatsBar'
import EnqueueForm from './components/EnqueueForm'
import JobTable from './components/JobTable'
import JobDetail from './components/JobDetail'

export default function App() {
  const [refreshKey, setRefreshKey] = useState(0)
  const [selectedJobId, setSelectedJobId] = useState(null)

  return (
    <div style={{ minHeight: '100vh', background: 'var(--bg)' }}>
      {/* Header */}
      <header style={{
        borderBottom: '1px solid var(--border)',
        padding: '0 40px',
        height: 56,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        background: 'var(--bg2)',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <span style={{ fontSize: 20 }}>⚡</span>
          <span style={{ fontWeight: 700, fontSize: 16, letterSpacing: '-0.02em' }}>Hermes</span>
          <span style={{
            background: 'rgba(124,106,247,0.15)',
            color: 'var(--accent2)',
            fontSize: 10,
            fontFamily: 'var(--mono)',
            padding: '2px 8px',
            borderRadius: 4,
            fontWeight: 500,
          }}>distributed task queue</span>
        </div>
        <div style={{ fontSize: 12, color: 'var(--text3)', fontFamily: 'var(--mono)' }}>
          localhost:8080
        </div>
      </header>

      {/* Main layout */}
      <main style={{ padding: '32px 40px', display: 'flex', flexDirection: 'column', gap: 24, maxWidth: 1400, margin: '0 auto' }}>
        <StatsBar />

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 420px', gap: 24, alignItems: 'start' }}>
          <JobTable
            refresh={refreshKey}
            onSelectJob={setSelectedJobId}
          />
          <EnqueueForm onEnqueued={() => setRefreshKey(k => k + 1)} />
        </div>
      </main>

      <JobDetail jobId={selectedJobId} onClose={() => setSelectedJobId(null)} />
    </div>
  )
}
