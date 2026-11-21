export const dateKey = date => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`

export function startOfWeek(value) {
  const date = new Date(value)
  date.setHours(0, 0, 0, 0)
  date.setDate(date.getDate() - ((date.getDay() + 6) % 7))
  return date
}

export function getWeekDays(start) {
  return Array.from({ length: 7 }, (_, index) => {
    const date = new Date(start)
    date.setDate(date.getDate() + index)
    return date
  })
}

export function dateAtTime(date, time) {
  return new Date(`${date}T${time}:00`)
}

export function isoWeekNumber(date) {
  const utcDate = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()))
  utcDate.setUTCDate(utcDate.getUTCDate() + 4 - (utcDate.getUTCDay() || 7))
  return Math.ceil((((utcDate - new Date(Date.UTC(utcDate.getUTCFullYear(), 0, 1))) / 86400000) + 1) / 7)
}
