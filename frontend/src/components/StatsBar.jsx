import { useEffect, useState } from 'react'
import { api } from '../api'

const STATUS_CONFIG = {
  pending:    { label: 'Pending',    color: '#fbbf24' },
  processing: { label: 'Processing', color: '#7c6af7' },
  completed:  { label: 'Completed',  color: '#34d399' },
  failed:     { label: 'Failed',     color: '#f87171' },
  dead:       { label: 'Dead',       color: '#6b7280' },
}

export default function StatsBar() {
  const [stats, setStats] = useState(null)
  const [queues, setQueues] = useState(null)

  useEffect(() => {
    const load = async () => {
      try {
        const data = await api.getStats()
        setStats(data.db)
        setQueues(data.redis)
      } catch (e) { console.error(e) }
    }
    load()
    const id = setInterval(load, 3000)
    return () => clearInterval(id)
  }, [])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(130px, 1fr))',
        gap: 12,
      }}>
        {Object.entries(STATUS_CONFIG).map(([key, { label, color }]) => (
          <StatCard
            key={key}
            label={label}
            value={stats ? (stats[key] ?? 0) : '—'}
            color={color}
          />
        ))}
        <StatCard label="Total" value={stats?.total ?? '—'} color="#60a5fa" />
      </div>

      {queues && (
        <div style={{
          background: 'var(--bg2)',
          border: '1px solid var(--border)',
          borderRadius: 'var(--radius)',
          padding: '12px 16px',
          display: 'flex',
          gap: 24,
          alignItems: 'center',
          flexWrap: 'wrap',
        }}>
          <span style={{ color: 'var(--text3)', fontSize: 12, fontFamily: 'var(--mono)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>
            Redis queues
          </span>
          {Object.entries(queues).map(([key, count]) => (
            <div key={key} style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <span style={{ color: 'var(--text2)', fontSize: 12, fontFamily: 'var(--mono)' }}>{key}</span>
              <span style={{
                background: count > 0 ? 'rgba(124,106,247,0.15)' : 'var(--bg3)',
                color: count > 0 ? 'var(--accent2)' : 'var(--text3)',
                fontFamily: 'var(--mono)',
                fontSize: 12,
                padding: '2px 8px',
                borderRadius: 20,
                fontWeight: 500,
              }}>{count}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function StatCard({ label, value, color }) {
  return (
    <div style={{
      background: 'var(--bg2)',
      border: '1px solid var(--border)',
      borderRadius: 'var(--radius)',
      padding: '14px 16px',
    }}>
      <div style={{ fontSize: 11, color: 'var(--text3)', textTransform: 'uppercase', letterSpacing: '0.06em', marginBottom: 6 }}>
        {label}
      </div>
      <div style={{ fontSize: 28, fontWeight: 600, color, fontFamily: 'var(--mono)' }}>
        {value}
      </div>
    </div>
  )
}
