<!-- Reusable tag picker: renders a popover with suggested tags (filterable),
     a type-to-create input, and toggle-to-apply/remove. The trigger is supplied
     by the parent via the #trigger slot, so the picker can sit in a header,
     drawer, or any other surface. -->
<template>
  <div ref="wrapperRef" :class="`tag-picker-wrapper${className ? ' ' + className : ''}`">
    <slot name="trigger" :open="open" :toggle="toggleOpen" />
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
      <!-- Template tags -->
      <template v-for="tag in displayedTags" :key="tag">
        <button class="tag-picker-item" @click="toggleTag(tag)">
          <span class="tag-picker-check">{{ currentTags.includes(tag) ? '✓' : '' }}</span>
          <span class="tag-picker-hash">#</span>{{ tag }}
        </button>
      </template>
      <button
        v-if="createable"
        class="tag-picker-item tag-picker-create"
        @click="applyCustomTag"
      >
        + <strong>#{{ filter }}</strong>
      </button>
      <div v-if="displayedTags.length === 0 && !createable && customTags.length === 0" class="tag-picker-empty">
        No matches
      </div>
      <!-- Custom (non-template) tags -->
      <template v-if="customTags.length > 0">
        <div class="tag-picker-separator" />
        <button
          v-for="tag in customTags"
          :key="tag"
          class="tag-picker-item"
          @click="toggleTag(tag)"
        >
          <span class="tag-picker-check">✓</span>
          <span class="tag-picker-hash">#</span>{{ tag }}
        </button>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import { useI18n } from "../composables/i18n";
import { parseTags } from "./conversationDrawerShared";
import { api } from "../../services/api";
import type { Conversation } from "../../types";

const SUGGESTED_TAGS = ["explore", "plan", "verify-plan", "working", "verify-work", "audit", "revise", "done"];

const { t } = useI18n();

const props = defineProps<{
  conversation: Conversation | undefined;
  className?: string;
}>();

const emit = defineEmits<{
  (e: "update", conversation: Conversation): void;
}>();

const open = ref(false);
const filter = ref("");
const wrapperRef = ref<HTMLElement | null>(null);
const inputRef = ref<HTMLInputElement | null>(null);

const currentTags = computed(() => (props.conversation ? parseTags(props.conversation) : []));

const customTags = computed(() =>
  currentTags.value.filter((tag) => !SUGGESTED_TAGS.includes(tag)),
);

const displayedTags = computed(() => {
  const q = filter.value.trim().toLowerCase();
  if (!q) return SUGGESTED_TAGS;
  return SUGGESTED_TAGS.filter((tag) => tag.toLowerCase().includes(q));
});

const createable = computed(() => {
  const v = filter.value.trim().replace(/^#+/, "").toLowerCase();
  return v.length > 0 && !SUGGESTED_TAGS.some((tag) => tag.toLowerCase() === v);
});

function toggleOpen() {
  open.value = !open.value;
  if (open.value) {
    filter.value = "";
    nextTick(() => inputRef.value?.focus());
  }
}

function toggleTag(tag: string) {
  if (!props.conversation) return;
  const next = currentTags.value.includes(tag)
    ? currentTags.value.filter((t) => t !== tag)
    : [...currentTags.value, tag];
  void persist(next);
}

function applyCustomTag() {
  if (!props.conversation) return;
  const value = filter.value.trim().replace(/^#+/, "");
  if (!value || currentTags.value.includes(value)) return;
  filter.value = "";
  void persist([...currentTags.value, value]);
}

async function persist(next: string[]) {
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
    emit("update", updated);
  } catch (err) {
    console.error("Failed to update tags:", err);
  }
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
