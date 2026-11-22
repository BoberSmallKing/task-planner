import { Check } from 'lucide-react'
import { dateKey } from '../utils/date.js'

const dayName = new Intl.DateTimeFormat('ru-RU', { weekday: 'short' })
const monthName = new Intl.DateTimeFormat('ru-RU', { month: 'short' })

export default function WeekCalendar({ days, tasks, onToggle, onEdit, onMove }) {
  const today = dateKey(new Date())
  return <section className="week-grid" aria-label="Календарь на неделю">
    {days.map(day => {
      const key = dateKey(day)
      const dayTasks = tasks.filter(task => dateKey(new Date(task.startAt)) === key).sort((a, b) => new Date(a.startAt) - new Date(b.startAt))
      return <section className={`day-column ${key === today ? 'is-today' : ''}`} key={key} onDragOver={event => event.preventDefault()} onDrop={event => { event.preventDefault(); const id = Number(event.dataTransfer.getData('text/plain')); const task = tasks.find(item => item.id === id); if (task && dateKey(new Date(task.startAt)) !== key) onMove(task, key) }}>
        <h2 className="day-heading"><strong>{monthName.format(day)} {day.getDate()}</strong><span>{key === today ? 'Сегодня' : dayName.format(day).replace('.', '')}</span></h2>
        <div className="day-lines">
          {dayTasks.map(task => <article className={`task-row ${task.done ? 'is-done' : ''}`} style={{ '--task-color': task.color || '#fff' }} key={task.id} draggable onDragStart={event => { event.dataTransfer.setData('text/plain', String(task.id)); event.dataTransfer.effectAllowed = 'move' }} onClick={() => onEdit(task)} title="Нажмите, чтобы изменить · перетащите, чтобы перенести">
            <div className="task-content"><span className="task-title">{task.title}</span><span className="task-time">{new Date(task.startAt).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })}</span></div>
            <button className="task-check" onClick={event => { event.stopPropagation(); onToggle(task) }} aria-label={task.done ? 'Вернуть задачу' : 'Завершить задачу'}><Check size={12} /></button>
          </article>)}
          {Array.from({ length: Math.max(8, 12 - dayTasks.length) }, (_, i) => <div className="ruled-row" key={`line-${i}`} />)}
        </div>
      </section>
    })}
  </section>
}
