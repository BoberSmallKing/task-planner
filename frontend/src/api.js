const API_BASE = '/api'
const ACCESS_TOKEN_KEY = 'taskplanner.accessToken'
const REFRESH_TOKEN_KEY = 'taskplanner.refreshToken'
const USER_KEY = 'taskplanner.user'
let refreshRequest = null

async function request(path, options = {}) {
  const send = () => {
    const token = localStorage.getItem(ACCESS_TOKEN_KEY)
    const headers = { ...options.headers, ...(token ? { Authorization: `Bearer ${token}` } : {}) }
    return fetch(`${API_BASE}${path}`, { ...options, headers })
  }
  let response = await send()
  const publicEndpoint = ['/user/lookup', '/user/register', '/user/login', '/user/refresh'].includes(path)
  if (response.status === 401 && !publicEndpoint && localStorage.getItem(REFRESH_TOKEN_KEY)) {
    try { await refreshAccessToken(); response = await send() } catch {
      clearSession()
      window.dispatchEvent(new Event('taskplanner:session-expired'))
    }
  }
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

export const registerUser = credentials => request('/user/register', json(credentials))
export const loginUser = credentials => request('/user/login', json(credentials))
export const lookupUser = credentials => request('/user/lookup', json(credentials))
export const logoutUser = () => request('/user/logout', json({}))

export function saveSession(session) {
  localStorage.setItem(ACCESS_TOKEN_KEY, session.accessToken)
  localStorage.setItem(REFRESH_TOKEN_KEY, session.refreshToken)
  localStorage.setItem(USER_KEY, JSON.stringify(session.user))
}

async function refreshAccessToken() {
  if (!refreshRequest) {
    refreshRequest = (async () => {
      const response = await fetch(`${API_BASE}/user/refresh`, {
        ...json({ refreshToken: localStorage.getItem(REFRESH_TOKEN_KEY) }),
      })
      if (!response.ok) throw new Error('Сессия истекла.')
      const pair = await response.json()
      saveSession({ ...pair, user: getStoredUser() })
      return pair.accessToken
    })().finally(() => { refreshRequest = null })
  }
  return refreshRequest
}

export function getStoredUser() {
  if (!localStorage.getItem(ACCESS_TOKEN_KEY) || !localStorage.getItem(REFRESH_TOKEN_KEY)) return null
  try { return JSON.parse(localStorage.getItem(USER_KEY) || 'null') } catch { return null }
}

export function clearSession() {
  localStorage.removeItem(ACCESS_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
  localStorage.removeItem('taskplanner.authToken')
}
