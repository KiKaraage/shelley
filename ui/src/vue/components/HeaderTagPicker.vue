<!-- Header button wrapper around the reusable TagPicker. Keeps the header's
     pi-tag icon button; the picker popover and tag logic live in TagPicker. -->
<template>
  <TagPicker :conversation="conversation" teleport @update="onUpdated">
    <template #trigger="{ toggle }">
      <Button
        class="btn-icon"
        text
        severity="secondary"
        :aria-label="t('editTags')"
        v-tooltip.top="t('editTags')"
        @click="toggle"
      >
        <i class="pi pi-tag chat-icon-1rem" aria-hidden="true" />
      </Button>
    </template>
  </TagPicker>
</template>

<script setup lang="ts">
import Button from "primevue/button";
import { useI18n } from "../composables/i18n";
import TagPicker from "./TagPicker.vue";
import type { Conversation } from "../../types";

const { t } = useI18n();

defineProps<{
  conversation: Conversation | undefined;
  onUpdated?: (conversation: Conversation) => void;
}>();
</script>
