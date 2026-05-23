import { useEffect, useState } from 'react'
import { api } from '../api'

const STATUS_COLOR = {
  pending:    '#fbbf24',
  processing: '#7c6af7',
  completed:  '#34d399',
  failed:     '#f87171',
  dead:       '#6b7280',
}

export default function JobDetail({ jobId, onClose }) {
  const [job, setJob] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!jobId) return
    const load = async () => {
      try {
        const j = await api.getJob(jobId)
        setJob(j)
      } catch (e) { console.error(e) }
      finally { setLoading(false) }
    }
    load()
    const id = setInterval(load, 2000)
    return () => clearInterval(id)
  }, [jobId])

  if (!jobId) return null

  return (
    <div style={{
      position: 'fixed', inset: 0,
      background: 'rgba(0,0,0,0.7)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      zIndex: 100,
    }} onClick={onClose}>
      <div style={{
        background: 'var(--bg2)',
        border: '1px solid var(--border2)',
        borderRadius: 'var(--radius-lg)',
        padding: '28px 32px',
        width: '640px',
        maxWidth: '95vw',
        maxHeight: '85vh',
        overflow: 'auto',
      }} onClick={e => e.stopPropagation()}>
        {loading && <p style={{ color: 'var(--text2)' }}>Loading...</p>}
        {job && (
          <>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: 20 }}>
              <div>
                <div style={{ fontSize: 12, color: 'var(--text3)', fontFamily: 'var(--mono)', marginBottom: 4 }}>{job.id}</div>
                <div style={{ fontSize: 18, fontWeight: 600 }}>{job.type}</div>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                <StatusPill status={job.status} />
                <button onClick={onClose} style={{ background: 'var(--bg3)', color: 'var(--text2)', padding: '6px 12px', borderRadius: 'var(--radius)', fontSize: 12 }}>
                  Close
                </button>
              </div>
            </div>

            <Grid>
              <Field label="Priority" value={job.priority} />
              <Field label="Retries" value={`${job.retry_count} / ${job.max_retries}`} />
              <Field label="Created" value={fmt(job.created_at)} />
              <Field label="Started" value={fmt(job.started_at)} />
              <Field label="Finished" value={fmt(job.finished_at)} />
              <Field label="Duration" value={duration(job.started_at, job.finished_at)} />
            </Grid>

            {job.error && (
              <Section label="Error">
                <pre style={{ color: 'var(--red)', fontFamily: 'var(--mono)', fontSize: 12, whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
                  {job.error}
                </pre>
              </Section>
            )}

            <Section label="Payload">
              <JsonBlock data={job.payload} />
            </Section>

            {job.result && (
              <Section label="Result">
                <JsonBlock data={job.result} />
              </Section>
            )}
          </>
        )}
      </div>
    </div>
  )
}

function StatusPill({ status }) {
  const color = STATUS_COLOR[status] || '#888'
  return (
    <span style={{
      background: color + '22',
      color,
      border: `1px solid ${color}44`,
      padding: '3px 12px',
      borderRadius: 20,
      fontSize: 12,
      fontWeight: 500,
      fontFamily: 'var(--mono)',
    }}>{status}</span>
  )
}

function Grid({ children }) {
  return <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 12, marginBottom: 20 }}>{children}</div>
}

function Field({ label, value }) {
  return (
    <div style={{ background: 'var(--bg3)', borderRadius: 'var(--radius)', padding: '10px 14px' }}>
      <div style={{ fontSize: 11, color: 'var(--text3)', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 4 }}>{label}</div>
      <div style={{ fontSize: 13, fontFamily: 'var(--mono)', color: 'var(--text)' }}>{value ?? '—'}</div>
    </div>
  )
}

function Section({ label, children }) {
  return (
    <div style={{ marginBottom: 16 }}>
      <div style={{ fontSize: 11, color: 'var(--text3)', textTransform: 'uppercase', letterSpacing: '0.06em', marginBottom: 8 }}>{label}</div>
      <div style={{ background: 'var(--bg3)', borderRadius: 'var(--radius)', padding: '12px 14px' }}>{children}</div>
    </div>
  )
}

function JsonBlock({ data }) {
  return (
    <pre style={{ fontFamily: 'var(--mono)', fontSize: 12, color: 'var(--text2)', whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
      {JSON.stringify(data, null, 2)}
    </pre>
  )
}

function fmt(ts) {
  if (!ts) return '—'
  return new Date(ts).toLocaleString()
}

function duration(start, end) {
  if (!start || !end) return '—'
  const ms = new Date(end) - new Date(start)
  if (ms < 1000) return ms + 'ms'
  return (ms / 1000).toFixed(1) + 's'
}
