import { ChevronLeft, ChevronRight, Plus } from 'lucide-react'

const titleDate = new Intl.DateTimeFormat('ru-RU', { month: 'long', year: 'numeric' })
const weekNumber = date => {
  const d = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()))
  d.setUTCDate(d.getUTCDate() + 4 - (d.getUTCDay() || 7))
  return Math.ceil((((d - new Date(Date.UTC(d.getUTCFullYear(), 0, 1))) / 86400000) + 1) / 7)
}

export default function Header({ anchor, onAdd, onPrevious, onNext }) {
  return <header className="topbar">
    <h1>{titleDate.format(anchor).replace(/\s*г\./, '')} <span>№{weekNumber(anchor)}</span></h1>
    <nav className="controls" aria-label="Управление календарём">
      <button className="add-button" onClick={onAdd} aria-label="Добавить задачу"><Plus size={20} /></button>
      <button className="nav-button" onClick={onPrevious} aria-label="Предыдущая неделя"><ChevronLeft size={21} /></button>
      <button className="nav-button" onClick={onNext} aria-label="Следующая неделя"><ChevronRight size={21} /></button>
    </nav>
  </header>
}
