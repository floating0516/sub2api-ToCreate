import { describe, expect, it } from 'vitest'

import {
  getCalendarCellSize,
  getCalendarPageEndDate,
  getCalendarWeekCount,
  getVisibleCalendarStartDate
} from '../dashboardCalendarLayout'

describe('dashboard calendar responsive layout', () => {
  it('shows the full year when there is enough width', () => {
    expect(getCalendarWeekCount(690)).toBe(53)
    expect(getCalendarWeekCount(900)).toBe(53)
  })

  it('reduces the visible weeks to preserve usable cells on narrow screens', () => {
    expect(getCalendarWeekCount(330)).toBe(19)
    expect(getCalendarWeekCount(260)).toBe(14)
    expect(getCalendarWeekCount(220)).toBe(12)
  })

  it('keeps calendar cells readable within the available dimensions', () => {
    expect(getCalendarCellSize(330, 164, 19)).toBe(14)
    expect(getCalendarCellSize(220, 150, 12)).toBe(13)
  })

  it('uses recent complete weeks on compact layouts', () => {
    expect(getVisibleCalendarStartDate('2025-08-11', '2026-08-10', 19)).toBe('2026-04-06')
    expect(getVisibleCalendarStartDate('2025-08-11', '2026-08-10', 53)).toBe('2025-08-11')
  })

  it('pages through older compact ranges without leaving the available year', () => {
    expect(getCalendarPageEndDate('2025-08-11', '2026-08-10', 19, 1)).toBe('2026-03-30')
    expect(getCalendarPageEndDate('2025-08-11', '2026-08-10', 19, 2)).toBe('2025-12-21')
    expect(getCalendarPageEndDate('2025-08-11', '2026-08-10', 53, 1)).toBe('2026-08-10')
  })
})
