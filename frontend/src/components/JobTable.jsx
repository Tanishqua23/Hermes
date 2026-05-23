import { useEffect, useState, useCallback } from 'react'
import { api } from '../api'

const STATUS_COLOR = {
  pending:    '#fbbf24',
  processing: '#7c6af7',
  completed:  '#34d399',
  failed:     '#f87171',
  dead:       '#6b7280',
}

const STATUSES = ['', 'pending', 'processing', 'completed', 'failed', 'dead']

export default function JobTable({ refresh, onSelectJob }) {
  const [jobs, setJobs] = useState([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState('')
  const [loading, setLoading] = useState(false)
  const LIMIT = 15

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const data = await api.listJobs({ status: statusFilter, page, limit: LIMIT })
      setJobs(data.jobs || [])
      setTotal(data.total)
    } catch (e) { console.error(e) }
    finally { setLoading(false) }
  }, [statusFilter, page])

  useEffect(() => { load() }, [load, refresh])
  useEffect(() => { const id = setInterval(load, 4000); return () => clearInterval(id) }, [load])

  const totalPages = Math.ceil(total / LIMIT)

  return (
    <div style={{
      background: 'var(--bg2)',
      border: '1px solid var(--border)',
      borderRadius: 'var(--radius-lg)',
      overflow: 'hidden',
    }}>
      {/* Toolbar */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 12,
        padding: '14px 20px',
        borderBottom: '1px solid var(--border)',
      }}>
        <h2 style={{ fontSize: 15, fontWeight: 600, flex: 1 }}>Jobs</h2>
        <span style={{ fontSize: 12, color: 'var(--text3)' }}>{total} total</span>
        <select
          value={statusFilter}
          onChange={e => { setStatusFilter(e.target.value); setPage(1) }}
          style={{ fontSize: 12, padding: '5px 10px' }}
        >
          {STATUSES.map(s => <option key={s} value={s}>{s || 'All statuses'}</option>)}
        </select>
        <button onClick={load} style={{
          background: 'var(--bg3)', color: 'var(--text2)',
          padding: '5px 12px', borderRadius: 'var(--radius)', fontSize: 12,
        }}>
          Refresh
        </button>
      </div>

      {/* Table */}
      <div style={{ overflowX: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr style={{ borderBottom: '1px solid var(--border)' }}>
              {['Type', 'Status', 'Priority', 'Retries', 'Created', 'Duration'].map(h => (
                <th key={h} style={{ padding: '10px 16px', textAlign: 'left', fontSize: 11, color: 'var(--text3)', textTransform: 'uppercase', letterSpacing: '0.06em', fontWeight: 500 }}>
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {loading && jobs.length === 0 && (
              <tr><td colSpan={6} style={{ padding: '24px', textAlign: 'center', color: 'var(--text3)' }}>Loading...</td></tr>
            )}
            {!loading && jobs.length === 0 && (
              <tr><td colSpan={6} style={{ padding: '24px', textAlign: 'center', color: 'var(--text3)' }}>No jobs found</td></tr>
            )}
            {jobs.map(job => (
              <tr
                key={job.id}
                onClick={() => onSelectJob(job.id)}
                style={{
                  borderBottom: '1px solid var(--border)',
                  cursor: 'pointer',
                  transition: 'background 0.1s',
                }}
                onMouseEnter={e => e.currentTarget.style.background = 'var(--bg3)'}
                onMouseLeave={e => e.currentTarget.style.background = ''}
              >
                <td style={{ padding: '12px 16px' }}>
                  <span style={{ fontFamily: 'var(--mono)', fontSize: 12, color: 'var(--accent2)' }}>{job.type}</span>
                </td>
                <td style={{ padding: '12px 16px' }}>
                  <StatusPill status={job.status} />
                </td>
                <td style={{ padding: '12px 16px' }}>
                  <PriorityBadge priority={job.priority} />
                </td>
                <td style={{ padding: '12px 16px', fontFamily: 'var(--mono)', fontSize: 12, color: 'var(--text2)' }}>
                  {job.retry_count} / {job.max_retries}
                </td>
                <td style={{ padding: '12px 16px', fontSize: 12, color: 'var(--text2)' }}>
                  {timeAgo(job.created_at)}
                </td>
                <td style={{ padding: '12px 16px', fontFamily: 'var(--mono)', fontSize: 12, color: 'var(--text2)' }}>
                  {duration(job.started_at, job.finished_at)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', gap: 8, padding: '12px' }}>
          <PageBtn onClick={() => setPage(p => Math.max(1, p-1))} disabled={page === 1}>←</PageBtn>
          <span style={{ fontSize: 12, color: 'var(--text2)' }}>Page {page} of {totalPages}</span>
          <PageBtn onClick={() => setPage(p => Math.min(totalPages, p+1))} disabled={page === totalPages}>→</PageBtn>
        </div>
      )}
    </div>
  )
}

function StatusPill({ status }) {
  const color = STATUS_COLOR[status] || '#888'
  return (
    <span style={{
      background: color + '18',
      color,
      padding: '2px 10px',
      borderRadius: 20,
      fontSize: 11,
      fontWeight: 500,
      fontFamily: 'var(--mono)',
    }}>{status}</span>
  )
}

function PriorityBadge({ priority }) {
  const label = priority >= 10 ? 'high' : priority <= 1 ? 'low' : 'normal'
  const color = priority >= 10 ? '#f87171' : priority <= 1 ? '#6b7280' : '#60a5fa'
  return (
    <span style={{ fontSize: 11, color, fontFamily: 'var(--mono)' }}>{label} ({priority})</span>
  )
}

function PageBtn({ onClick, disabled, children }) {
  return (
    <button onClick={onClick} disabled={disabled} style={{
      background: 'var(--bg3)', color: disabled ? 'var(--text3)' : 'var(--text2)',
      padding: '4px 12px', borderRadius: 'var(--radius)', fontSize: 13,
      opacity: disabled ? 0.4 : 1, cursor: disabled ? 'default' : 'pointer',
    }}>{children}</button>
  )
}

function timeAgo(ts) {
  if (!ts) return '—'
  const seconds = Math.floor((Date.now() - new Date(ts)) / 1000)
  if (seconds < 60) return seconds + 's ago'
  if (seconds < 3600) return Math.floor(seconds / 60) + 'm ago'
  return Math.floor(seconds / 3600) + 'h ago'
}

function duration(start, end) {
  if (!start || !end) return '—'
  const ms = new Date(end) - new Date(start)
  if (ms < 1000) return ms + 'ms'
  return (ms / 1000).toFixed(1) + 's'
}
