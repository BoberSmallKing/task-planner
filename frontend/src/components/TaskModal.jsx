import { useMemo, useState } from 'react'
import { ArrowRight, CalendarDays, CheckCircle2, ChevronLeft, ChevronRight, Clock3, Droplet, Expand, ListTodo, MoreVertical, RotateCcw, Trash2 } from 'lucide-react'
import { dateKey } from '../utils/date.js'
import { displayTime, timeChoices } from '../utils/time.js'

const monthLabel = new Intl.DateTimeFormat('ru-RU', { month: 'long', year: 'numeric' })
const monthShort = new Intl.DateTimeFormat('ru-RU', { month: 'short' })
const colors = ['#d52d58', '#ffeb4d', '#aeb8ff', '#9de8d4', '#f0a8d8', '#ffd0a5', '#ffffff']

export default function TaskModal({ task, initialDate, onClose, onDelete, onSave }) {
  const initial = task ? new Date(task.startAt) : null
  const [form, setForm] = useState({ title: task?.title || '', description: task?.description || '', date: initial ? dateKey(initial) : initialDate, time: initial ? `${String(initial.getHours()).padStart(2, '0')}:${String(initial.getMinutes()).padStart(2, '0')}` : '09:00', color: task?.color || '#ffffff' })
  const [error, setError] = useState('')
  const [expanded, setExpanded] = useState(false)
  const [picker, setPicker] = useState('')
  const [menuOpen, setMenuOpen] = useState(false)
  const [visibleMonth, setVisibleMonth] = useState(() => { const d = new Date(`${form.date}T12:00:00`); return new Date(d.getFullYear(), d.getMonth(), 1) })
  const calendarDays = useMemo(() => {
    const first = new Date(visibleMonth.getFullYear(), visibleMonth.getMonth(), 1)
    const start = new Date(first)
    start.setDate(1 - ((first.getDay() + 6) % 7))
    return Array.from({ length: 42 }, (_, i) => { const d = new Date(start); d.setDate(start.getDate() + i); return d })
  }, [visibleMonth])
  const selectedDate = new Date(`${form.date}T12:00:00`)
  const today = new Date()
  const dateText = `${form.date === dateKey(today) ? 'Сегодня, ' : ''}${monthShort.format(selectedDate)} ${selectedDate.getDate()}`
  const update = (name, value) => setForm(prev => ({ ...prev, [name]: value }))
  const submit = async event => {
    event.preventDefault()
    try { await onSave(form) } catch (e) { setError(e.message || 'Не удалось сохранить задачу.') }
  }
  const chooseDate = day => { update('date', dateKey(day)); setPicker('') }
  const dismiss = async () => {
    if (form.title.trim() || form.description.trim()) {
      try { await onSave({ ...form, title: form.title.trim() || 'Без названия' }) } catch (e) { setError(e.message || 'Не удалось сохранить задачу.') }
      return
    }
    onClose()
  }

  return <div className="overlay" onMouseDown={e => e.target === e.currentTarget && dismiss()}>
    <form className={`note-modal ${expanded ? 'expanded' : ''}`} style={{ '--note-color': form.color }} onSubmit={submit}>
      <div className="note-topline">
        <div className="date-time-controls">
          <button type="button" className={`date-trigger ${picker === 'date' ? 'active' : ''}`} onClick={() => setPicker(picker === 'date' ? '' : 'date')}><CalendarDays size={16}/>{dateText}</button>
          <button type="button" className={`time-trigger ${picker === 'time' ? 'active' : ''}`} onClick={() => setPicker(picker === 'time' ? '' : 'time')}><Clock3 size={16}/></button>
        </div>
        <button type="button" className="expand-trigger" onClick={() => setExpanded(!expanded)}><Expand size={14}/>{expanded ? 'Свернуть' : 'Развернуть'}</button>
      </div>
      {picker === 'date' && <div className="date-popover">
        <div className="calendar-popover-head"><button type="button" onClick={() => setVisibleMonth(new Date(visibleMonth.getFullYear(), visibleMonth.getMonth() - 1, 1))}><ChevronLeft size={16}/></button><strong>{monthLabel.format(visibleMonth).replace(/\s*г\./, '')}</strong><button type="button" onClick={() => setVisibleMonth(new Date(visibleMonth.getFullYear(), visibleMonth.getMonth() + 1, 1))}><ChevronRight size={16}/></button></div>
        <div className="calendar-popover-grid">{['П','В','С','Ч','П','С','В'].map((d,i)=><span className="weekday" key={`${d}-${i}`}>{d}</span>)}{calendarDays.map(day=><button type="button" key={dateKey(day)} className={`${day.getMonth() !== visibleMonth.getMonth() ? 'other-month' : ''} ${dateKey(day) === form.date ? 'chosen' : ''}`} onClick={() => chooseDate(day)}>{day.getDate()}</button>)}</div>
      </div>}
      {picker === 'time' && <div className="time-popover">{timeChoices.map(time=><button type="button" key={time} className={form.time === time ? 'chosen' : ''} onClick={() => { update('time', time); setPicker('') }}>{displayTime(time)}</button>)}</div>}
      {picker === 'color' && <div className="color-popover">{colors.map(color=><button type="button" key={color} className={form.color === color ? 'selected' : ''} style={{ '--swatch-color': color }} onClick={() => { update('color', color); setPicker('') }} aria-label={`Выбрать цвет ${color}`} />)}</div>}
      <div className="title-line"><input className="note-title" autoFocus required maxLength="120" placeholder="Название задачи" value={form.title} onChange={e => update('title', e.target.value)} /><button type="submit" className="done-trigger" title="Сохранить задачу"><CheckCircle2 size={20}/></button></div>
      <textarea className="note-description" rows={expanded ? 8 : 4} maxLength="1000" placeholder="Добавьте заметку..." value={form.description} onChange={e => update('description', e.target.value)} />
      {error && <p className="form-error">{error}</p>}
      <div className="note-toolbar">
        <button type="button" title="Дата" onClick={() => setPicker(picker === 'date' ? '' : 'date')}><ListTodo size={18}/></button>
        <button type="button" title="Сбросить время" onClick={() => update('time', '09:00')}><RotateCcw size={18}/></button>
        <button type="button" className={picker === 'color' ? 'tool-active' : ''} title="Цвет задачи" onClick={() => setPicker(picker === 'color' ? '' : 'color')}><Droplet size={18}/></button>
        <button type="submit" title="Сохранить"><CheckCircle2 size={18}/></button>
        <button type="button" title="Дополнительные действия" className={menuOpen ? 'tool-active' : ''} onClick={() => setMenuOpen(!menuOpen)}><MoreVertical size={18}/></button>
      </div>
      {menuOpen && <div className="task-menu">
        <button type="button" onClick={() => { setExpanded(true); setMenuOpen(false) }}><ArrowRight size={16}/> Развернуть задачу</button>
        {task && <button type="button" className="delete-action" onClick={() => onDelete(task.id).catch(e => setError(e.message || 'Не удалось удалить задачу.'))}><Trash2 size={16}/> Удалить</button>}
      </div>}
    </form>
  </div>
}
