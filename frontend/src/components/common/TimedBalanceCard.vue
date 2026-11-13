<template>
  <article class="rounded-xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
    <div class="mb-4 flex flex-wrap items-start justify-between gap-3">
      <div><slot /><p class="text-xs text-gray-500 dark:text-gray-400">剩余限时余额</p><p class="mt-1 text-2xl font-semibold tabular-nums text-gray-900 dark:text-white">${{ usable.toFixed(2) }}</p></div>
      <span class="rounded-full px-3 py-1 text-xs font-medium" :class="active ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'">{{ active ? daysLabel : balance.remaining <= 0 ? '已用完' : '已过期' }}</span>
    </div>
    <div class="mb-2 flex justify-between gap-2 text-xs text-gray-500 dark:text-gray-400"><span>余量 {{ percent.toFixed(1) }}%</span><span class="tabular-nums">总额 ${{ balance.amount.toFixed(2) }}</span></div>
    <div role="progressbar" aria-label="限时余额剩余比例" :aria-valuenow="percent" :aria-valuemin="0" :aria-valuemax="100" class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700"><div class="h-full rounded-full" :class="!active ? 'bg-gray-400' : percent <= 20 ? 'bg-amber-500' : 'bg-emerald-500'" :style="{ width: `${percent}%` }" /></div>
    <div class="mt-3 flex flex-wrap justify-between gap-2 text-xs text-gray-500 dark:text-gray-400"><span>{{ active ? '生效中 · 优先扣费' : '历史余额' }}</span><time :datetime="balance.expires_at">{{ new Date(balance.expires_at).toLocaleString() }} 到期</time></div>
  </article>
</template>
<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import type { TimedBalance } from '@/api/subscriptions'
const props = defineProps<{ balance: TimedBalance }>()
const now = ref(Date.now())
let timer: ReturnType<typeof setInterval> | undefined
onMounted(() => { timer = setInterval(() => { now.value = Date.now() }, 30000) })
onUnmounted(() => { if (timer) clearInterval(timer) })
const remainingMs = computed(() => Date.parse(props.balance.expires_at) - now.value)
const active = computed(() => remainingMs.value > 0 && props.balance.remaining > 0 && props.balance.status === 'active')
const usable = computed(() => active.value ? props.balance.remaining : 0)
const percent = computed(() => props.balance.amount > 0 ? Math.max(0, Math.min(100, usable.value / props.balance.amount * 100)) : 0)
const daysLabel = computed(() => remainingMs.value < 86400000 ? '不足 1 天' : `剩余 ${Math.ceil(remainingMs.value / 86400000)} 天`)
</script>
