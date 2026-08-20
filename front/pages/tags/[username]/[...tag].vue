<template>
  <div>
    <Header :user="user" v-if="user"/>
    <div class="px-4 py-3 flex flex-wrap items-center gap-1 text-sm text-gray-500">
      <NuxtLink to="/" class="hover:text-gray-800 dark:hover:text-gray-200">首页</NuxtLink>
      <template v-for="(seg, i) in pathSegs" :key="`crumb-${i}`">
        <UIcon name="i-carbon-chevron-right" class="w-3.5 h-3.5 text-gray-400"/>
        <NuxtLink v-if="i < pathSegs.length - 1" :to="tagUrl(seg.full)"
                  class="hover:text-gray-800 dark:hover:text-gray-200">{{ seg.name }}</NuxtLink>
        <span v-else class="text-gray-800 dark:text-gray-200 font-medium">{{ seg.name }}</span>
      </template>
      <span v-if="total > 0" class="text-xs text-gray-400">共 {{ total }} 条</span>
    </div>

    <div v-if="childTags.length" class="px-4 pb-2 flex flex-wrap items-center gap-2">
      <span class="text-xs text-gray-400">子标签:</span>
      <NuxtLink v-for="child in childTags" :key="`child-${child.id}`" :to="tagUrl(child.path)">
        <UBadge size="xs" :color="selectedTags.includes(child.path) ? 'primary' : 'gray'" variant="subtle"
                class="cursor-pointer">
          {{ child.name }}
          <span class="opacity-60 ml-0.5">{{ child.total }}</span>
        </UBadge>
      </NuxtLink>
    </div>

    <div class="flex flex-col divide-y divide-[#C0BEBF]/20 ">
      <Memo v-bind:memo="m" v-for="m in memos" :key="m.id"/>
    </div>
    <div ref="loadMoreEle" class="text-xs text-center text-gray-500 py-2" @click="loadMore" v-if="hasNext">
      点击加载更多
    </div>
    <div class="text-xs text-center text-gray-500 py-2" @click="loadMore" v-else>
      已经到底啦
    </div>
  </div>
</template>

<script setup lang="ts">
import type {MemoVO, TagListResp, TagNode, UserVO} from "~/types";
import Memo from "~/components/Memo.vue";
import {useElementVisibility} from "@vueuse/core";
import {memoChangedEvent, memoReloadEvent} from "~/event";

const route = useRoute()
const username = route.params.username as string

// catch-all 路由: tag 为段数组,如 ['AI','Skill'];逐段已由 vue-router 解码
const tagFromRoute = () => {
  const t = route.params.tag
  return (Array.isArray(t) ? t.join("/") : (t || "")).toString()
}
const tag = ref(tagFromRoute())
const user = ref<UserVO>()
const total = ref(0)

// 层级用纯路径段表达:/tags/user/AI/Skill;每段单独编码以兼容含特殊字符的段名
const tagUrl = (path: string) => `/tags/${username}/${path.split("/").map(encodeURIComponent).join("/")}`

watch(() => route.params.tag, (newTag) => {
  const joined = Array.isArray(newTag) ? newTag.join("/") : (newTag || "")
  if (joined && joined !== tag.value) {
    tag.value = joined
    reload()
  }
})

const pathSegs = computed(() => {
  const segs = tag.value.split("/").filter(Boolean)
  return segs.map((name, i) => ({
    name,
    full: segs.slice(0, i + 1).join("/")
  }))
})

const findNode = (nodes: TagNode[], path: string): TagNode | null => {
  for (const n of nodes) {
    if (n.path === path) {
      return n
    }
    const found = findNode(n.children, path)
    if (found) {
      return found
    }
  }
  return null
}

const selectedTags = ref<string[]>([])
const childTags = computed(() => {
  if (selectedTags.value.length === 0) {
    return []
  }
  const node = findNode(tagTree.value, tag.value)
  return node ? node.children : []
})
const tagTree = ref<TagNode[]>([])

const loadTagTree = async () => {
  try {
    const res = await useMyFetch<TagListResp>(`/tag/list?username=${encodeURIComponent(username)}`)
    tagTree.value = res.tree || []
    const node = findNode(tagTree.value, tag.value)
    selectedTags.value = node ? [node.path] : []
  } catch (e) {
    selectedTags.value = []
  }
}

onMounted(async () => {
  user.value = await useMyFetch<UserVO>('/user/profile/' + username)
  await loadTagTree()
  await reload()
})

const loadMoreEle = ref(null)
const targetIsVisible = useElementVisibility(loadMoreEle)
watch(targetIsVisible, async (visible) => {
  if (visible) {
    await loadMore()
  }
})
const hasNext = ref(false)
const state = reactive({
  page: 1,
  size: 10,
  tag,
  username,
})

const memos = ref<Array<MemoVO>>([])

const reload = async () => {
  state.page = 1
  const res = await useMyFetch<{
    list: Array<MemoVO>,
    total: number,
    hasNext: boolean
  }>('/memo/list', state)
  memos.value = res.list || []
  total.value = res.total || 0
  hasNext.value = res.hasNext
}

const loadMore = async () => {
  if (!hasNext.value) {
    return
  }
  state.page = state.page + 1
  const res = await useMyFetch<{
    list: Array<MemoVO>,
    total: number,
    hasNext: boolean
  }>('/memo/list', state)
  memos.value = [...memos.value, ...(res.list || [])]
  hasNext.value = res.hasNext
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
