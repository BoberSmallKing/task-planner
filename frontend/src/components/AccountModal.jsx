import { useState } from 'react'
import { ArrowLeft, Search, Settings, Share2, X } from 'lucide-react'
import { loginUser, lookupUser, registerUser } from '../api.js'
import Mascot from './Mascot.jsx'

export default function AccountModal({ user, onClose, onAuthenticated, onLogout }) {
  const [stage, setStage] = useState('login')
  const [accountExists, setAccountExists] = useState(false)
  const [login, setLogin] = useState('')
  const [password, setPassword] = useState('')
  const [passwordConfirmation, setPasswordConfirmation] = useState('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [busy, setBusy] = useState(false)

  const submit = async event => {
    event.preventDefault()
    setError('')
    if (stage === 'login') {
      setBusy(true)
      try {
        const result = await lookupUser({ login: login.trim() })
        setAccountExists(result.exists)
        setStage('password')
      } catch (e) { setError(e.message || 'Не удалось проверить логин.') }
      finally { setBusy(false) }
      return
    }
    if (!accountExists) {
      if ([...password].length < 8 || !/\p{N}/u.test(password)) {
        setError('Пароль должен содержать минимум 8 символов и хотя бы одну цифру.')
        return
      }
      if (password !== passwordConfirmation) {
        setError('Пароли не совпадают.')
        return
      }
    }
    setBusy(true)
    try {
      const session = !accountExists
        ? await registerUser({ login: login.trim(), password, passwordConfirmation })
        : await loginUser({ login: login.trim(), password })
      onAuthenticated(session)
    } catch (e) {
      setError(e.message || 'Не удалось выполнить вход.')
    } finally { setBusy(false) }
  }

  if (user) return <div className="account-backdrop" onMouseDown={event => event.target === event.currentTarget && onClose()}>
    <section className="account-panel" aria-label="Профиль пользователя">
      <div className="account-heading"><strong>{user.login}</strong><button onClick={onClose} aria-label="Закрыть"><X size={17}/></button></div>
      <label className="account-search"><Search size={17}/><input placeholder="Поиск" onFocus={() => setNotice('Поиск скоро появится.')} /><kbd>Ctrl</kbd><kbd>F</kbd></label>
      <div className="account-actions">
        <button onClick={() => setNotice('Поделиться профилем скоро станет доступно.')}><Share2 size={19}/><span>Поделиться</span></button>
        <button onClick={() => setNotice('Настройки скоро появятся.')}><Settings size={19}/><span>Настройки</span></button>
      </div>
      {notice && <p className="account-notice">{notice}</p>}
      <button className="account-logout" onClick={onLogout}>Выйти из аккаунта</button>
    </section>
  </div>

  return <div className="overlay account-overlay" onMouseDown={event => event.target === event.currentTarget && onClose()}>
    <form className="account-auth" onSubmit={submit}>
      <div className="auth-heading">{stage === 'password' && <button className="auth-back" type="button" onClick={() => { setStage('login'); setPassword(''); setPasswordConfirmation(''); setError('') }} aria-label="Изменить логин"><ArrowLeft size={17}/></button>}<button type="button" onClick={onClose} aria-label="Закрыть"><X size={18}/></button></div>
      <Mascot />
      <h2>Привет!</h2>
      {stage === 'login'
        ? <label className="auth-field"><input autoFocus required type="email" autoComplete="email" value={login} onChange={e => setLogin(e.target.value)} placeholder="Электронная почта" /></label>
        : <>
          <div className="auth-login-value">{login}</div>
          <label className="auth-field"><input autoFocus required type="password" autoComplete={accountExists ? 'current-password' : 'new-password'} value={password} onChange={e => setPassword(e.target.value)} placeholder="Пароль" /></label>
          {!accountExists && <label className="auth-field"><input required type="password" autoComplete="new-password" value={passwordConfirmation} onChange={e => setPasswordConfirmation(e.target.value)} placeholder="Повторите пароль" /></label>}
        </>}
      {error && <p className="auth-error">{error}</p>}
      <button className="auth-submit" type="submit" disabled={busy}>{busy ? 'Подождите…' : stage === 'login' ? 'Продолжить' : accountExists ? 'Войти' : 'Создать аккаунт'}</button>
      {stage === 'password' && !accountExists && <p className="password-hint">Минимум 8 символов и одна цифра.</p>}
    </form>
  </div>
}
