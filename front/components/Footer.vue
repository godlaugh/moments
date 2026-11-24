<template>
  <div
    v-if="sysConfig?.beiAnNo"
    class="footer p-4 flex justify-center items-center text-xs"
  >
    <div class="flex flex-col gap-1 items-center">
      <a
        class="my-2 text-gray-500"
        href="https://beian.miit.gov.cn/"
        target="_blank"
      >
        {{ sysConfig.beiAnNo }}
      </a>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { SysConfigVO } from "~/types";

const sysConfig = useState<SysConfigVO>("sysConfig");
</script>

<style scoped></style>
