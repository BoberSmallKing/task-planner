import { useEffect, useMemo, useState } from 'react'
import Header from './components/Header.jsx'
import WeekCalendar from './components/WeekCalendar.jsx'
import TaskModal from './components/TaskModal.jsx'
import AccountModal from './components/AccountModal.jsx'
import { clearSession, createTask, getStoredUser, getTasks, logoutUser, removeTask, saveSession, updateTask as patchTask } from './api.js'
import { dateAtTime, dateKey, getWeekDays, startOfWeek } from './utils/date.js'
const today = new Date()

export default function App() {
  const [anchor, setAnchor] = useState(startOfWeek(today))
  const [tasks, setTasks] = useState([])
  const [modal, setModal] = useState(false)
  const [editingTask, setEditingTask] = useState(null)
  const [user, setUser] = useState(getStoredUser)
  const [accountOpen, setAccountOpen] = useState(false)
  const [error, setError] = useState('')
  const days = useMemo(() => getWeekDays(anchor), [anchor])

  useEffect(() => {
    if (!user) { setTasks([]); return }
    let active = true
    getTasks().then(result => { if (active) setTasks(result) })
      .catch(() => { if (active) setError('Не удалось загрузить задачи. Проверьте соединение или войдите снова.') })
    return () => { active = false }
  }, [user])

  useEffect(() => {
    const onSessionExpired = () => {
      setUser(null); setTasks([]); setModal(false); setEditingTask(null); setAccountOpen(true)
    }
    window.addEventListener('taskplanner:session-expired', onSessionExpired)
    return () => window.removeEventListener('taskplanner:session-expired', onSessionExpired)
  }, [])

  const changeWeek = step => { const d = new Date(anchor); d.setDate(d.getDate() + step * 7); setAnchor(d) }
  const saveTask = async form => {
    const payload = { title: form.title.trim(), description: form.description.trim(), startAt: dateAtTime(form.date, form.time).toISOString(), color: form.color }
    if (editingTask) {
      await patchTask(editingTask.id, payload)
      setTasks(prev => prev.map(task => task.id === editingTask.id ? { ...task, ...payload } : task))
    } else {
      const task = await createTask(payload)
      setTasks(prev => [...prev, { ...task, done: false }])
    }
    const date = new Date(`${form.date}T12:00:00`)
    setAnchor(startOfWeek(date)); setModal(false); setEditingTask(null); setError('')
  }
  const updateTask = async (task, payload) => {
    await patchTask(task.id, payload)
    setTasks(prev => prev.map(item => item.id === task.id ? { ...item, ...payload, ...(payload.startAt ? { startAt: payload.startAt } : {}) } : item))
  }
  const toggleTask = async task => {
    try {
      await patchTask(task.id, { done: !task.done })
      setTasks(prev => prev.map(t => t.id === task.id ? { ...t, done: !t.done } : t))
    } catch (e) { setError(e.message) }
  }
  const deleteTask = async id => {
    await removeTask(id)
    setTasks(prev => prev.filter(t => t.id !== id))
  }
  const moveTask = async (task, date) => {
    const old = new Date(task.startAt)
    const moved = dateAtTime(date, `${String(old.getHours()).padStart(2, '0')}:${String(old.getMinutes()).padStart(2, '0')}`)
    try { await updateTask(task, { startAt: moved.toISOString() }) } catch (e) { setError(e.message) }
  }
  const startTask = () => {
    setError('')
    if (!user) { setAccountOpen(true); return }
    setEditingTask(null); setModal(true)
  }
  const logout = async () => {
    try { await logoutUser() } catch { /* Remove the local session even when offline. */ }
    clearSession(); setUser(null); setTasks([]); setAccountOpen(false); setModal(false)
  }

  return <main className="planner">
    <Header anchor={anchor} onAdd={startTask} onUser={() => setAccountOpen(true)} onPrevious={() => changeWeek(-1)} onNext={() => changeWeek(1)} />
    {error && <p className="connection-error">{error}</p>}
    <WeekCalendar days={days} tasks={tasks} onToggle={toggleTask} onEdit={task => { setEditingTask(task); setModal(true) }} onMove={moveTask} />
    {modal && <TaskModal task={editingTask} initialDate={dateKey(today >= anchor && today <= days[6] ? today : anchor)} onClose={() => { setModal(false); setEditingTask(null) }} onDelete={async id => { await deleteTask(id); setModal(false); setEditingTask(null) }} onSave={saveTask} />}
    {accountOpen && <AccountModal user={user} onClose={() => setAccountOpen(false)} onAuthenticated={session => { saveSession(session); setUser(session.user); setAccountOpen(false); setError('') }} onLogout={logout} />}
  </main>
}
