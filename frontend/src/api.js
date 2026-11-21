const API_BASE = '/api'

async function request(path, options = {}) {
  const response = await fetch(`${API_BASE}${path}`, options)
  if (!response.ok) {
    const data = await response.json().catch(() => ({}))
    throw new Error(data.error || 'Ошибка запроса к серверу.')
  }
  if (response.status === 204) return null
  return response.json()
}

const json = body => ({
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(body),
})

export const getTasks = () => request('/tasks')
export const createTask = task => request('/tasks', json(task))
export const updateTask = (id, changes) => request(`/tasks/${id}`, { ...json(changes), method: 'PATCH' })
export const removeTask = id => request(`/tasks/${id}`, { method: 'DELETE' })
