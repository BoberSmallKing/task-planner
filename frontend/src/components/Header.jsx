import { ChevronLeft, ChevronRight, Plus, UserRound } from 'lucide-react'
import { isoWeekNumber } from '../utils/date.js'

const titleDate = new Intl.DateTimeFormat('ru-RU', { month: 'long', year: 'numeric' })
export default function Header({ anchor, onAdd, onUser, onPrevious, onNext }) {
  return <header className="topbar">
    <h1>{titleDate.format(anchor).replace(/\s*г\./, '')} <span>№{isoWeekNumber(anchor)}</span></h1>
    <nav className="controls" aria-label="Управление календарём">
      <button className="add-button" onClick={onAdd} aria-label="Добавить задачу"><Plus size={20} /></button>
      <button className="user-button" onClick={onUser} aria-label="Войти или открыть профиль"><UserRound size={18} /></button>
      <button className="nav-button" onClick={onPrevious} aria-label="Предыдущая неделя"><ChevronLeft size={21} /></button>
      <button className="nav-button" onClick={onNext} aria-label="Следующая неделя"><ChevronRight size={21} /></button>
    </nav>
  </header>
}
