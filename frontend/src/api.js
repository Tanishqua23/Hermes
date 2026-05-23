const BASE = '/api'

async function request(path, options = {}) {
  const res = await fetch(BASE + path, {
    headers: { 'Content-Type': 'application/json', ...options.headers },
    ...options,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || 'Request failed')
  }
  return res.json()
}

export const api = {
  getStats: () => request('/stats'),
  getJobTypes: () => request('/job-types'),
  listJobs: (params = {}) => {
    const qs = new URLSearchParams(
      Object.fromEntries(Object.entries(params).filter(([, v]) => v))
    ).toString()
    return request('/jobs' + (qs ? '?' + qs : ''))
  },
  getJob: (id) => request('/jobs/' + id),
  enqueueJob: (body) => request('/jobs', { method: 'POST', body: JSON.stringify(body) }),
}
