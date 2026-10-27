import { apiClient } from './client'

export interface CheckinDay { date: string; base_cents: number; bonus_cents: number; checked: boolean }
export interface CheckinCalendar { eligible: boolean; month: string; today: string; streak: number; total_cents: number; days: CheckinDay[] }
export interface RewardCard { code: string; group_name: string; status: string; validity_days: number }
export async function getCheckinCalendar(): Promise<CheckinCalendar> {
  return (await apiClient.get<CheckinCalendar>('/user/checkin')).data
}
export async function claimCheckin(): Promise<CheckinCalendar> {
  return (await apiClient.post<CheckinCalendar>('/user/checkin')).data
}
export async function getRewardCards(): Promise<RewardCard[]> {
  return (await apiClient.get<RewardCard[]>('/user/aff/reward-cards')).data
}
