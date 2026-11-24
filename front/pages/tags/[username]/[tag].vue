<template>
  <div>
    <Header :user="user" v-if="user"/>
    <div class="flex flex-col divide-y divide-[#C0BEBF]/20 ">
      <Memo v-bind:memo="m" v-for="m in memos" :key="m.id" />
    </div>
    <MemoListFooter
      :loading="initialLoading"
      :loading-more="loadingMore"
      :has-next="hasNext"
      @load-more="loadMore"
    />
  </div>
</template>

<script setup lang="ts">
import type {MemoVO, UserVO} from "~/types";
import Memo from "~/components/Memo.vue";
import {memoChangedEvent, memoReloadEvent} from "~/event";

const route = useRoute()
const username = route.params.username
const tag = route.params.tag
const user = ref<UserVO>()

const initialLoading = ref(true)
const loadingMore = ref(false)
const hasNext = ref(false)
const state = reactive({
  page: 1,
  size: 10,
  tag,
})

const memos = ref<Array<MemoVO>>([])

onMounted(async ()=>{
  try {
    user.value = await useMyFetch<UserVO>('/user/profile/' + username)
    await reload()
  } finally {
    initialLoading.value = false
  }
})

const reload = async () => {
  initialLoading.value = true
  state.page = 1
  try {
    const res = await useMyFetch<{
      list: Array<MemoVO>,
      total: number,
      hasNext: boolean
    }>('/memo/list', state)
    memos.value = res?.list || []
    hasNext.value = Boolean(res?.hasNext)
  } finally {
    initialLoading.value = false
  }
}

const loadMore = async () => {
  if (initialLoading.value || loadingMore.value || !hasNext.value) {
    return
  }
  loadingMore.value = true
  const nextPage = state.page + 1
  try {
    const res = await useMyFetch<{
      list: Array<MemoVO>,
      total: number,
      hasNext: boolean
    }>('/memo/list', {
      ...state,
      page: nextPage,
    })
    state.page = nextPage
    memos.value = [...memos.value, ...(res?.list || [])]
    hasNext.value = Boolean(res?.hasNext)
  } finally {
    loadingMore.value = false
  }
}

memoReloadEvent.on(async () => {
  await reload()
})

memoChangedEvent.on(async (id: number) => {
  const res = await useMyFetch<MemoVO>('/memo/get?latest=1&id=' + id)
  const index = memos.value.findIndex(r => r.id === id)
  if (index >= 0) {
    memos.value[index] = res
  }
})
</script>

<style scoped>

</style>
