<template>
  <div>
    <Header :user="currentUser"/>

    <div class="p-4 space-y-4">
      <UFormGroup label="日期范围" name="contentContains" :ui="{label:{base:'font-bold'}}">
        <UPopover :popper="{ placement: 'bottom-start' }">
          <UButton icon="i-heroicons-calendar-days-20-solid" color="white" variant="solid" class="w-full">
            从 {{ format(state.range.start, 'yyy-MM-dd') }} 到 {{ format(state.range.end, 'yyy-MM-dd') }}
          </UButton>

          <template #panel="{ close }">
            <div class="flex flex-col items-center sm:divide-x divide-gray-200 dark:divide-gray-800">
              <div class="hidden sm:flex flex-row py-4">
                <UButton
                    v-for="(range, index) in ranges"
                    :key="index"
                    :label="range.label"
                    color="gray"
                    variant="ghost"
                    class="rounded-none px-6"
                    :class="[isRangeSelected(range.duration) ? 'bg-gray-100 dark:bg-gray-800' : 'hover:bg-gray-50 dark:hover:bg-gray-800/50']"
                    truncate
                    @click="selectRange(range.duration)"
                />
              </div>
              <DatePicker v-model="state.range" @close="close"/>
            </div>
          </template>
        </UPopover>
      </UFormGroup>

      <UFormGroup label="包含内容" name="contentContains" :ui="{label:{base:'font-bold'}}">
        <UInput v-model="state.contentContains"/>
      </UFormGroup>
      <UFormGroup label="包含标签" name="tagContains" :ui="{label:{base:'font-bold'}}">
        <USelectMenu multiple v-model="state.tags" searchable :options="tags">
          <template #label>
            <span v-if="state.tags.length" class="truncate">{{ state.tags.join(', ') }}</span>
            <span v-else>选择标签</span>
          </template>
        </USelectMenu>
      </UFormGroup>
      <UFormGroup label="可见性" name="showType" :ui="{label:{base:'font-bold'}}">
        <USelectMenu v-model="state.showType"
                     :options="[{value:-1,label:'所有的'},{value:1,label:'公开的'},{value:0,label:'自己可见'}]"
                     option-attribute="label" value-attribute="value"/>
      </UFormGroup>
      <UButton class="my-2" @click="reload">搜索</UButton>
    </div>

    <div class="flex flex-col divide-y divide-[#C0BEBF]/20 ">
      <Memo v-bind:memo="m" v-for="m in memos" :key="m.id"/>
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
import {add, format, isSameDay, sub} from "date-fns";
import Memo from "~/components/Memo.vue";
import {memoChangedEvent, memoReloadEvent} from "~/event";

const ranges = [
  {label: '一周内', duration: {days: 7}},
  {label: '一月内', duration: {days: 31}},
  {label: '三月内', duration: {days: 90}},
  {label: '本年', duration: {months: 12}},
  {label: '近三年', duration: {years: 3}},
]
const tags = ref<string[]>([])
const currentUser = useState<UserVO>('userinfo')
const state = reactive({
  page: 1,
  size: 10,
  contentContains: "",
  tags: [],
  showType: -1,
  range: {
    start: sub(new Date(), {days: 31}),
    end: add(new Date(), {days: 1})
  }
})

function isRangeSelected(duration: Duration) {
  return isSameDay(state.range.start, sub(new Date(), duration)) && isSameDay(state.range.end, new Date())
}

function selectRange(duration: Duration) {
  state.range = {start: sub(new Date(), duration), end: new Date()}
}

const loadTags = async () => {
  const res = await useMyFetch<{
    tags: string[]
  }>("/tag/list")
  tags.value = res.tags
}


const initialLoading = ref(true)
const loadingMore = ref(false)
const hasNext = ref(false)


const memos = ref<Array<MemoVO>>([])
onMounted(async () => {
  await loadTags()
  await reload()
})

const reload = async () => {
  initialLoading.value = true
  state.page = 1
  try {
    const res = await useMyFetch<{
      list: Array<MemoVO>,
      total: number,
      hasNext: boolean
    }>('/memo/list', {
      page: state.page,
      size: state.size,
      start: state.range.start,
      end: state.range.end,
      contentContains: state.contentContains,
      showType: state.showType,
      tag: state.tags.join(','),
    })
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
      page: nextPage,
      size: state.size,
      start: state.range.start,
      end: state.range.end,
      contentContains: state.contentContains,
      showType: state.showType,
      tag: state.tags.join(','),
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