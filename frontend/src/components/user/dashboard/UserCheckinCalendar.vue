<template>
  <section class="card p-4 sm:p-6" aria-labelledby="checkin-heading">
    <div class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <h2 id="checkin-heading" class="text-balance text-lg font-semibold text-gray-900 dark:text-white">每日签到 <span v-if="calendar" class="text-sm font-normal text-gray-500">{{ calendar.month }}</span></h2>
        <p class="mt-1 text-pretty text-sm text-gray-500 dark:text-dark-400">满勤基础奖励 $25，每连续 7 天额外 $2，当月最高 $33。按北京时间每日重置，断签及月初重新累计。</p>
      </div>
      <button class="btn btn-primary" :disabled="busy || !calendar?.eligible || checkedToday" @click="checkin">{{ busy ? '加载中…' : calendar && !calendar.eligible ? '仅限订阅用户' : checkedToday ? '今日已签到' : '领取今日奖励' }}</button>
    </div>
    <p v-if="error" role="alert" class="mt-3 text-sm text-red-600">{{ error }} <button class="underline" @click="load">重试</button></p>
    <template v-if="calendar">
      <p v-if="!calendar.eligible" class="mt-3 text-sm text-gray-600 dark:text-dark-300">签到仅限持有有效订阅的用户，订阅到期、暂停或取消后无法领取。<RouterLink to="/subscriptions" class="text-primary-600 underline">查看订阅</RouterLink></p>
      <p class="my-4 text-sm tabular-nums text-gray-700 dark:text-gray-300" aria-live="polite">本月已领取 ${{ (calendar.total_cents / 100).toFixed(2) }} · 连续签到 {{ calendar.streak }} 天</p>
      <div class="grid grid-cols-7 gap-1 sm:gap-2" aria-label="当月签到日历">
        <div v-for="week in ['一', '二', '三', '四', '五', '六', '日']" :key="week" class="py-1 text-center text-xs text-gray-500">{{ week }}</div>
        <div v-for="blank in leadingBlanks" :key="`blank-${blank}`" aria-hidden="true" />
        <div v-for="day in calendar.days" :key="day.date" class="rounded-lg border p-1 text-center text-xs tabular-nums sm:p-3" :class="day.checked ? 'border-primary-200 bg-primary-50 text-primary-700 dark:border-primary-800 dark:bg-primary-900/20 dark:text-primary-300' : 'border-gray-200 text-gray-600 dark:border-dark-700 dark:text-dark-300'" :aria-current="day.date === calendar.today ? 'date' : undefined">
          <div class="font-medium">{{ Number(day.date.slice(-2)) }}<span v-if="day.date === calendar.today">·今</span></div>
          <div class="mt-1">${{ (day.base_cents / 100).toFixed(2) }}</div>
          <div v-if="day.checked" class="mt-1">已签<span v-if="day.bonus_cents"> +$2</span></div>
          <div v-else-if="day.date < calendar.today" class="mt-1 text-gray-400">未签</div>
        </div>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { getCheckinCalendar, claimCheckin, type CheckinCalendar } from '@/api/activityRewards'
import { useAuthStore } from '@/stores/auth'
import { extractApiErrorMessage } from '@/utils/apiError'
const auth = useAuthStore()
const calendar = ref<CheckinCalendar | null>(null)
const busy = ref(false)
const error = ref('')
const checkedToday = computed(() => calendar.value?.days.some(day => day.date === calendar.value?.today && day.checked))
const leadingBlanks = computed(() => calendar.value ? (new Date(`${calendar.value.month}-01T00:00:00Z`).getUTCDay() + 6) % 7 : 0)
async function load() {
  busy.value = true; error.value = ''
  try { calendar.value = await getCheckinCalendar() }
  catch (e) { error.value = extractApiErrorMessage(e, '签到日历加载失败') }
  finally { busy.value = false }
}
async function checkin() {
  busy.value = true; error.value = ''
  try { calendar.value = await claimCheckin(); await auth.refreshUser() }
  catch (e) { error.value = extractApiErrorMessage(e, '签到失败，请重试') }
  finally { busy.value = false }
}
onMounted(load)
</script>
