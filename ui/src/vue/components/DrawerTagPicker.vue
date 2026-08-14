<!-- Drawer-row wrapper around the reusable TagPicker. Keeps the row's
     edit-tags icon button; the picker popover and tag logic live in
     TagPicker. Teleported to body so the menu escapes the row's
     overflow-clipped action container. -->
<template>
  <TagPicker
    :conversation="conversation"
    class-name="drawer-tag-picker"
    teleport
    @click.stop
    @update="$emit('update', $event)"
  >
    <template #trigger="{ toggle }">
      <Button
        class="btn-icon-sm"
        text
        severity="secondary"
        size="small"
        v-tooltip.top="t('editTags')"
        :aria-label="t('editTags')"
        @click="toggle"
      >
        <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" class="drawer-icon-size">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            :stroke-width="2"
            d="M7 7h.01M7 3h5a1.99 1.99 0 011.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.99 1.99 0 013 12V7a4 4 0 014-4z"
          />
        </svg>
      </Button>
    </template>
  </TagPicker>
</template>

<script setup lang="ts">
import Button from "primevue/button";
import { useI18n } from "../composables/i18n";
import TagPicker from "./TagPicker.vue";
import type { Conversation, ConversationWithState } from "../../types";

const { t } = useI18n();

defineProps<{
  conversation: Conversation | ConversationWithState;
}>();

defineEmits<{
  (e: "update", conversation: Conversation): void;
}>();
</script>
