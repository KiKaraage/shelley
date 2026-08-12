<!-- Row child of ConversationDrawer.vue. Renders one conversation-item (plus
     its expanded subagents). Split out of the parent only to avoid duplicating
     ~200 lines of markup; it has multiple root nodes so it renders inline with
     no wrapper element, preserving the .conversation-group > .conversation-item
     DOM contract the grouping e2e test relies on. All state/handlers come from
     the DrawerCtx inject. Mirrors renderConversationItem in the React source. -->
<template>
  <div
    :class="`conversation-item ${isActive ? 'active' : ''}${isNew ? ' conversation-item-enter' : ''}`"
    :data-conversation-id="conversation.conversation_id"
    style="cursor: pointer"
    @click="onRowClick"
    @auxclick="ctx.handleAuxClick($event, conversation)"
  >
    <div class="drawer-conversation-item-flex-container">
      <!-- Row 1: [favicon · cwd · branch] -->
      <div class="drawer-conversation-header-row drawer-meta-row">
        <img
          v-if="faviconUrl && !faviconFailed"
          :src="faviconUrl"
          class="drawer-favicon"
          @error="faviconFailed = true"
        />
        <span
          v-if="conversation.cwd && ctx.groupBy.value !== 'cwd'"
          class="conversation-cwd"
          :title="conversation.cwd"
        >
          {{ gitRemoteName || ctx.formatCwdForDisplay(conversation.cwd) }}
        </span>
        <template v-if="conversation.cwd && ctx.groupBy.value !== 'cwd' && gitBranchName">
          <GitBranchIcon class="drawer-meta-git-icon" />
          <span class="conversation-cwd drawer-branch-name" :title="gitBranchTooltip">
            {{ gitBranchName }}
          </span>
        </template>

      </div>

      <!-- Row 2: title (left) + actions + working indicator + badge (right) -->
      <div class="drawer-conversation-header-row">
        <div class="drawer-conversation-item-flex-container">
          <input
            v-if="ctx.editingId.value === conversation.conversation_id"
            ref="renameInput"
            type="text"
            :value="ctx.editingSlug.value"
            class="conversation-title drawer-rename-input"
            @input="ctx.editingSlug.value = ($event.target as HTMLInputElement).value"
            @blur="ctx.handleRename(conversation.conversation_id)"
            @keydown="ctx.handleRenameKeyDown($event, conversation.conversation_id)"
            @click.stop
          />
          <div v-else-if="isDraft" class="conversation-title conversation-title-draft">
            {{ ctx.draftLabels.value[conversation.conversation_id] || "draft" }}
          </div>
          <div v-else class="conversation-title">
            <em v-if="!conversation.slug">untitled</em>
            <template v-else>{{ conversation.slug }}</template>
          </div>
        </div>
        <div class="drawer-actions-row drawer-row-actions">
          <template v-if="isDraft">
            <DeleteButton :conversation-id="conversation.conversation_id" />
          </template>
          <template v-else-if="!itemArchived">
            <Button
              class="btn-icon-sm"
              text
              severity="secondary"
              size="small"
              v-tooltip.top="ctx.t('rename')"
              :aria-label="ctx.t('rename')"
              @click="ctx.handleStartRename($event, conversation)"
            >
              <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" class="drawer-icon-size">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  :stroke-width="2"
                  d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                />
              </svg>
            </Button>
            <Button
              class="btn-icon-sm"
              text
              severity="secondary"
              size="small"
              v-tooltip.top="ctx.t('editTags')"
              :aria-label="ctx.t('editTags')"
              @click="ctx.handleOpenTagEditor($event, conversation.conversation_id)"
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
            <Button
              class="btn-icon-sm"
              text
              severity="secondary"
              size="small"
              v-tooltip.top="ctx.t('archive')"
              :aria-label="ctx.t('archive')"
              @click="ctx.handleArchive($event, conversation.conversation_id)"
            >
              <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" class="drawer-icon-size">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  :stroke-width="2"
                  d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
                />
              </svg>
            </Button>
          </template>
        </div>
        <span
          v-if="convState.working && runningSubagentCount === 0"
          class="working-indicator drawer-working-indicator"
          :title="ctx.t('agentIsWorking')"
        />
        <span
          v-if="wasWorking"
          class="working-indicator drawer-working-indicator unread-indicator"
          :title="ctx.t('unreadResponses')"
        />
        <span
          v-if="!isDraft && !itemArchived && hasSubagents"
          class="subagent-count-badge"
          v-tooltip.top="subagentBadgeTooltip"
          :aria-label="isExpanded ? ctx.t('collapseSubagents') : ctx.t('expandSubagents')"
          @click="ctx.toggleSubagents($event, conversation.conversation_id)"
        >
          <span
            v-if="runningSubagentCount > 0"
            class="working-indicator"
            data-testid="subagent-badge-running"
            aria-hidden="true"
          />
          <span class="drawer-subagent-count-badge-text">{{ subagentBadgeText }}</span>
          <svg
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            :class="`drawer-subagent-chevron ${isExpanded ? 'drawer-subagent-chevron-expanded' : 'drawer-subagent-chevron-collapsed'}`"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              :stroke-width="2"
              d="M9 5l7 7-7 7"
            />
          </svg>
        </span>
      </div>

      <!-- Tags / tag editor -->
      <div
        v-if="tagsEditing || conversationTags.length > 0"
        :ref="setTagEditorRefMaybe"
        :class="`conversation-tags${tagsEditing ? ' conversation-tags-editing' : ''}`"
        @click="tagsEditing ? $event.stopPropagation() : undefined"
      >
        <template v-for="tag in conversationTags" :key="tag">
          <span v-if="tagsEditing" class="conversation-tag conversation-tag-removable">
            <span class="conversation-tag-hash">#</span>
            <span class="conversation-tag-text">{{ tag }}</span>
            <button
              type="button"
              class="conversation-tag-remove"
              :aria-label="`${ctx.t('removeTag')} ${tag}`"
              v-tooltip.top="ctx.t('removeTag')"
              @click="ctx.handleRemoveTag(conversation, tag)"
            >
              ×
            </button>
          </span>
          <!-- Not editing: clicking a chip toggles it in the drawer's tag
               filter. While the editor is open the chips mean "remove". -->
          <button
            v-else
            type="button"
            :class="`conversation-tag conversation-tag-filterable${tagFiltered(tag) ? ' conversation-tag-filter-on' : ''}`"
            :title="`#${tag}`"
            :aria-pressed="tagFiltered(tag)"
            data-testid="conversation-tag-chip"
            :data-tag="tag"
            @click="onTagChipClick($event, tag)"
          >
            <span class="conversation-tag-hash">#</span>{{ tag }}
          </button>
        </template>
        <form
          v-if="tagsEditing"
          class="conversation-tag-inline-form"
          @submit.prevent="onTagSubmit"
        >
          <span class="conversation-tag-hash">#</span>
          <input
            ref="tagInput"
            type="text"
            :value="ctx.tagInput.value"
            :placeholder="ctx.t('addTagPlaceholder')"
            class="conversation-tag-inline-input"
            autocomplete="off"
            autocapitalize="off"
            spellcheck="false"
            role="combobox"
            aria-autocomplete="list"
            :aria-expanded="tagMenuOpen"
            @input="onTagInput"
            @keydown="onTagInputKeyDown"
            @focus="onTagInputFocus"
            @blur="onTagInputBlur"
          />
          <!-- Suggestion dropdown, teleported out of the drawer's clipping
               overflow and pinned under the input. Mirrors the search box's
               `tag:` menu: substring matches, arrow/Enter selection, counts. -->
          <Teleport to="body">
            <div
              v-if="tagMenuOpen"
              class="tag-filter-menu tag-editor-menu"
              :style="tagMenuStyle"
              data-testid="tag-editor-menu"
              role="listbox"
              @mousedown.prevent
            >
              <div class="tag-filter-options scrollable">
                <button
                  v-for="(offer, i) in tagOffers"
                  :key="offer.tag"
                  type="button"
                  role="option"
                  :aria-selected="i === tagHighlightIndex"
                  :class="`tag-filter-option${i === tagHighlightIndex ? ' highlighted' : ''}`"
                  data-testid="tag-editor-option"
                  :data-tag="offer.tag"
                  @mousemove="tagHighlightIndex = i"
                  @click="chooseTagOffer(offer.tag)"
                >
                  <span class="tag-filter-option-name">
                    <span class="conversation-tag-hash">#</span>{{ offer.tag }}
                  </span>
                  <span class="tag-filter-option-count">{{ offer.count }}</span>
                </button>
              </div>
            </div>
          </Teleport>
        </form>
      </div>

      <!-- Row 3: timestamp + preview -->
      <div class="conversation-meta">
        <span class="conversation-date">{{ ctx.formatDate(conversation.updated_at) }}</span>
        <span
          v-if="convState.search_snippet"
          class="conversation-preview conversation-snippet"
          :title="stripSnippetMarks(convState.search_snippet)"
        >
          <template v-for="(seg, i) in renderSnippetSegments(convState.search_snippet)" :key="i">
            <mark v-if="seg.mark" class="conversation-snippet-mark">{{ seg.text }}</mark>
            <template v-else>{{ seg.text }}</template>
          </template>
        </span>
        <span
          v-else-if="isDraft"
          class="conversation-preview"
          :title="conversation.draft?.trim() || undefined"
        >
          {{ conversation.draft?.trim() || "\u00a0" }}
        </span>
        <span v-else class="conversation-preview" :title="convState.preview || undefined">
          {{ convState.preview || "\u00a0" }}
        </span>
        <!-- Terminal count. Only shown when the conversation has more than
             one terminal pinned to it: a single terminal is the ordinary
             case and not worth a badge. -->
        <span
          v-if="!isDraft && !itemArchived && terminalCount > 1"
          class="conversation-terminal-count"
          v-tooltip.top="`${terminalCount} ${ctx.t('terminalsPinnedHere')}`"
          :aria-label="`${terminalCount} ${ctx.t('terminalsPinnedHere')}`"
        >
          <svg
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            class="conversation-terminal-count-icon"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              :stroke-width="2"
              d="M8 9l3 3-3 3m5 0h3M4 5h16a1 1 0 011 1v12a1 1 0 01-1 1H4a1 1 0 01-1-1V6a1 1 0 011-1z"
            />
          </svg>
          {{ terminalCount }}
        </span>
      </div>
    </div>

    <div v-if="itemArchived" class="conversation-actions drawer-actions-row-offset">
      <Button
        class="btn-icon-sm"
        text
        severity="secondary"
        size="small"
        v-tooltip.top="ctx.t('restore')"
        :aria-label="ctx.t('restore')"
        @click="ctx.handleUnarchive($event, conversation.conversation_id)"
      >
        <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" class="drawer-icon-size">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            :stroke-width="2"
            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
          />
        </svg>
      </Button>
      <DeleteButton :conversation-id="conversation.conversation_id" />
    </div>
  </div>

  <!-- Subagents -->
  <div
    v-if="!itemArchived && isExpanded && conversationSubagents.length > 0"
    class="subagent-list drawer-subagent-list"
  >
    <div
      v-for="sub in conversationSubagents"
      :key="sub.conversation_id"
      :class="`conversation-item subagent-item drawer-subagent-item-style ${sub.conversation_id === ctx.currentConversationId.value ? 'active' : ''}${ctx.seenIds.value !== null && !ctx.seenIds.value.has(sub.conversation_id) ? ' conversation-item-enter' : ''}`"
      @click="onSubClick($event, sub)"
      @auxclick="ctx.handleAuxClick($event, sub)"
    >
      <div class="drawer-conversation-item-flex-container">
        <div class="drawer-conversation-header-row">
          <div class="drawer-conversation-item-flex-container">
            <div class="conversation-title">
              <em v-if="!sub.slug">untitled</em>
              <template v-else>{{ sub.slug }}</template>
            </div>
          </div>
          <span
            v-if="sub.working"
            class="working-indicator"
            :title="ctx.t('subagentIsWorking')"
          />
          <span
            v-else-if="ctx.seenWorkingIds.value.has(sub.conversation_id) && sub.conversation_id !== ctx.currentConversationId.value"
            class="working-indicator unread-indicator"
            :title="ctx.t('unreadResponses')"
          />
        </div>
        <div class="conversation-meta">
          <span class="conversation-date ">{{
            ctx.formatDate(sub.updated_at)
          }}</span>
          <div class="conversation-preview" :title="sub.preview || undefined">
            {{ sub.preview || "\u00a0" }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, inject, nextTick, onBeforeUnmount, ref, watch, type VNode } from "vue";
import Button from "primevue/button";
import GitBranchIcon from "./GitBranchIcon.vue";
import type { Conversation, ConversationWithState } from "../../types";
import { isImeComposing } from "../../utils/imeComposing";
import {
  DrawerCtxKey,
  parseTags,
  stripSnippetMarks,
  renderSnippetSegments,
} from "./conversationDrawerShared";
import { isTagSelected } from "../../utils/tagFilter";
import { perfCount } from "../../utils/perf";

const props = defineProps<{
  conversation: Conversation | ConversationWithState;
}>();

const ctx = inject(DrawerCtxKey)!;

const renameInput = ref<HTMLInputElement | null>(null);
const tagInput = ref<HTMLInputElement | null>(null);

const convState = computed(() => props.conversation as ConversationWithState);
const isDraft = computed(() => !!props.conversation.is_draft);
const gitRepoName = computed(() => {
  const root = convState.value.git_worktree_root || convState.value.git_repo_root;
  return root ? root.split("/").pop() || null : null;
});
// Favicon URL for the repo root. The server resolves the favicon from
// well-known paths and <link rel="icon"> declarations in the repo.
const repoRootForFavicon = computed(() => convState.value.git_repo_root || props.conversation.cwd || null);
const faviconFailed = ref(false);
const faviconUrl = computed(() => {
  const root = repoRootForFavicon.value;
  if (!root) return "";
  return `/api/repo-favicon?root=${encodeURIComponent(root)}`;
});
// Reset favicon failure state when the repo root changes (new conversation).
watch(repoRootForFavicon, () => { faviconFailed.value = false; });
// The "owner/repo" slug from the origin remote, shown in place of the cwd
// folder name when available.
const gitRemoteName = computed(() => convState.value.git_remote || null);
// The git branch name shown as the second meta item on the header row. Falls
// back to the repo name when the branch is unavailable (e.g. detached HEAD).
const gitBranchName = computed(() => convState.value.git_branch || gitRepoName.value);
const gitBranchTooltip = computed(() =>
  convState.value.git_branch
    ? `${gitRepoName.value || ""} · ${convState.value.git_branch}`.trim()
    : convState.value.git_worktree_root || convState.value.git_repo_root || "",
);
const isActive = computed(
  () => props.conversation.conversation_id === ctx.currentConversationId.value,
);
const conversationSubagents = computed<ConversationWithState[]>(() =>
  isDraft.value ? [] : ctx.subagentsByParent.value[props.conversation.conversation_id] || [],
);
const subagentCount = computed(() =>
  isDraft.value ? 0 : conversationSubagents.value.length || convState.value.subagent_count || 0,
);
const hasSubagents = computed(() => subagentCount.value > 0);
// Live terminals pinned to this conversation. Badged only when > 1.
const terminalCount = computed(
  () => ctx.terminalCounts.value[props.conversation.conversation_id] ?? 0,
);
// How many of this conversation's subagents are currently working. Drives
// the working ring + "running/total" split on the count badge.
const runningSubagentCount = computed(
  () => conversationSubagents.value.filter((s) => s.working).length,
);
const subagentBadgeText = computed(() =>
  runningSubagentCount.value > 0
    ? `${runningSubagentCount.value}/${subagentCount.value}`
    : `${subagentCount.value}`,
);
const subagentBadgeTooltip = computed(() => {
  const base = isExpanded.value ? ctx.t("hideSubagents") : ctx.t("showSubagents");
  return runningSubagentCount.value > 0
    ? `${base} (${runningSubagentCount.value} ${ctx.t("running")})`
    : base;
});
const isExpanded = computed(() =>
  ctx.expandedSubagents.value.has(props.conversation.conversation_id),
);
const itemArchived = computed(() => props.conversation.archived);
const isNew = computed(
  () =>
    !isDraft.value &&
    ctx.seenIds.value !== null &&
    !ctx.seenIds.value.has(props.conversation.conversation_id),
);
const conversationTags = computed(() => {
  perfCount("drawerRow.tags");
  return isDraft.value ? [] : parseTags(props.conversation);
});
const tagsEditing = computed(
  () => !isDraft.value && ctx.tagEditorId.value === props.conversation.conversation_id,
);
function tagFiltered(tag: string): boolean {
  return isTagSelected(ctx.selectedTags.value, tag);
}
function onTagChipClick(e: MouseEvent, tag: string) {
  // Don't let the click also select the conversation.
  e.stopPropagation();
  ctx.toggleTagFilter(tag);
}

// Track when a conversation finishes working so we can show a "was working" blue dot.
const wasWorking = computed(() =>
  ctx.seenWorkingIds.value.has(convState.value.conversation_id) &&
  !convState.value.working &&
  !isDraft.value &&
  convState.value.conversation_id !== ctx.currentConversationId.value,
);
watch(
  () => convState.value.working,
  (prev, was) => {
    if (was && !prev && convState.value.conversation_id !== ctx.currentConversationId.value)
      ctx.seenWorkingIds.value.add(convState.value.conversation_id);
  },
);

// Track subagent working→idle transitions so the blue dot appears on subagent rows too.
const prevWorkingSubIds = new Set<string>();
watch(
  conversationSubagents,
  (subs) => {
    const nowWorking = new Set(subs.filter((s) => s.working).map((s) => s.conversation_id));
    for (const id of prevWorkingSubIds) {
      if (!nowWorking.has(id) && id !== ctx.currentConversationId.value)
        ctx.seenWorkingIds.value.add(id);
    }
    prevWorkingSubIds.clear();
    for (const id of nowWorking) prevWorkingSubIds.add(id);
  },
  { deep: true },
);

function onRowClick(e: MouseEvent) {
  if (ctx.handleModifiedClick(e, props.conversation)) return;
  ctx.selectConversation(props.conversation);
}
function onSubClick(e: MouseEvent, sub: Conversation) {
  if (ctx.handleModifiedClick(e, sub)) return;
  ctx.selectConversation(sub);
}

// --- Tag editor dropdown ---------------------------------------------------
// The row's "Edit tags" input opens a suggestion dropdown, mirroring the
// search box's `tag:` menu: existing tags matching what's typed anywhere in
// their text (not just as a prefix), ranked best-first, each with its usage
// count. Arrow keys move the highlight; Enter commits the highlighted match
// (the best one, by default), or the typed text when it matches nothing.
const tagInputFocused = ref(false);
// Escape closes the menu without closing the editor; latched until the typed
// text changes so it does not immediately reopen on the next keystroke.
const tagMenuDismissed = ref(false);
const tagHighlightIndex = ref(0);

const tagOffers = computed(() =>
  tagsEditing.value ? ctx.matchTagOffers(props.conversation, ctx.tagInput.value) : [],
);
const tagMenuOpen = computed(
  () =>
    tagsEditing.value &&
    tagInputFocused.value &&
    !tagMenuDismissed.value &&
    tagOffers.value.length > 0,
);

// The menu is teleported to <body> (the drawer's overflow would clip it), so
// it is positioned in viewport coordinates under the input. Recomputed on
// open and whenever the drawer scrolls or the window resizes.
const menuRect = ref({ left: 0, top: 0, width: 0 });
const tagMenuStyle = computed(() => ({
  left: `${menuRect.value.left}px`,
  top: `${menuRect.value.top}px`,
  width: `${menuRect.value.width}px`,
}));
function updateMenuRect() {
  const el = tagInput.value;
  if (!el) return;
  const r = el.getBoundingClientRect();
  menuRect.value = { left: r.left, top: r.bottom + 4, width: Math.max(r.width, 160) };
}
watch(tagMenuOpen, (open) => {
  if (!open) {
    window.removeEventListener("scroll", updateMenuRect, true);
    window.removeEventListener("resize", updateMenuRect);
    return;
  }
  void nextTick(updateMenuRect);
  window.addEventListener("scroll", updateMenuRect, true);
  window.addEventListener("resize", updateMenuRect);
});
onBeforeUnmount(() => {
  window.removeEventListener("scroll", updateMenuRect, true);
  window.removeEventListener("resize", updateMenuRect);
});
// Keep the highlight in range as the offered set narrows.
watch(tagOffers, () => {
  if (tagHighlightIndex.value >= tagOffers.value.length) tagHighlightIndex.value = 0;
});

function onTagInput(e: Event) {
  ctx.tagInput.value = (e.target as HTMLInputElement).value;
  tagMenuDismissed.value = false;
  tagHighlightIndex.value = 0;
}
function onTagInputFocus() {
  tagInputFocused.value = true;
}
// A click on an option keeps focus (its @mousedown.prevent), so blurring means
// the field really lost focus: close the menu.
function onTagInputBlur() {
  tagInputFocused.value = false;
}
async function commitTag(tag: string) {
  await ctx.handleAddTag(props.conversation, tag);
  tagMenuDismissed.value = false;
  tagHighlightIndex.value = 0;
  await nextTick();
  tagInput.value?.focus();
}
function chooseTagOffer(tag: string) {
  void commitTag(tag);
}
// The form's submit path: Enter with the menu open commits the highlighted
// offer; otherwise it commits exactly what was typed (a brand-new tag).
async function onTagSubmit() {
  const best = tagMenuOpen.value ? tagOffers.value[tagHighlightIndex.value] : undefined;
  await commitTag(best ? best.tag : ctx.tagInput.value);
}
function onTagInputKeyDown(e: KeyboardEvent) {
  if (isImeComposing(e)) return;
  if (tagMenuOpen.value) {
    const n = tagOffers.value.length;
    if (e.key === "ArrowDown") {
      e.preventDefault();
      tagHighlightIndex.value = (tagHighlightIndex.value + 1) % n;
      return;
    }
    if (e.key === "ArrowUp") {
      e.preventDefault();
      tagHighlightIndex.value = (tagHighlightIndex.value - 1 + n) % n;
      return;
    }
  }
  if (e.key === "Escape") {
    e.preventDefault();
    // Peel one layer: first the open menu, then the editor.
    if (tagMenuOpen.value) {
      tagMenuDismissed.value = true;
      return;
    }
    ctx.tagEditorId.value = null;
    ctx.tagInput.value = "";
  }
}

// Forward the active rename/tag-editor DOM refs up to the parent so its
// focus/select/outside-click logic can reach them (mirrors the React refs,
// which are bound only on the active row).
function setTagEditorRef(el: Element | null) {
  ctx.tagEditorRef.value = (el as HTMLElement) ?? null;
}
// Bound unconditionally; only writes the shared ref while this row is the
// active tag editor (and clears it back to null otherwise via the v-if).
const setTagEditorRefMaybe = (el: unknown) => {
  if (tagsEditing.value) setTagEditorRef((el as Element) ?? null);
};
watch(renameInput, (el) => {
  if (el) ctx.renameInputRef.value = el;
});
watch(tagInput, (el) => {
  if (el) ctx.tagInputRef.value = el;
});

// Inline delete button (mirrors renderDeleteButton). Defined as a render
// component so the confirm/trash markup isn't duplicated in template.
const DeleteButton = defineComponent({
  props: { conversationId: { type: String, required: true } },
  setup(p) {
    const checkIcon = () =>
      h(
        "svg",
        { fill: "none", stroke: "currentColor", viewBox: "0 0 24 24", class: "drawer-icon-size" },
        [
          h("path", {
            "stroke-linecap": "round",
            "stroke-linejoin": "round",
            "stroke-width": 2,
            d: "M5 13l4 4L19 7",
          }),
        ],
      );
    const xIcon = () =>
      h(
        "svg",
        { fill: "none", stroke: "currentColor", viewBox: "0 0 24 24", class: "drawer-icon-size" },
        [
          h("path", {
            "stroke-linecap": "round",
            "stroke-linejoin": "round",
            "stroke-width": 2,
            d: "M6 18L18 6M6 6l12 12",
          }),
        ],
      );
    const trashIcon = () =>
      h(
        "svg",
        { fill: "none", stroke: "currentColor", viewBox: "0 0 24 24", class: "drawer-icon-size" },
        [
          h("path", {
            "stroke-linecap": "round",
            "stroke-linejoin": "round",
            "stroke-width": 2,
            d: "M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16",
          }),
        ],
      );
    return () => {
      if (ctx.pendingDeleteId.value === p.conversationId) {
        return h(
          "div",
          {
            class: "drawer-delete-confirm",
            ref: (el: unknown) => (ctx.pendingDeleteRef.value = (el as HTMLElement) ?? null),
            onClick: (e: MouseEvent) => e.stopPropagation(),
            title: ctx.t("confirmDelete"),
          },
          [
            h("span", { class: "drawer-delete-confirm-label" }, ctx.t("confirmDeleteShort")),
            h(
              "button",
              {
                type: "button",
                class: "btn-icon-sm btn-danger drawer-delete-confirm-yes",
                title: ctx.t("delete_"),
                "aria-label": ctx.t("delete_"),
                onClick: (e: MouseEvent) => ctx.handleConfirmDelete(e, p.conversationId),
              },
              [checkIcon()],
            ),
            h(
              "button",
              {
                type: "button",
                class: "btn-icon-sm",
                title: ctx.t("cancel"),
                "aria-label": ctx.t("cancel"),
                onClick: (e: MouseEvent) => ctx.handleCancelDelete(e),
              },
              [xIcon()],
            ),
          ] as VNode[],
        );
      }
      return h(
        "button",
        {
          class: "btn-icon-sm btn-danger",
          title: ctx.t("deletePermanently"),
          "aria-label": ctx.t("delete_"),
          onClick: (e: MouseEvent) => ctx.handleDeleteClick(e, p.conversationId),
        },
        [trashIcon()],
      );
    };
  },
});
</script>
