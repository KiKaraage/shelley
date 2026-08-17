<!-- Floating dropdown create/edit modal for tasks. No header, no title, no X;
     closes via Escape / Cancel / outside-click. Outside-click closes without
     clearing the draft; Cancel clears it. Tags are derived from #hashtags in
     the title (no tag picker). -->
<template>
  <Teleport to="body">
    <Transition name="p-anchored-overlay">
      <div v-if="isOpen" class="task-modal-wrap">
        <div class="task-modal" ref="modalRef" role="dialog" aria-modal="true">
        <div class="task-modal-body">
          <div class="field">
            <label for="task-modal-title">
              {{ t("taskTitleLabel") }}
              <span class="req">*</span>
            </label>
            <div class="title-editor">
              <div
                ref="titleInput"
                class="input task-title-input title-editable"
                contenteditable="true"
                role="textbox"
                aria-multiline="true"
                :aria-label="t('taskTitleLabel')"
                :data-placeholder="t('taskTitlePlaceholder')"
                @input="onTitleInput"
                @keydown="onTitleKeydown"
                @paste="onTitlePaste"
              ></div>
            </div>
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

          <div v-if="titleError" class="field-error">{{ titleError }}</div>

          <div class="field">
            <div class="dir-target-row">
              <div class="dir-target-col">
                <label for="task-modal-dir">{{ t("targetDirectory") }}</label>
                <div
                  class="dir-folder"
                  :class="{ empty: !selectedDir }"
                  ref="dirFolderRef"
                  :title="dirPath"
                >{{ dirDisplay }}</div>
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
    </Transition>
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
const titleInput = ref<HTMLElement | null>(null);
const dirFolderRef = ref<HTMLElement | null>(null);
const modalRef = ref<HTMLElement | null>(null);
const dirDisplay = ref("");

const editing = ref(false);
const faviconFailed = ref<Set<string>>(new Set());
// When true, reopening the modal keeps the current draft (title + dir) instead
// of resetting. Set by outside-click close; cleared by Cancel.
const preserveDraft = ref(false);

const HASHTAG_RE = /#([a-zA-Z0-9_-]+)/g;

// ---- contenteditable title editor ----
// The title is a contenteditable div whose #hashtags are real styled <span>
// elements. Editing re-renders the spans and restores the caret, so the
// visible text, selection, and caret all live in one layout engine (no
// overlay to drift).

// Current caret position as a character offset into the plain text.
function caretCharOffset(el: HTMLElement): number {
  const sel = el.ownerDocument.getSelection();
  if (!sel || sel.rangeCount === 0) return (el.textContent ?? "").length;
  const range = sel.getRangeAt(0);
  const node = range.startContainer;
  const offset = range.startOffset;
  if (node === el) return offset;
  let count = 0;
  const walker = el.ownerDocument.createTreeWalker(el, NodeFilter.SHOW_TEXT);
  let n: Node | null;
  while ((n = walker.nextNode())) {
    if (n === node) return count + offset;
    count += (n.textContent ?? "").length;
  }
  return count;
}

// Place the caret at a character offset into the plain text.
function setCaret(el: HTMLElement, charOffset: number) {
  const walker = el.ownerDocument.createTreeWalker(el, NodeFilter.SHOW_TEXT);
  let remaining = charOffset;
  let node: Node | null;
  let lastNode: Node | null = null;
  while ((node = walker.nextNode())) {
    const len = (node.textContent ?? "").length;
    if (remaining <= len) {
      const range = el.ownerDocument.createRange();
      range.setStart(node, remaining);
      range.collapse(true);
      const sel = el.ownerDocument.getSelection();
      sel?.removeAllRanges();
      sel?.addRange(range);
      return;
    }
    remaining -= len;
    lastNode = node;
  }
  // Past the end: place at the end of the last text node (or the element).
  const target = lastNode ?? el;
  const range = el.ownerDocument.createRange();
  range.setStart(target, (target.textContent ?? "").length);
  range.collapse(true);
  const sel = el.ownerDocument.getSelection();
  sel?.removeAllRanges();
  sel?.addRange(range);
}

// Re-render the title as plain text + tag spans, restoring the caret.
function renderTitle(el: HTMLElement) {
  const caret = caretCharOffset(el);
  const segments: { text: string; tag: boolean }[] = [];
  let last = 0;
  for (const match of title.value.matchAll(HASHTAG_RE)) {
    const idx = match.index ?? 0;
    if (idx > last) segments.push({ text: title.value.slice(last, idx), tag: false });
    segments.push({ text: match[0], tag: true });
    last = idx + match[0].length;
  }
  if (last < title.value.length) segments.push({ text: title.value.slice(last), tag: false });
  el.textContent = "";
  for (const seg of segments) {
    if (seg.tag) {
      const span = el.ownerDocument.createElement("span");
      span.className = "tag-inline";
      span.textContent = seg.text;
      el.appendChild(span);
    } else {
      el.appendChild(el.ownerDocument.createTextNode(seg.text));
    }
  }
  setCaret(el, caret);
}

function dirLabel(cwd: string): string {
  const parts = cwd.split("/").filter(Boolean);
  return parts.length ? parts[parts.length - 1] : cwd;
}

const dirPath = computed(() => selectedDir.value || t("noDirectory"));

// Truncate the full path at the start (…tail) so the leaf directory stays
// visible, measured against the folder's actual width. Uses a hidden
// measuring span so we don't rely on direction: rtl.
function truncateStart(path: string, maxWidth: number): string {
  if (!path || maxWidth <= 0) return path;
  const probe = document.createElement("span");
  probe.style.cssText =
    "position:absolute;visibility:hidden;white-space:nowrap;font-size:0.8125rem;font-family:var(--font-sans);";
  document.body.appendChild(probe);
  const fits = (s: string) => {
    probe.textContent = s;
    return probe.getBoundingClientRect().width <= maxWidth;
  };
  try {
    if (fits(path)) return path;
    const parts = path.split("/");
    let tail = "";
    for (let i = parts.length - 1; i >= 0; i--) {
      const candidate = (tail ? parts[i] + "/" + tail : parts[i]);
      if (!fits("…" + candidate)) break;
      tail = candidate;
    }
    return tail ? "…" + tail : "…";
  } finally {
    probe.remove();
  }
}

function updateDirDisplay() {
  const el = dirFolderRef.value;
  if (!el) return;
  dirDisplay.value = truncateStart(dirPath.value, el.clientWidth);
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
      document.addEventListener("mousedown", onDocumentMouseDown);
      nextTick(() => {
        if (titleInput.value) {
          renderTitle(titleInput.value);
          titleInput.value.focus();
        }
        updateDirDisplay();
      });
    } else {
      document.removeEventListener("mousedown", onDocumentMouseDown);
    }
  },
);

// Recompute the truncated path when the folder's width changes (e.g. the
// modal is resized or the window changes width).
let resizeObserver: ResizeObserver | null = null;
watch(dirFolderRef, (el) => {
  resizeObserver?.disconnect();
  resizeObserver = null;
  if (el) {
    resizeObserver = new ResizeObserver(() => updateDirDisplay());
    resizeObserver.observe(el);
  }
});

// Recompute when the selected directory changes (pill click / browse pick).
watch(dirPath, () => updateDirDisplay());

onBeforeUnmount(() => {
  resizeObserver?.disconnect();
  resizeObserver = null;
  document.removeEventListener("mousedown", onDocumentMouseDown);
});

// Outside-click closes without clearing the draft; Cancel clears it.
function onDocumentMouseDown(e: MouseEvent) {
  if (modalRef.value && !modalRef.value.contains(e.target as Node)) {
    preserveDraft.value = true;
    emit("close");
  }
}

function onTitleInput() {
  const el = titleInput.value;
  if (!el) return;
  // Sync the plain-text title from the editable content, then re-render the
  // tag spans and restore the caret.
  title.value = el.textContent ?? "";
  renderTitle(el);
  if (titleError.value) titleError.value = "";
}

// Enter inserts a newline; pressing Enter on an empty line (or a double Enter)
// submits. Shift+Enter always inserts a newline.
function onTitleKeydown(e: KeyboardEvent) {
  if (e.key !== "Enter" || e.shiftKey) return;
  const el = e.target as HTMLElement;
  const before = (el.textContent ?? "").slice(0, caretCharOffset(el));
  const lineStart = before.lastIndexOf("\n") + 1;
  const currentLine = before.slice(lineStart);
  if (currentLine.trim() === "") {
    e.preventDefault();
    save();
  }
}

// Paste as plain text so no foreign markup (spans, divs) enters the editor.
function onTitlePaste(e: ClipboardEvent) {
  e.preventDefault();
  const text = e.clipboardData?.getData("text/plain") ?? "";
  const el = titleInput.value;
  if (!el) return;
  const sel = el.ownerDocument.getSelection();
  if (sel && sel.rangeCount) {
    sel.deleteFromDocument();
  }
  el.appendChild(el.ownerDocument.createTextNode(text));
  onTitleInput();
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
