import { apiClient } from '../client'
import type {
  AdminLeaderboardSnapshot,
  GenerateLeaderboardRewardResult,
  LeaderboardPeriod,
  LeaderboardRewardSettings
} from '../leaderboard'

export async function getAdminLeaderboard(
  period: LeaderboardPeriod,
  limit = 50
): Promise<AdminLeaderboardSnapshot> {
  const { data } = await apiClient.get<AdminLeaderboardSnapshot>('/admin/leaderboard', {
    params: { period, limit }
  })
  return data
}

export async function getLeaderboardSettings(): Promise<LeaderboardRewardSettings> {
  const { data } = await apiClient.get<LeaderboardRewardSettings>('/admin/leaderboard/settings')
  return data
}

export async function updateLeaderboardSettings(
  settings: LeaderboardRewardSettings
): Promise<LeaderboardRewardSettings> {
  const { data } = await apiClient.put<LeaderboardRewardSettings>(
    '/admin/leaderboard/settings',
    settings
  )
  return data
}

export async function generateLeaderboardReward(
  period: LeaderboardPeriod
): Promise<GenerateLeaderboardRewardResult> {
  const { data } = await apiClient.post<GenerateLeaderboardRewardResult>(
    '/admin/leaderboard/rewards/generate',
    { period }
  )
  return data
}

export const adminLeaderboardAPI = {
  get: getAdminLeaderboard,
  getSettings: getLeaderboardSettings,
  updateSettings: updateLeaderboardSettings,
  generateReward: generateLeaderboardReward
}

export default adminLeaderboardAPI

