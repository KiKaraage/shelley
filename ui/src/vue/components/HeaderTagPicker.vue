<!-- Tag picker for the chat header. A pi pi-tag icon that opens a dropdown
     with fixed suggested tags (filterable), a type-to-create input, and
     toggle-to-apply/remove. No delete buttons — removal lives in the drawer. -->
<template>
  <div ref="wrapperRef" class="tag-picker-wrapper">
    <Button
      class="btn-icon"
      text
      severity="secondary"
      :aria-label="t('editTags')"
      v-tooltip.top="t('editTags')"
      @click="toggleOpen"
    >
      <i class="pi pi-tag chat-icon-1rem" aria-hidden="true" />
    </Button>
    <div v-if="open" class="tag-picker-menu">
      <input
        ref="inputRef"
        type="text"
        class="tag-picker-input"
        :placeholder="t('addTagPlaceholder')"
        :value="filter"
        @input="filter = ($event.target as HTMLInputElement).value"
        @keydown="onInputKeydown"
      />
      <div class="tag-picker-separator" />
      <template v-for="tag in displayedTags" :key="tag">
        <button
          class="tag-picker-item"
          @click="toggleTag(tag)"
        >
          <span class="tag-picker-check">{{ currentTags.includes(tag) ? '✓' : '' }}</span>
          <span class="tag-picker-hash">#</span>{{ tag }}
        </button>
      </template>
      <button
        v-if="createable"
        class="tag-picker-item tag-picker-create"
        @click="applyCustomTag"
      >
        + Create "<strong>#{{ filter }}</strong>"
      </button>
      <div v-if="displayedTags.length === 0 && !createable" class="tag-picker-empty">
        No matches
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import Button from "primevue/button";
import { useI18n } from "../composables/i18n";
import { parseTags } from "./conversationDrawerShared";
import { api } from "../../services/api";
import type { Conversation } from "../../types";

const SUGGESTED_TAGS = ["explore", "plan", "ongoing", "human-verify", "revise", "done"];

const { t } = useI18n();

const props = defineProps<{
  conversation: Conversation | undefined;
  onUpdated?: (conversation: Conversation) => void;
}>();

const open = ref(false);
const filter = ref("");
const wrapperRef = ref<HTMLElement | null>(null);
const inputRef = ref<HTMLInputElement | null>(null);

const currentTags = computed(() => (props.conversation ? parseTags(props.conversation) : []));

const displayedTags = computed(() => {
  const q = filter.value.trim().toLowerCase();
  if (!q) return SUGGESTED_TAGS;
  return SUGGESTED_TAGS.filter((t) => t.toLowerCase().includes(q));
});

const createable = computed(() => {
  const v = filter.value.trim().replace(/^#+/, "").toLowerCase();
  return v.length > 0 && !SUGGESTED_TAGS.some((t) => t.toLowerCase() === v);
});

async function saveTags(next: string[]) {
  const normalized: string[] = [];
  const seen = new Set<string>();
  for (const tag of next) {
    const trimmed = tag.trim();
    if (!trimmed || seen.has(trimmed)) continue;
    seen.add(trimmed);
    normalized.push(trimmed);
  }
  try {
    const updated = await api.updateConversationTags(
      props.conversation!.conversation_id,
      normalized,
    );
    props.onUpdated?.(updated);
  } catch (err) {
    console.error("Failed to update tags:", err);
  }
}

async function toggleTag(tag: string) {
  if (!props.conversation) return;
  const next = currentTags.value.includes(tag)
    ? currentTags.value.filter((t) => t !== tag)
    : [...currentTags.value, tag];
  await saveTags(next);
}

async function applyCustomTag() {
  if (!props.conversation) return;
  const value = filter.value.trim().replace(/^#+/, "");
  if (!value || currentTags.value.includes(value)) return;
  filter.value = "";
  await saveTags([...currentTags.value, value]);
}

function onInputKeydown(e: KeyboardEvent) {
  if (e.key === "Enter") {
    e.preventDefault();
    if (createable.value) {
      applyCustomTag();
    } else if (displayedTags.value.length === 1) {
      toggleTag(displayedTags.value[0]);
    }
  } else if (e.key === "Escape") {
    open.value = false;
  }
}

function toggleOpen() {
  open.value = !open.value;
  if (open.value) {
    filter.value = "";
    nextTick(() => inputRef.value?.focus());
  }
}

// --- Outside-click ---
function onOutside(e: MouseEvent) {
  if (wrapperRef.value && !wrapperRef.value.contains(e.target as Node)) {
    open.value = false;
  }
}
watch(open, (isOpen) => {
  if (isOpen) document.addEventListener("mousedown", onOutside);
  else document.removeEventListener("mousedown", onOutside);
});
</script>
