import { useEffect, useMemo, useState } from 'react'
import Header from './companents/Header.jsx'
import WeekCalendar from './companents/WeekCalendar.jsx'
import TaskModal from './companents/TaskModal.jsx'

const dateKey = d => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
const startOfWeek = value => { const d = new Date(value); d.setHours(0, 0, 0, 0); d.setDate(d.getDate() - ((d.getDay() + 6) % 7)); return d }
const weekDays = start => Array.from({ length: 7 }, (_, i) => { const d = new Date(start); d.setDate(d.getDate() + i); return d })
const today = new Date()

export default function App() {
  const [anchor, setAnchor] = useState(startOfWeek(today))
  const [tasks, setTasks] = useState([])
  const [modal, setModal] = useState(false)
  const [editingTask, setEditingTask] = useState(null)
  const [error, setError] = useState('')
  const days = useMemo(() => weekDays(anchor), [anchor])

  useEffect(() => {
    fetch('/api/tasks').then(r => { if (!r.ok) throw Error(); return r.json() }).then(setTasks)
      .catch(() => setError('Не удалось подключиться к серверу. Запустите backend и обновите страницу.'))
  }, [])

  const changeWeek = step => { const d = new Date(anchor); d.setDate(d.getDate() + step * 7); setAnchor(d) }
  const saveTask = async form => {
    const payload = { title: form.title.trim(), description: form.description.trim(), startAt: new Date(`${form.date}T${form.time}:00`).toISOString(), color: form.color }
    const response = await fetch(editingTask ? `/api/tasks/${editingTask.id}` : '/api/tasks', { method: editingTask ? 'PATCH' : 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) })
    if (!response.ok) throw Error('Не удалось сохранить задачу.')
    if (editingTask) setTasks(prev => prev.map(task => task.id === editingTask.id ? { ...task, ...payload } : task))
    else { const task = await response.json(); setTasks(prev => [...prev, { ...task, done: false }]) }
    const date = new Date(`${form.date}T12:00:00`)
    setAnchor(startOfWeek(date)); setModal(false); setEditingTask(null); setError('')
  }
  const updateTask = async (task, payload) => {
    const response = await fetch(`/api/tasks/${task.id}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) })
    if (!response.ok) throw Error('Не удалось обновить задачу.')
    setTasks(prev => prev.map(item => item.id === task.id ? { ...item, ...payload, ...(payload.startAt ? { startAt: payload.startAt } : {}) } : item))
  }
  const toggleTask = async task => {
    const response = await fetch(`/api/tasks/${task.id}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ done: !task.done }) })
    if (response.ok) setTasks(prev => prev.map(t => t.id === task.id ? { ...t, done: !t.done } : t))
  }
  const deleteTask = async id => {
    const response = await fetch(`/api/tasks/${id}`, { method: 'DELETE' })
    if (response.ok) setTasks(prev => prev.filter(t => t.id !== id))
  }
  const moveTask = async (task, date) => {
    const old = new Date(task.startAt)
    const moved = new Date(`${date}T${String(old.getHours()).padStart(2, '0')}:${String(old.getMinutes()).padStart(2, '0')}:00`)
    try { await updateTask(task, { startAt: moved.toISOString() }) } catch (e) { setError(e.message) }
  }

  return <main className="planner">
    <Header anchor={anchor} onAdd={() => { setError(''); setEditingTask(null); setModal(true) }} onPrevious={() => changeWeek(-1)} onNext={() => changeWeek(1)} />
    {error && <p className="connection-error">{error}</p>}
    <WeekCalendar days={days} tasks={tasks} onToggle={toggleTask} onDelete={deleteTask} onEdit={task => { setEditingTask(task); setModal(true) }} onMove={moveTask} />
    {modal && <TaskModal task={editingTask} initialDate={dateKey(today >= anchor && today <= days[6] ? today : anchor)} onClose={() => { setModal(false); setEditingTask(null) }} onSave={saveTask} />}
  </main>
}
