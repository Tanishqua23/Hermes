import { useState, useEffect } from 'react'
import { api } from '../api'

const PAYLOADS = {
  send_email: { to: 'user@example.com', subject: 'Hello from Hermes', body: 'Test email' },
  resize_image: { source_url: 'https://example.com/photo.jpg', width: 800, height: 600 },
  send_notification: { user_id: 'user_123', message: 'Your report is ready', channel: 'push' },
  generate_report: { report_type: 'monthly_sales', start_date: '2025-01-01', end_date: '2025-01-31' },
  process_payment: { amount: 99.99, currency: 'USD', user_id: 'user_456' },
}

const PRIORITIES = [
  { value: 1, label: 'Low' },
  { value: 5, label: 'Normal' },
  { value: 10, label: 'High' },
]

export default function EnqueueForm({ onEnqueued }) {
  const [jobTypes, setJobTypes] = useState([])
  const [type, setType] = useState('send_email')
  const [priority, setPriority] = useState(5)
  const [maxRetries, setMaxRetries] = useState(3)
  const [payloadText, setPayloadText] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  useEffect(() => {
    api.getJobTypes().then(d => {
      setJobTypes(d.types.sort())
      if (d.types.length) setType(d.types[0])
    }).catch(() => {})
  }, [])

  useEffect(() => {
    setPayloadText(JSON.stringify(PAYLOADS[type] || {}, null, 2))
  }, [type])

  const handleSubmit = async () => {
    setError('')
    setSuccess('')
    let payload
    try {
      payload = JSON.parse(payloadText)
    } catch {
      setError('Invalid JSON payload')
      return
    }
    setLoading(true)
    try {
      const job = await api.enqueueJob({ type, payload, priority, max_retries: maxRetries })
      setSuccess(`Job enqueued: ${job.id}`)
      onEnqueued()
    } catch (e) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{
      background: 'var(--bg2)',
      border: '1px solid var(--border)',
      borderRadius: 'var(--radius-lg)',
      padding: '20px 24px',
    }}>
      <h2 style={{ fontSize: 15, fontWeight: 600, marginBottom: 18, color: 'var(--text)' }}>
        Enqueue job
      </h2>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 12, marginBottom: 14 }}>
        <div>
          <Label>Job type</Label>
          <select value={type} onChange={e => setType(e.target.value)} style={{ width: '100%' }}>
            {jobTypes.map(t => <option key={t} value={t}>{t}</option>)}
          </select>
        </div>
        <div>
          <Label>Priority</Label>
          <select value={priority} onChange={e => setPriority(Number(e.target.value))} style={{ width: '100%' }}>
            {PRIORITIES.map(p => <option key={p.value} value={p.value}>{p.label} ({p.value})</option>)}
          </select>
        </div>
        <div>
          <Label>Max retries</Label>
          <input
            type="number"
            value={maxRetries}
            onChange={e => setMaxRetries(Number(e.target.value))}
            min={0} max={10}
            style={{ width: '100%' }}
          />
        </div>
      </div>

      <div style={{ marginBottom: 14 }}>
        <Label>Payload (JSON)</Label>
        <textarea
          value={payloadText}
          onChange={e => setPayloadText(e.target.value)}
          rows={7}
          style={{ width: '100%', fontFamily: 'var(--mono)', fontSize: 12, resize: 'vertical' }}
        />
      </div>

      {error && <div style={{ color: 'var(--red)', fontSize: 13, marginBottom: 10 }}>{error}</div>}
      {success && <div style={{ color: 'var(--green)', fontSize: 13, fontFamily: 'var(--mono)', marginBottom: 10 }}>{success}</div>}

      <button
        onClick={handleSubmit}
        disabled={loading}
        style={{
          background: loading ? 'var(--bg3)' : 'var(--accent)',
          color: '#fff',
          padding: '9px 20px',
          borderRadius: 'var(--radius)',
          fontWeight: 600,
          fontSize: 13,
          opacity: loading ? 0.7 : 1,
        }}
      >
        {loading ? 'Enqueueing...' : 'Enqueue →'}
      </button>
    </div>
  )
}

function Label({ children }) {
  return (
    <div style={{ fontSize: 11, color: 'var(--text3)', textTransform: 'uppercase', letterSpacing: '0.06em', marginBottom: 6 }}>
      {children}
    </div>
  )
}
