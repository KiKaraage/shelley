<!-- Homepage shell: header + tabbed content area. Used as the landing
     page (Tasks / Timeline / Stats). Reuses the same header layout as
     ChatInterface so the shell stays consistent across routes. -->
<template>
  <div class="full-height flex flex-col">
    <!-- Header -->
    <div class="header">
      <div class="header-left">
        <Button
          class="btn-icon hide-on-desktop"
          text
          severity="secondary"
          :aria-label="t('openConversations')"
          v-tooltip.top="t('openConversations')"
          @click="emit('open-drawer')"
        >
          <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              :stroke-width="2"
              d="M4 6h16M4 12h16M4 18h16"
            />
          </svg>
        </Button>

        <Button
          v-if="isDrawerCollapsed"
          class="btn-icon show-on-desktop-only"
          text
          severity="secondary"
          :aria-label="t('expandSidebar')"
          v-tooltip.top="t('expandSidebar')"
          @click="emit('toggle-drawer-collapse')"
        >
          <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              :stroke-width="2"
              d="M13 5l7 7-7 7M5 5l7 7-7 7"
            />
          </svg>
        </Button>

        <h1 class="app-bar-title header-title">Shelley</h1>
      </div>

      <div class="header-actions">
        <!-- Split button: [+] New Conversation | [receipt] New Task -->
        <div class="split-btn">
          <button
            class="split-half"
            :aria-label="t('newConversation')"
            v-tooltip.top="t('newConversation')"
            @click="emit('new-conversation')"
          >
            <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" class="chat-icon-1rem">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                :stroke-width="2"
                d="M12 4v16m8-8H4"
              />
            </svg>
          </button>
          <span class="split-divider"></span>
          <button
            class="split-half"
            :aria-label="t('newTask')"
            v-tooltip.top="t('newTask')"
            @click="emit('open-create')"
          >
            <i class="pi pi-receipt" aria-hidden="true" />
          </button>
        </div>
      </div>
    </div>

    <!-- Tabs -->
    <div class="home-tabs">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        :class="['home-tab', { active: activeTab === tab.key }]"
        @click="activeTab = tab.key"
      >
        {{ tab.label }}
      </button>
    </div>

    <!-- Tab content -->
    <div class="home-content">
      <div v-if="activeTab === 'tasks'" class="home-tab-content home-tasks-content">
        <TaskList
          :tasks="tasks"
          :directories="directories"
          @open-create="emit('open-create')"
          @edit="(task) => emit('edit', task)"
          @start-thread="(task) => emit('start-thread', task)"
          @delete="(task) => emit('delete', task)"
          @open-thread="(task) => emit('open-thread', task)"
        />
      </div>
      <div v-else-if="activeTab === 'timeline'" class="home-tab-content">
        <p class="home-placeholder">Timeline coming soon.</p>
      </div>
      <div v-else-if="activeTab === 'stats'" class="home-tab-content">
        <p class="home-placeholder">Stats coming soon.</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import Button from "primevue/button";
import { useI18n } from "../composables/i18n";
import TaskList from "./TaskList.vue";
import type { Task, TaskDirectories } from "../../services/api_tasks";

const { t } = useI18n();

defineProps<{
  isDrawerCollapsed: boolean;
  tasks: Task[];
  directories: TaskDirectories;
}>();

const emit = defineEmits<{
  (e: "open-drawer"): void;
  (e: "toggle-drawer-collapse"): void;
  (e: "new-conversation"): void;
  (e: "open-create"): void;
  (e: "edit", task: Task): void;
  (e: "start-thread", task: Task): void;
  (e: "delete", task: Task): void;
  (e: "open-thread", task: Task): void;
}>();

type TabKey = "tasks" | "timeline" | "stats";

const tabs: { key: TabKey; label: string }[] = [
  { key: "tasks", label: "Tasks" },
  { key: "timeline", label: "Timeline" },
  { key: "stats", label: "Stats" },
];

const activeTab = ref<TabKey>("tasks");
</script>
