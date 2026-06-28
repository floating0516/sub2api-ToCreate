import { apiClient } from '../client'

export interface CustomBuildNotes {
  content: string
  path: string
  updated_at: string
}

export async function getCustomBuildNotes(): Promise<CustomBuildNotes> {
  const { data } = await apiClient.get<CustomBuildNotes>('/admin/custom-build/notes')
  return data
}

export const customBuildAPI = {
  getNotes: getCustomBuildNotes,
}

export default customBuildAPI
