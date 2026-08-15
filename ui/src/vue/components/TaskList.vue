<!-- Tasks homepage tab: the task list with repo/dir filter pills, idle |
     handled+done columns, and a floating create/edit modal. -->
<template>
  <div class="tasks-home">
    <div class="filters-row">
      <div class="filters">
        <button
          v-for="pill in filterPills"
          :key="pill.key"
          :class="['filter', { active: filter === pill.key }]"
          @click="filter = pill.key"
        >
          <img
            v-if="pill.favicon && !faviconFailed.has(pill.key)"
            class="pill-favicon"
            :src="pill.favicon"
            alt=""
            @error="onFaviconError(pill.key)"
          />
          <span>{{ pill.label }}</span>
          <span class="filter-count">{{ pill.count }}</span>
        </button>
      </div>
    </div>

    <div class="task-grid">
      <!-- Empty: no tasks at all -->
      <div v-if="tasks.length === 0" class="empty-all">
        <div class="big">{{ t("noTasksYet") }}</div>
        <div>{{ t("noTasksYetHint") }}</div>
        <div class="empty-cta">
          <button class="btn btn-primary" @click="emit('open-create')">
            {{ t("createFirstTask") }}
          </button>
        </div>
      </div>

      <!-- Empty: filter matches nothing -->
      <div v-else-if="filteredTasks.length === 0" class="empty-all">
        <div class="big">{{ t("nothingHere") }}</div>
        <div>{{ t("noTasksIn") }} {{ filterLabel }}.</div>
      </div>

      <!-- Groups: DOM order = handled, idle, done (mobile order) -->
      <template v-else>
        <section v-for="group in groups" :key="group.key" :class="['group', `group-${group.key}`]">
          <div class="group-label">
            {{ groupCopy(group) }}
          </div>
          <div class="group-body">
            <template v-if="group.tasks.length">
              <div
                v-for="task in group.tasks"
                :key="task.task_id"
                :class="['row', { 'is-done': isDone(task) }]"
              >
                <div class="main">
                  <div class="title-row">
                    <span class="task-title" :title="displayTitle(task)">{{
                      displayTitle(task)
                    }}</span>
                    <span class="row-actions">
                      <button
                        class="icon-btn"
                        :title="t('editTask')"
                        v-tooltip.top="t('editTask')"
                        @click="emit('edit', task)"
                      >
                        <i class="pi pi-pencil" aria-hidden="true" />
                      </button>
                      <button
                        class="icon-btn"
                        :title="t('startThread')"
                        v-tooltip.top="t('startThread')"
                        @click="emit('start-thread', task)"
                      >
                        <i class="pi pi-arrow-up-right" aria-hidden="true" />
                      </button>
                      <button
                        :class="[
                          'icon-btn',
                          { 'delete-armed': confirmingDeleteId === task.task_id },
                        ]"
                        :title="confirmingDeleteId === task.task_id ? t('confirmDelete') : t('deleteTask')"
                        v-tooltip.top="confirmingDeleteId === task.task_id ? t('confirmDelete') : t('deleteTask')"
                        @click="onDeleteClick(task)"
                      >
                        <i class="pi pi-trash" aria-hidden="true" />
                      </button>
                    </span>
                  </div>
                  <div class="meta">
                    <span v-if="task.cwd" class="dir-name">{{ dirLabel(task.cwd) }}</span>
                    <span v-if="task.cwd && task.missing" class="missing-badge">{{
                      t("missingOnDisk")
                    }}</span>
                    <button
                      v-if="task.thread_slug"
                      class="thread-badge"
                      :title="t('openLinkedConversation')"
                      @click="emit('open-thread', task)"
                    >
                      <span class="arrow">➤</span>
                      <span class="slug">{{ task.thread_slug }}</span>
                    </button>
                    <span
                      v-for="tag in deriveTags(task.title)"
                      :key="tag"
                      :class="['tag', { 'done-tag': tag === 'done' }]"
                    >
                      <span class="hash">#</span>{{ tag }}
                    </span>
                  </div>
                </div>
              </div>
            </template>
            <div v-else class="group-empty">{{ groupEmptyCopy(group.key) }}</div>
          </div>
        </section>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "../composables/i18n";
import { tasksApi, type Task } from "../../services/api_tasks";

const { t } = useI18n();

const props = defineProps<{
  tasks: Task[];
  directories: { git_roots: string[]; cwds: string[] };
}>();

const emit = defineEmits<{
  (e: "open-create"): void;
  (e: "edit", task: Task): void;
  (e: "start-thread", task: Task): void;
  (e: "delete", task: Task): void;
  (e: "open-thread", task: Task): void;
}>();

const filter = ref<string>("all");
const faviconFailed = ref<Set<string>>(new Set());
const confirmingDeleteId = ref<string | null>(null);

// Two-step delete confirmation: first click arms, second click deletes.
function onDeleteClick(task: Task) {
  if (confirmingDeleteId.value === task.task_id) {
    confirmingDeleteId.value = null;
    emit("delete", task);
    return;
  }
  confirmingDeleteId.value = task.task_id;
  window.setTimeout(() => {
    if (confirmingDeleteId.value === task.task_id) confirmingDeleteId.value = null;
  }, 2500);
}

// ---- tag derivation (from #hashtags in the title) ----
const HASHTAG_RE = /#([a-zA-Z0-9_-]+)/g;
function deriveTags(title: string): string[] {
  const out: string[] = [];
  const m = title.matchAll(HASHTAG_RE);
  for (const match of m) {
    if (!out.includes(match[1])) out.push(match[1]);
  }
  return out;
}
const isDone = (task: Task) => deriveTags(task.title).includes("done");
function displayTitle(task: Task): string {
  return task.title.replace(HASHTAG_RE, "").trim();
}

// ---- directory helpers ----
function dirLabel(cwd: string): string {
  const parts = cwd.split("/").filter(Boolean);
  return parts.length ? parts[parts.length - 1] : cwd;
}

// ---- filter pills ----
const allDirs = computed(() => {
  const seen = new Set<string>();
  const out: { path: string; kind: "git" | "cwd" }[] = [];
  for (const g of props.directories.git_roots) {
    if (!seen.has(g)) {
      seen.add(g);
      out.push({ path: g, kind: "git" });
    }
  }
  for (const c of props.directories.cwds) {
    if (!seen.has(c)) {
      seen.add(c);
      out.push({ path: c, kind: "cwd" });
    }
  }
  return out;
});

const filterPills = computed(() => {
  const pills: { key: string; label: string; count: number; favicon?: string }[] = [
    { key: "all", label: t("all"), count: props.tasks.length },
  ];
  // Only surface directories that actually have tasks.
  const dirsWithTasks = allDirs.value.filter((d) =>
    props.tasks.some((task) => task.cwd === d.path),
  );
  for (const d of dirsWithTasks) {
    pills.push({
      key: d.path,
      label: dirLabel(d.path),
      count: props.tasks.filter((task) => task.cwd === d.path).length,
      favicon: d.kind === "git" ? `/api/repo-favicon?root=${encodeURIComponent(d.path)}` : undefined,
    });
  }
  const noDirCount = props.tasks.filter((task) => !task.cwd).length;
  if (noDirCount > 0) {
    pills.push({
      key: "none",
      label: t("noDir"),
      count: noDirCount,
    });
  }
  return pills;
});

function onFaviconError(key: string) {
  faviconFailed.value = new Set(faviconFailed.value).add(key);
}

const filterLabel = computed(() => {
  if (filter.value === "none") return t("noDir");
  const d = allDirs.value.find((x) => x.path === filter.value);
  return d ? dirLabel(d.path) : filter.value;
});

const filteredTasks = computed(() => {
  if (filter.value === "all") return props.tasks;
  if (filter.value === "none") return props.tasks.filter((task) => !task.cwd);
  return props.tasks.filter((task) => task.cwd === filter.value);
});

// ---- grouping ----
const groups = computed(() => {
  const list = filteredTasks.value;
  const byKey = (key: "idle" | "handled" | "done") =>
    list.filter((task) => {
      if (isDone(task)) return key === "done";
      if (task.handled) return key === "handled";
      return key === "idle";
    });
  return [
    { key: "handled" as const, tasks: byKey("handled") },
    { key: "idle" as const, tasks: byKey("idle") },
    { key: "done" as const, tasks: byKey("done") },
  ];
});

function groupCopy(group: { key: "idle" | "handled" | "done"; tasks: Task[] }) {
  const n = group.tasks.length;
  if (group.key === "idle") return `${n} ${t("idleItems")}`;
  if (group.key === "handled") return `${n} ${t("currentlyHandled")}`;
  return `${n} ${t("tasksDone")}`;
}
function groupEmptyCopy(key: "idle" | "handled" | "done") {
  if (key === "idle") return t("nothingIdle");
  if (key === "handled") return t("nothingHandled");
  return t("nothingDone");
}
</script>
