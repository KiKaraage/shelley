<!-- Floating dropdown create/edit modal for tasks. No header, no title, no X;
     closes via Escape / Cancel / outside-click. Tags are derived from
     #hashtags in the title (no tag picker). -->
<template>
  <Teleport to="body">
    <div v-if="isOpen" class="task-modal-wrap" @mousedown.self="emit('close')">
      <div class="task-modal" role="dialog" aria-modal="true">
        <div class="task-modal-body">
          <div class="field">
            <label for="task-modal-title">
              {{ t("taskTitleLabel") }}
              <span class="req">*</span>
            </label>
            <input
              id="task-modal-title"
              ref="titleInput"
              v-model="title"
              class="input"
              type="text"
              :placeholder="t('taskTitlePlaceholder')"
              maxlength="140"
              @input="onTitleInput"
              @keydown.enter.prevent="save"
            />
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
          </div>

          <div class="field">
            <label for="task-modal-dir">{{ t("targetDirectory") }}</label>
            <div class="dir-row">
              <div class="select-wrap">
                <select id="task-modal-dir" v-model="selectedDir">
                  <option value="">{{ t("noDirectory") }}</option>
                  <optgroup :label="t('gitRoots')">
                    <option
                      v-for="g in gitRoots"
                      :key="g"
                      :value="g"
                    >
                      {{ dirLabel(g) }}
                    </option>
                  </optgroup>
                  <optgroup :label="t('recentDirectories')">
                    <option
                      v-for="c in cwds"
                      :key="c"
                      :value="c"
                    >
                      {{ dirLabel(c) }}
                    </option>
                  </optgroup>
                </select>
              </div>
              <button class="browse-btn" :title="t('browse')" :aria-label="t('browse')" @click="openBrowse">
                <i class="pi pi-folder-open" aria-hidden="true" />
              </button>
            </div>
            <div v-if="selectedDir" class="dir-note">{{ dirLabel(selectedDir) }}</div>
          </div>
        </div>

        <div class="task-modal-foot">
          <button class="btn" @click="emit('close')">{{ t("cancel") }}</button>
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
import { nextTick, ref, watch } from "vue";
import DirectoryPickerModal from "./DirectoryPickerModal.vue";
import { useI18n } from "../composables/i18n";
import type { Task } from "../../services/api_tasks";

const { t } = useI18n();

const props = defineProps<{
  isOpen: boolean;
  task: Task | null; // null = create mode
  gitRoots: string[];
  cwds: string[];
}>();

const emit = defineEmits<{
  (e: "close"): void;
  (e: "save", input: { title: string; cwd: string | null }): void;
}>();

const title = ref("");
const selectedDir = ref("");
const titleError = ref("");
const browseOpen = ref(false);
const titleInput = ref<HTMLInputElement | null>(null);

const editing = ref(false);
const derivedTags = ref<string[]>([]);

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

watch(
  () => props.isOpen,
  (open) => {
    if (open) {
      editing.value = props.task !== null;
      title.value = props.task ? props.task.title : "";
      selectedDir.value = props.task?.cwd || "";
      titleError.value = "";
      derivedTags.value = deriveTags(title.value);
      nextTick(() => titleInput.value?.focus());
    }
  },
);

function onTitleInput() {
  derivedTags.value = deriveTags(title.value);
  if (titleError.value) titleError.value = "";
}

function openBrowse() {
  browseOpen.value = true;
}

function onBrowseSelect(path: string) {
  selectedDir.value = path;
  browseOpen.value = false;
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
