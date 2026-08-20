<template>
  <div class="px-4 space-y-2">
    <div class="flex justify-between items-center pt-4 text-gray-600">
      <NuxtLink class="flex items-center" title="返回主页">
        <UIcon @click="navigateTo('/')" name="i-carbon-chevron-left" class="w-5 h-5 cursor-pointer mr-4"/>
        <span v-if="$route.path==='/new'">新增内容</span>
        <span v-else>修改内容</span>
      </NuxtLink>
      <UButton @click="saveMemo">发表</UButton>
    </div>
    <div class="flex gap-2 text-lg text-gray-600 pt-4 ">
      <ExternalUrl v-model:favicon="state.externalFavicon" v-model:title="state.externalTitle"
                   v-model:url="state.externalUrl"/>

      <upload-image v-model:imgs="state.imgs"/>
      <music v-bind="state.music" @confirm="updateMusic"/>
      <upload-video @confirm="handleVideo" v-bind="state.video"/>
      <douban-edit v-model:type="doubanType" v-model:data="doubanData"/>
      <UPopover :popper="{ arrow: true }" mode="click">
        <UIcon name="i-carbon-calendar" class="w-6 h-6" title="自定义时间"/>
        <template #panel="{close}">
          <DatePicker
            v-model="state.createdAt"
            mode="datetime"
            is24hr
            :time-accuracy="2"
            :rules="{ seconds: 0 }"
            @close="close"
          />
        </template>
      </UPopover>
      <UIcon name="i-carbon-text-clear-format" @click="reset" class="w-6 h-6 cursor-pointer" title="清空"></UIcon>
    </div>

    <div class="w-full" @contextmenu.prevent="onContextMenu">
      <div class="relative">
        <UTextarea ref="contentRef" v-model="state.content" :rows="8" autoresize padded autofocus/>
        <UIcon class="text-[#9fc84a] w-6 h-6 animate-bounce absolute right-2 bottom-1 cursor-pointer select-none" name="i-carbon-face-satisfied" @click="toggleEmoji"/>
      </div>

      <Emoji v-if="emojiShow" @selected="emojiSelected" @close="emojiShow=false"/>

      <div class="flex flex-wrap items-center gap-1 my-2">
        <UBadge v-for="(tag, index) in selectedTags" :key="`selected-${index}`" color="gray" variant="solid"
                class="cursor-pointer flex items-center gap-1" :title="tag">
          <span>{{ tag }}</span>
          <UIcon name="i-carbon-close" class="w-3 h-3" @click.stop="removeTag(tag)"/>
        </UBadge>
        <UPopover :popper="{ arrow: true }" mode="click">
          <UButton color="white" variant="solid" size="xs" icon="i-carbon-tag" :trailing="false">
            {{ selectedTags.length ? '标签' : '选择标签' }}
          </UButton>
          <template #panel="{close}">
            <div class="p-2 w-80">
              <UInput v-model="tagQuery" placeholder="输入过滤;或输入 父/子 新建多级标签" autofocus
                      class="w-full" size="xs"/>
              <div class="max-h-60 overflow-auto mt-1">
                <div v-if="filteredTree.length === 0 && !canCreate" class="text-xs text-gray-400 px-2 py-3 text-center">
                  暂无标签,输入名称可新建
                </div>
                <template v-for="node in filteredTree" :key="node.id">
                  <div class="flex items-center justify-between px-2 py-1.5 rounded cursor-pointer text-sm hover:bg-gray-100 dark:hover:bg-gray-700"
                       :style="{ paddingLeft: `${8 + node._depth * 16}px` }"
                       :class="{ 'bg-gray-100 dark:bg-gray-700': selectedTags.includes(node.path) }"
                       @click="toggleTag(node.path)">
                    <span class="flex items-center gap-1 truncate">
                      <UIcon v-if="selectedTags.includes(node.path)" name="i-carbon-checkmark" class="w-3.5 h-3.5 text-green-500"/>
                      <UIcon v-else-if="node.children.length" name="i-carbon-folder" class="w-3.5 h-3.5 text-gray-400"/>
                      <UIcon v-else name="i-carbon-tag" class="w-3.5 h-3.5 text-gray-400"/>
                      <span class="truncate">{{ node.name }}</span>
                    </span>
                    <span class="text-xs text-gray-400 shrink-0 ml-2">{{ node.total }}</span>
                  </div>
                </template>
                <div v-if="canCreate"
                     class="flex items-center gap-1 px-2 py-1.5 rounded cursor-pointer text-sm text-green-600 hover:bg-gray-100 dark:hover:bg-gray-700"
                     @click="createTag">
                  <UIcon name="i-carbon-add" class="w-3.5 h-3.5"/>
                  <span>新建"{{ tagQuery.trim() }}"</span>
                </div>
              </div>
            </div>
          </template>
        </UPopover>
      </div>

      <UContextMenu v-model="isOpen" :virtual-element="virtualElement">
        <div class="px-2 py-1 flex flex-col gap-2 text-xs">
          <div class="mb-2 text-gray-300">点击标签插入</div>
          <div v-for="(tag,index) in existTags" :key="index" class="cursor-pointer">
            <UBadge size="xs" color="gray" variant="solid" @click="clickTag(tag)">{{ tag }}</UBadge>
          </div>
        </div>
      </UContextMenu>
    </div>

    <div class="flex justify-between items-center">
      <div class="flex flex-row gap-1 items-center text-[#576b95] text-sm cursor-pointer">
        <UPopover :popper="{ arrow: true }" mode="click">
          <div class="flex items-center gap-1">
            <UIcon name="i-carbon-location"/>
            <span>{{ state.location ? locationLabel : '自定义位置' }}</span>
          </div>
          <template #panel="{close}">
            <div class="p-4">
              <UButtonGroup>
                <UInput v-model="state.location" placeholder="自定义位置,空格分隔"/>
                <UButton @click="close" color="white" variant="solid">关闭</UButton>
              </UButtonGroup>
            </div>
          </template>
        </UPopover>
      </div>

      <div class="flex gap-1 text-gray-500 items-center">
          <span>{{ state.showType ? '公开' : '私密' }}</span>
          <UToggle v-model="state.showType"/>
        </div>
      </div>

    <div class="flex flex-col gap-2">
      <external-url-preview :favicon="state.externalFavicon" :title="state.externalTitle" :url="state.externalUrl"/>
      <upload-image-preview :imgs="state.imgs" @remove-image="handleRemoveImage" @drag-image="handleDragImage"/>
      <music-preview v-if="state.music && state.music.id && state.music.type && state.music.server"
                     v-bind="state.music"/>
      <douban-book-preview :book="doubanData" v-if="doubanType === 'book' && doubanData&& doubanData.title"/>
      <douban-movie-preview :movie="doubanData" v-if="doubanType === 'movie' && doubanData&& doubanData.title"/>
      <video-preview-iframe v-if="['bilibili', 'youtube'].includes(state.video.type) && state.video.value" :url="state.video.value"/>
      <video-preview v-if="state.video.type === 'online' && state.video.value" :url="state.video.value"/>
    </div>
  </div>


</template>

<script setup lang="ts">
import {useMouse, useWindowScroll} from '@vueuse/core'
import type {
  DoubanBook,
  DoubanMovie,
  ExtDTO,
  MemoVO,
  MetingMusicServer,
  MetingMusicType,
  MusicDTO,
  TagListResp,
  TagNode,
  Video,
  VideoType
} from "~/types";
import {toast} from "vue-sonner";
import UploadImage from "~/components/UploadImage.vue";
import Emoji from "~/components/Emoji.vue";
import dayjs from "dayjs";

const doubanType = ref<'book' | 'movie'>('book')
const doubanData = ref<DoubanBook | DoubanMovie>({})
const contentRef = ref(null)
const props = defineProps<{ id?: number }>()
const defaultState = {
  id: props.id || 0,
  createdAt: '' as string,
  content: "",
  ext: "",
  pinned: false,
  showType: true,
  location: "",
  externalFavicon: "",
  externalTitle: "",
  externalUrl: "",
  imgs: "",
  music: {
    id: '',
    api: 'https://api.i-meto.com/meting/api?server=:server&type=:type&id=:id&r=:r',
    server: 'netease' as MetingMusicServer,
    type: 'song' as MetingMusicType
  },
  video: {
    type: 'youtube' as VideoType,
    value: ""
  },
  doubanBook: {} as DoubanBook,
  doubanMovie: {} as DoubanMovie,
  tags: Array<string>(),
}
const selectedTags = ref<Array<string>>([])
const tagQuery = ref("")
const tagTree = ref<TagNode[]>([])

const flattenTree = (nodes: TagNode[], depth: number): Array<TagNode & { _depth: number }> => {
  const result: Array<TagNode & { _depth: number }> = []
  for (const n of nodes) {
    result.push({...n, _depth: depth})
    result.push(...flattenTree(n.children, depth + 1))
  }
  return result
}

const filteredTree = computed(() => {
  const q = tagQuery.value.trim().toLowerCase()
  const flat = flattenTree(tagTree.value, 0)
  if (!q) {
    return flat
  }
  // 命中:名字或全路径包含关键字;同时保留祖先以便理解层级
  const hitPaths = new Set<string>()
  for (const n of flat) {
    if (n.name.toLowerCase().includes(q) || n.path.toLowerCase().includes(q)) {
      hitPaths.add(n.path)
    }
  }
  return flat.filter(n => {
    if (hitPaths.has(n.path)) {
      return true
    }
    // 是某命中节点的祖先
    return flat.some(h => hitPaths.has(h.path) && h.path.startsWith(n.path + "/"))
  })
})

const canCreate = computed(() => {
  const q = tagQuery.value.trim()
  if (!q) {
    return false
  }
  const paths = new Set(flattenTree(tagTree.value, 0).map(n => n.path))
  return !paths.has(q) && q.split("/").length <= 10 && !q.split("/").some((s: string) => !s.trim() || s.length > 50)
})

const toggleTag = (path: string) => {
  const idx = selectedTags.value.indexOf(path)
  if (idx >= 0) {
    selectedTags.value.splice(idx, 1)
  } else {
    selectedTags.value.push(path)
  }
}

const removeTag = (path: string) => {
  const idx = selectedTags.value.indexOf(path)
  if (idx >= 0) {
    selectedTags.value.splice(idx, 1)
  }
}

const createTag = () => {
  const path = tagQuery.value.trim()
  if (!path || selectedTags.value.includes(path)) {
    return
  }
  selectedTags.value.push(path)
  tagQuery.value = ""
}
const state = reactive({
  ...defaultState
})
const existTags = ref<string[]>([])
const reset = () => {
  Object.assign(state, defaultState)
}

const locationLabel = computed(() => {
  return state.location.split(" ").join(" · ")
})

const handleDragImage = (imgs: string[]) => {
  state.imgs = imgs.filter(Boolean).join(",")
}

const updateMusic = (music: MusicDTO) => {
  state.music.id = ""
  setTimeout(() => {
    Object.assign(state.music, music)
  }, 500)
}

const handleVideo = (video: Video) => {
  state.video = video
}

const {x, y} = useMouse()
const {y: windowY} = useWindowScroll()
const isOpen = ref(false)
const virtualElement = ref({getBoundingClientRect: () => ({})})

const handleRemoveImage = (index: number) => {
  const arr = state.imgs.split(",").filter(Boolean)
  arr.splice(index, 1)
  state.imgs = arr.join(",")
}

function onContextMenu() {
  if (existTags.value.length <= 0) {
    return
  }
  const top = unref(y) - unref(windowY)
  const left = unref(x)

  virtualElement.value.getBoundingClientRect = () => ({
    width: 0,
    height: 0,
    top,
    left
  })

  isOpen.value = true
}

const loadTags = async () => {
  const res = await useMyFetch<TagListResp>("/tag/list")
  tagTree.value = res.tree || []
  existTags.value = res.tags || []
}

const emojiShow = ref(false)

const toggleEmoji = () => {
  emojiShow.value = !emojiShow.value
}
const emojiSelected = (emoji: string) => {
  state.content = state.content + emoji
}

const clickTag = (tag: string) => {
  isOpen.value = false;
  if (!selectedTags.value.includes(tag)){
    selectedTags.value = [...selectedTags.value , tag]
  }

  //@ts-ignore
  (contentRef.value?.textarea as HTMLTextAreaElement).focus()
}
onMounted(async () => {
  if (state.id > 0) {
    const res = await useMyFetch<MemoVO>('/memo/get?id=' + state.id)
    Object.assign(state, res)
    state.showType = res.showType === 1
    const ext = JSON.parse(res.ext) as ExtDTO
    Object.assign(state.music, ext.music)
    Object.assign(state.video, ext.video)
    doubanType.value = ext.doubanBook && ext.doubanBook.title ? 'book' : 'movie'
    doubanData.value = doubanType.value === 'book' ? ext.doubanBook : ext.doubanMovie
    selectedTags.value = res.tagPaths && res.tagPaths.length
      ? res.tagPaths
      : (res.tags ? res.tags.substring(0, res.tags.length - 1).split(',') : [])
    state.createdAt = dayjs(res.createdAt).format()
  }
  await loadTags()
})

// const keydown=(event:KeyboardEvent)=>{
//   if(event.key === '#'){
//     tagPopoverOpen.value = true
//   }
// }

const saveMemo = async () => {

  const doubanKey = doubanType.value === 'book' ? 'doubanBook' : 'doubanMovie'
  await useMyFetch('/memo/save', {
    id: state.id,
    content: state.content,
    ext: {
      music: state.music.id ? state.music : {},
      [doubanKey]: doubanData.value,
      video: state.video.value ? state.video : {},
    },
    pinned: state.pinned,
    showType: state.showType ? 1 : 0,
    externalFavicon: state.externalUrl ? state.externalFavicon : "",
    externalTitle: state.externalTitle,
    externalUrl: state.externalUrl,
    imgs: state.imgs.split(",").filter(Boolean),
    location: state.location,
    tags: [...selectedTags.value],
    createdAt: state.createdAt || dayjs().format(),
  })
  toast.success("保存成功!")
  await navigateTo('/')
}

</script>

<style scoped>

</style>