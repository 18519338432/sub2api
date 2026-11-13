<template>
  <section class="card mb-6 p-5">
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3"><div><h2 class="font-semibold text-gray-900 dark:text-white">用户限时余额订阅 <span class="text-sm font-normal text-gray-500">{{ total }} 笔</span></h2><p class="mt-1 text-xs text-gray-500">可使用下方用户筛选查看指定用户</p></div><div class="flex gap-2"><select v-model="status" aria-label="限时余额状态" class="input min-h-11" @change="page = 1; load()"><option value="">全部状态</option><option value="active">生效中</option><option value="exhausted">已用完</option><option value="expired">已过期</option></select><button type="button" class="btn btn-secondary min-h-11" :disabled="loading" @click="load">刷新</button></div></div>
    <p v-if="error" role="alert" class="py-4 text-sm text-red-600">{{ error }}</p><p v-else-if="loading" class="py-4 text-sm text-gray-500">正在加载限时余额…</p><p v-else-if="!items.length" class="py-4 text-sm text-gray-500">暂无符合条件的限时余额订阅</p>
    <div v-else class="grid gap-4 md:grid-cols-2 xl:grid-cols-3"><TimedBalanceCard v-for="item in items" :key="item.id" :balance="item"><p class="mb-2 break-all text-sm font-medium text-gray-900 dark:text-white">{{ item.email }} <span class="text-xs text-gray-500">#{{ item.user_id }}</span></p></TimedBalanceCard></div>
    <div v-if="total > 6" class="mt-4 flex items-center justify-end gap-3 text-sm"><button type="button" class="btn btn-secondary min-h-11" :disabled="page <= 1 || loading" @click="page--; load()">上一页</button><span>{{ page }} / {{ Math.ceil(total / 6) }}</span><button type="button" class="btn btn-secondary min-h-11" :disabled="page * 6 >= total || loading" @click="page++; load()">下一页</button></div>
  </section>
</template>
<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue'
import { apiClient } from '@/api/client'
import type { TimedBalance } from '@/api/subscriptions'
import TimedBalanceCard from '@/components/common/TimedBalanceCard.vue'
const props = defineProps<{ userId?: number | null }>()
const items = ref<(TimedBalance & { user_id: number; email: string })[]>([])
const total = ref(0), page = ref(1), status = ref(''), loading = ref(false), error = ref('')
let request = 0
async function load() {
 const current = ++request; loading.value = true; error.value = ''
 try { const { data } = await apiClient.get('/admin/subscriptions/timed-balances', { params: { user_id: props.userId || undefined, page: page.value, page_size: 6, status: status.value || undefined } }); if (current === request) { items.value = data.items; total.value = data.total } } catch { if (current === request) error.value = '加载失败，请点击刷新重试' } finally { if (current === request) loading.value = false }
}
watch(() => props.userId, () => { page.value = 1; void load() }, { immediate: true })
onUnmounted(() => { request++ })
</script>
