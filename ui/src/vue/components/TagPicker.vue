<!-- Reusable tag picker: renders a popover with suggested tags (filterable),
     a type-to-create input, and toggle-to-apply/remove. The trigger is supplied
     by the parent via the #trigger slot, so the picker can sit in a header,
     drawer, or any other surface. -->
<template>
  <div ref="wrapperRef" :class="`tag-picker-wrapper${className ? ' ' + className : ''}`">
    <slot name="trigger" :open="open" :toggle="toggleOpen" />
    <Teleport to="body" :disabled="!teleport">
      <div
        v-if="open"
        ref="menuRef"
        class="tag-picker-menu"
        :class="{ 'tag-picker-menu-teleported': teleport }"
        :style="teleport ? menuStyle : undefined"
      >
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
    </Teleport>
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
  teleport?: boolean;
}>();

const emit = defineEmits<{
  (e: "update", conversation: Conversation): void;
}>();

const open = ref(false);
const filter = ref("");
const wrapperRef = ref<HTMLElement | null>(null);
const menuRef = ref<HTMLElement | null>(null);
const inputRef = ref<HTMLInputElement | null>(null);
const menuStyle = ref<Record<string, string>>({});

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
    positionMenu();
    nextTick(() => inputRef.value?.focus());
  }
}

function positionMenu() {
  if (!props.teleport) {
    menuStyle.value = {};
    return;
  }
  const anchor = wrapperRef.value;
  if (!anchor) return;
  const rect = anchor.getBoundingClientRect();
  const gap = 4;
  const style: Record<string, string> = {
    position: "fixed",
    left: "auto",
    // Right-align to the trigger so the menu never runs off the viewport's
    // right edge (the drawer trigger sits near the drawer's right edge).
    right: `${Math.max(0, window.innerWidth - rect.right)}px`,
  };
  // Open below the trigger unless the trigger is in the bottom half of the
  // viewport; then open above it. Using top/bottom (rather than a measured
  // height) keeps this cheap and avoids a first-paint flip.
  if (rect.top + rect.height / 2 > window.innerHeight / 2) {
    style.top = "auto";
    style.bottom = `${window.innerHeight - rect.top + gap}px`;
  } else {
    style.top = `${rect.bottom + gap}px`;
    style.bottom = "auto";
  }
  menuStyle.value = style;
}

// Keep a teleported menu glued to the trigger as the drawer scrolls.
function onScrollOrResize() {
  if (props.teleport && open.value) positionMenu();
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
  if (wrapperRef.value && !wrapperRef.value.contains(e.target as Node) && !menuRef.value?.contains(e.target as Node)) {
    open.value = false;
  }
}
watch(open, (isOpen) => {
  if (isOpen) {
    document.addEventListener("mousedown", onOutside);
    window.addEventListener("scroll", onScrollOrResize, true);
    window.addEventListener("resize", onScrollOrResize);
  } else {
    document.removeEventListener("mousedown", onOutside);
    window.removeEventListener("scroll", onScrollOrResize, true);
    window.removeEventListener("resize", onScrollOrResize);
  }
});
</script>
