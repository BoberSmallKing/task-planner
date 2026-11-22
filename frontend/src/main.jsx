import React from 'react'
import { createRoot } from 'react-dom/client'
import App from './App.jsx'
import './styles/global.css'
import './styles/task-modal.css'
import './styles/task-modal-extra.css'
import './styles/task-palette.css'
import './styles/week-calendar.css'
import './styles/account.css'
import './styles/auth.css'

createRoot(document.getElementById('root')).render(<React.StrictMode><App /></React.StrictMode>)
