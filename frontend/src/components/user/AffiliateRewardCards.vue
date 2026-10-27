<template>
  <section class="card p-6" aria-labelledby="reward-cards-title">
    <h3 id="reward-cards-title" class="text-balance text-base font-semibold text-gray-900 dark:text-white">邀请奖励</h3>
    <p class="mt-2 text-pretty text-sm text-gray-600 dark:text-dark-300">新用户通过邀请注册，双方各得 $5 余额。被邀请者每兑换一张 30 天订阅卡，邀请者获得同类型 3 天订阅卡，仅限本人兑换，奖励卡不再产生返利。</p>
    <p v-if="error" role="alert" class="mt-3 text-sm text-red-600">{{ error }} <button class="underline" @click="load">重试</button></p>
    <p v-else-if="loading" class="mt-3 text-sm text-gray-500">正在加载奖励卡…</p>
    <p v-else-if="!cards.length" class="mt-3 text-sm text-gray-500">暂无订阅奖励卡，分享下方邀请链接开始邀请。</p>
    <ul v-else class="mt-4 divide-y divide-gray-200 dark:divide-dark-700">
      <li v-for="card in cards" :key="card.code" class="flex flex-wrap items-center justify-between gap-3 py-3">
        <div class="min-w-0 text-sm text-gray-700 dark:text-gray-300"><p>{{ card.group_name }} · {{ card.validity_days }} 天</p><code class="break-all text-xs">{{ card.code }}</code></div>
        <button class="btn btn-secondary btn-sm" :disabled="card.status !== 'unused' || redeeming !== ''" @click="redeem(card)">{{ redeeming === card.code ? '兑换中…' : card.status === 'unused' ? '立即兑换' : '已使用' }}</button>
      </li>
    </ul>
    <p v-if="message" role="status" class="mt-3 text-sm text-primary-600">{{ message }}</p>
  </section>
</template>
<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { apiClient } from '@/api/client'
import { getRewardCards, type RewardCard } from '@/api/activityRewards'
import { extractApiErrorMessage } from '@/utils/apiError'
const cards = ref<RewardCard[]>([])
const loading = ref(true)
const redeeming = ref('')
const error = ref('')
const message = ref('')
async function load() {
  loading.value = true; error.value = ''
  try { cards.value = await getRewardCards() }
  catch (e) { error.value = extractApiErrorMessage(e, '奖励卡加载失败') }
  finally { loading.value = false }
}
async function redeem(card: RewardCard) {
  redeeming.value = card.code; error.value = ''; message.value = ''
  try { await apiClient.post('/redeem', { code: card.code }); message.value = '订阅奖励已到账'; await load() }
  catch (e) { error.value = extractApiErrorMessage(e, '兑换失败，请重试') }
  finally { redeeming.value = '' }
}
onMounted(load)
</script>
