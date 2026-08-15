<!-- Floating dropdown create/edit modal for tasks. No header, no title, no X;
     closes via Escape / Cancel / outside-click. Outside-click closes without
     clearing the draft; Cancel clears it. Tags are derived from #hashtags in
     the title (no tag picker). -->
<template>
  <Teleport to="body">
    <div v-if="isOpen" class="task-modal-wrap">
      <div class="task-modal" ref="modalRef" role="dialog" aria-modal="true">
        <div class="task-modal-body">
          <div class="field">
            <label for="task-modal-title">
              {{ t("taskTitleLabel") }}
              <span class="req">*</span>
            </label>
            <textarea
              id="task-modal-title"
              ref="titleInput"
              v-model="title"
              class="input task-title-input"
              rows="3"
              :placeholder="t('taskTitlePlaceholder')"
              @input="onTitleInput"
              @keydown="onTitleKeydown"
            />
          </div>

          <!-- Directory pills: above the Tags row, separate from Target directory. -->
          <div class="dir-pills-row">
            <div class="dir-pills">
              <button
                v-for="opt in dirOptions"
                :key="opt.path"
                :class="['filter', 'dir-pill', { active: selectedDir === opt.path }]"
                @click="selectedDir = selectedDir === opt.path ? '' : opt.path"
              >
                <img
                  v-if="opt.favicon && !faviconFailed.has(opt.path)"
                  class="pill-favicon"
                  :src="opt.favicon"
                  alt=""
                  @error="onFaviconError(opt.path)"
                />
                <span>{{ opt.label }}</span>
              </button>
              <span v-if="!dirOptions.length" class="dir-pills-empty">{{ t("noDirOptions") }}</span>
            </div>
          </div>

          <div class="tag-hint">
            <span class="lbl">{{ t("tagsLabel") }}</span>
            <span v-if="derivedTags.length" class="tag-hint-chips">
              <span
                v-for="tag in derivedTags"
                :key="tag"
                :class="['tag', { 'done-tag': tag === 'done' }]"
              >
                <span class="hash">#</span>{{ tag }}
              </span>
            </span>
            <span v-else class="tag-hint-empty">{{ t("typeTagInTitle") }}</span>
          </div>
          <div v-if="titleError" class="field-error">{{ titleError }}</div>

          <div class="field">
            <div class="dir-target-row">
              <div class="dir-target-col">
                <label for="task-modal-dir">{{ t("targetDirectory") }}</label>
                <div class="dir-folder" :class="{ empty: !selectedDir }">{{ selectedDir || t("noDirectory") }}</div>
              </div>
              <button
                class="browse-btn browse-circle"
                :title="t('browse')"
                :aria-label="t('browse')"
                @click="openBrowse"
              >
                <i class="pi pi-folder-open" aria-hidden="true" />
              </button>
            </div>
          </div>
        </div>

        <div class="task-modal-foot">
          <button class="btn" @click="onCancel">{{ t("cancel") }}</button>
          <button class="btn btn-primary" @click="save">
            {{ editing ? t("saveChanges") : t("createTask") }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>

  <DirectoryPickerModal
    :is-open="browseOpen"
    :initial-path="selectedDir || undefined"
    folders-only
    @close="browseOpen = false"
    @select="onBrowseSelect"
  />
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue";
import DirectoryPickerModal from "./DirectoryPickerModal.vue";
import { useI18n } from "../composables/i18n";
import type { Task } from "../../services/api_tasks";

const { t } = useI18n();

const props = defineProps<{
  isOpen: boolean;
  task: Task | null; // null = create mode
  gitRoots: string[];
  cwds: string[];
  // Maps a git root path to its owner/repo slug (e.g. "kikaraage/shelley").
  repoNames?: Record<string, string>;
  // Most recent cwd (from the latest thread); auto-populated in create mode.
  latestCwd?: string | null;
}>();

const emit = defineEmits<{
  (e: "close"): void;
  (e: "save", input: { title: string; cwd: string | null }): void;
}>();

const title = ref("");
const selectedDir = ref("");
const titleError = ref("");
const browseOpen = ref(false);
const titleInput = ref<HTMLTextAreaElement | null>(null);
const modalRef = ref<HTMLElement | null>(null);

const editing = ref(false);
const derivedTags = ref<string[]>([]);
const faviconFailed = ref<Set<string>>(new Set());
// When true, reopening the modal keeps the current draft (title + dir) instead
// of resetting. Set by outside-click close; cleared by Cancel.
const preserveDraft = ref(false);

const HASHTAG_RE = /#([a-zA-Z0-9_-]+)/g;

function deriveTags(title: string): string[] {
  const out: string[] = [];
  const m = title.matchAll(HASHTAG_RE);
  for (const match of m) {
    if (!out.includes(match[1])) out.push(match[1]);
  }
  return out;
}

function dirLabel(cwd: string): string {
  const parts = cwd.split("/").filter(Boolean);
  return parts.length ? parts[parts.length - 1] : cwd;
}

// ---- directory pills (deduped git roots first, then recent cwds) ----
// Pill labels use the owner/repo slug when the repo has one, else the folder
// name. The Target directory field below always shows the full local path.
const dirOptions = computed(() => {
  const seen = new Set<string>();
  const out: { path: string; label: string; favicon?: string }[] = [];
  const push = (path: string, favicon?: string) => {
    if (seen.has(path)) return;
    seen.add(path);
    out.push({
      path,
      label: props.repoNames?.[path] || dirLabel(path),
      favicon,
    });
  };
  for (const g of props.gitRoots) push(g, `/api/repo-favicon?root=${encodeURIComponent(g)}`);
  for (const c of props.cwds) push(c);
  return out;
});

function onFaviconError(path: string) {
  faviconFailed.value = new Set(faviconFailed.value).add(path);
}

watch(
  () => props.isOpen,
  (open) => {
    if (open) {
      editing.value = props.task !== null;
      if (!preserveDraft.value) {
        title.value = props.task ? props.task.title : "";
        // Create mode: auto-populate the target dir from the latest thread's cwd
        // (unless editing an existing task, which keeps its own cwd).
        selectedDir.value = props.task
          ? props.task.cwd || ""
          : props.latestCwd || "";
      }
      preserveDraft.value = false;
      titleError.value = "";
      derivedTags.value = deriveTags(title.value);
      document.addEventListener("mousedown", onDocumentMouseDown);
      nextTick(() => titleInput.value?.focus());
    } else {
      document.removeEventListener("mousedown", onDocumentMouseDown);
    }
  },
);

// Outside-click closes without clearing the draft; Cancel clears it.
function onDocumentMouseDown(e: MouseEvent) {
  if (modalRef.value && !modalRef.value.contains(e.target as Node)) {
    preserveDraft.value = true;
    emit("close");
  }
}

onBeforeUnmount(() => {
  document.removeEventListener("mousedown", onDocumentMouseDown);
});

function onTitleInput() {
  derivedTags.value = deriveTags(title.value);
  if (titleError.value) titleError.value = "";
}

// Enter inserts a newline; pressing Enter on an empty line (or a double Enter)
// submits. Shift+Enter always inserts a newline.
function onTitleKeydown(e: KeyboardEvent) {
  if (e.key !== "Enter" || e.shiftKey) return;
  const el = e.target as HTMLTextAreaElement;
  const before = el.value.slice(0, el.selectionStart);
  const lineStart = before.lastIndexOf("\n") + 1;
  const currentLine = before.slice(lineStart);
  if (currentLine.trim() === "") {
    e.preventDefault();
    save();
  }
}

function openBrowse() {
  browseOpen.value = true;
}

function onBrowseSelect(path: string) {
  selectedDir.value = path;
  browseOpen.value = false;
}

// Cancel clears the draft (so the next open resets), unlike outside-click
// which preserves it.
function onCancel() {
  preserveDraft.value = false;
  emit("close");
}

function save() {
  const trimmed = title.value.trim();
  if (!trimmed) {
    titleError.value = t("taskTitleRequired");
    return;
  }
  emit("save", { title: trimmed, cwd: selectedDir.value || null });
}
</script>
