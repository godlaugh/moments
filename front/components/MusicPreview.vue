<template>
  <meting-js
    v-if="assetsReady && id && server && type && api"
    :server="server"
    :type="type"
    :id="id"
    :api="api"
  />
</template>

<script setup lang="ts">
import type { MetingJSDTO } from "@/types";
import { ensureMusicAssets } from "~/utils";

const props = defineProps<MetingJSDTO>();
const assetsReady = ref(false);

onMounted(async () => {
  if (props.id && props.server && props.type && props.api) {
    await ensureMusicAssets();
    assetsReady.value = true;
  }
});
</script>

<style scoped>
</style>
