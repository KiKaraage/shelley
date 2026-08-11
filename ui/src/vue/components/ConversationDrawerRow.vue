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
          v-if="convState.working"
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
            <span class="conversation-tag-hash">#</span>{{ tag }}
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
          <span v-else class="conversation-tag" :title="`#${tag}`">
            <span class="conversation-tag-hash">#</span>{{ tag }}
          </span>
        </template>
        <form
          v-if="tagsEditing"
          class="conversation-tag-inline-form"
          @submit.prevent="ctx.handleAddTag(conversation)"
        >
          <span class="conversation-tag-hash">#</span>
          <input
            ref="tagInput"
            type="text"
            :value="ctx.tagInput.value"
            :placeholder="ctx.t('addTagPlaceholder')"
            class="conversation-tag-inline-input"
            @input="ctx.tagInput.value = ($event.target as HTMLInputElement).value"
            @keydown="onTagInputKeyDown"
          />
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
import { computed, defineComponent, h, inject, ref, watch, type VNode } from "vue";
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
function onTagInputKeyDown(e: KeyboardEvent) {
  if (isImeComposing(e)) {
    // Stop the Enter that confirms an IME conversion from submitting the form.
    if (e.key === "Enter") e.preventDefault();
    return;
  }
  if (e.key === "Escape") {
    e.preventDefault();
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
