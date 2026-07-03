import { apiClient } from './client'

export interface UserCheckin {
  id: number
  user_id: number
  checkin_date: string
  reward_amount: number
  streak_days: number
  balance_after?: number
  created_at: string
  updated_at?: string
}

export interface CheckinStatus {
  checked_in: boolean
  today: string
  reward_amount: number
  current_streak: number
  last_checkin_date?: string
  next_checkin_at: string
  today_checkin?: UserCheckin
  recent_checkins: UserCheckin[]
}

async function getStatus(): Promise<CheckinStatus> {
  const { data } = await apiClient.get<CheckinStatus>('/user/check-in/status')
  return data
}

async function checkIn(): Promise<CheckinStatus> {
  const { data } = await apiClient.post<CheckinStatus>('/user/check-in')
  return data
}

async function getHistory(limit = 30): Promise<UserCheckin[]> {
  const { data } = await apiClient.get<UserCheckin[]>('/user/check-in/history', {
    params: { limit }
  })
  return data
}

export const checkinAPI = {
  getStatus,
  checkIn,
  getHistory
}

export default checkinAPI
