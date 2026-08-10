export const FULL_CALENDAR_WEEK_COUNT = 53
const MIN_VISIBLE_WEEK_COUNT = 12
const CALENDAR_AXIS_ALLOWANCE = 54
const TARGET_CELL_SIZE = 14
const MIN_CELL_SIZE = 12
const MAX_CELL_SIZE = 18
const DAY_IN_MS = 24 * 60 * 60 * 1000

const parseDay = (day: string): number => {
  const [year, month, date] = day.split('-').map(Number)
  if (!year || !month || !date) return Number.NaN
  return Date.UTC(year, month - 1, date)
}

const formatDay = (timestamp: number): string => {
  const date = new Date(timestamp)
  return [
    date.getUTCFullYear(),
    String(date.getUTCMonth() + 1).padStart(2, '0'),
    String(date.getUTCDate()).padStart(2, '0')
  ].join('-')
}

const startOfWeek = (timestamp: number): number => {
  const day = new Date(timestamp).getUTCDay()
  return timestamp - ((day + 6) % 7) * DAY_IN_MS
}

export const getCalendarWeekCount = (width: number): number => {
  if (!Number.isFinite(width) || width <= 0) return FULL_CALENDAR_WEEK_COUNT

  const fullCalendarWidth = CALENDAR_AXIS_ALLOWANCE + FULL_CALENDAR_WEEK_COUNT * MIN_CELL_SIZE
  if (width >= fullCalendarWidth) return FULL_CALENDAR_WEEK_COUNT

  const fittedWeeks = Math.floor((width - CALENDAR_AXIS_ALLOWANCE) / TARGET_CELL_SIZE)
  return Math.max(MIN_VISIBLE_WEEK_COUNT, Math.min(FULL_CALENDAR_WEEK_COUNT, fittedWeeks))
}

export const getCalendarCellSize = (width: number, height: number, weekCount: number): number => {
  if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) {
    return TARGET_CELL_SIZE
  }

  const normalizedWeekCount = Math.max(1, weekCount)
  const widthSize = Math.floor((width - CALENDAR_AXIS_ALLOWANCE) / normalizedWeekCount)
  const heightSize = Math.floor((height - 28) / 7)
  return Math.max(MIN_CELL_SIZE, Math.min(MAX_CELL_SIZE, widthSize, heightSize))
}

export const getVisibleCalendarStartDate = (
  startDate: string,
  endDate: string,
  weekCount: number
): string => {
  if (weekCount >= FULL_CALENDAR_WEEK_COUNT) return startDate

  const start = parseDay(startDate)
  const end = parseDay(endDate)
  if (!Number.isFinite(start) || !Number.isFinite(end)) return startDate

  const visibleStart = startOfWeek(end) - (Math.max(1, weekCount) - 1) * 7 * DAY_IN_MS
  return formatDay(Math.max(start, visibleStart))
}

export const getCalendarPageEndDate = (
  startDate: string,
  endDate: string,
  weekCount: number,
  page: number
): string => {
  if (page <= 0 || weekCount >= FULL_CALENDAR_WEEK_COUNT) return endDate

  const start = parseDay(startDate)
  const end = parseDay(endDate)
  if (!Number.isFinite(start) || !Number.isFinite(end)) return endDate

  const windowDays = Math.max(1, weekCount) * 7
  const earliestEnd = Math.min(end, start + (windowDays - 1) * DAY_IN_MS)
  const pageEnd = end - page * windowDays * DAY_IN_MS
  return formatDay(Math.min(end, Math.max(earliestEnd, pageEnd)))
}
