<template>
  <div
    v-if="loading"
    class="py-10 flex items-center justify-center gap-2 text-xs text-gray-500"
    role="status"
    aria-live="polite"
  >
    <svg
      class="animate-spin h-4 w-4 text-gray-400 shrink-0"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      aria-hidden="true"
    >
      <circle
        class="opacity-25"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        stroke-width="3"
      />
      <path
        class="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8v3a5 5 0 00-5 5H4z"
      />
    </svg>
    <span>正在沏茶，请稍等…</span>
  </div>
  <div
    v-else-if="hasNext"
    ref="sentinel"
    class="py-4 flex items-center justify-center gap-2 text-xs text-gray-500"
    role="status"
    aria-live="polite"
  >
    <svg
      class="animate-spin h-3.5 w-3.5 text-gray-400 shrink-0"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      aria-hidden="true"
    >
      <circle
        class="opacity-25"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        stroke-width="3"
      />
      <path
        class="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8v3a5 5 0 00-5 5H4z"
      />
    </svg>
    <span>正在加载更多…</span>
  </div>
  <div v-else class="text-xs text-center text-gray-500 py-4">已经到底啦</div>
</template>

<script setup lang="ts">
import { useIntersectionObserver } from "@vueuse/core";

const props = defineProps<{
  loading: boolean;
  loadingMore: boolean;
  hasNext: boolean;
}>();

const emit = defineEmits<{
  (e: "load-more"): void;
}>();

const sentinel = ref<HTMLElement | null>(null);
const PRELOAD_MARGIN_PX = 240;

const triggerLoadMore = () => {
  if (!props.loading && !props.loadingMore && props.hasNext) {
    emit("load-more");
  }
};

useIntersectionObserver(
  sentinel,
  ([entry]) => {
    if (entry?.isIntersecting) {
      triggerLoadMore();
    }
  },
  { rootMargin: `0px 0px ${PRELOAD_MARGIN_PX}px 0px` }
);

watch(
  () => [props.loading, props.loadingMore, props.hasNext] as const,
  async ([loading, loadingMore, hasNext]) => {
    if (loading || loadingMore || !hasNext) return;
    await nextTick();
    if (!sentinel.value) return;
    const rect = sentinel.value.getBoundingClientRect();
    if (rect.top <= window.innerHeight + PRELOAD_MARGIN_PX) {
      triggerLoadMore();
    }
  }
);
</script>

<style scoped></style>
