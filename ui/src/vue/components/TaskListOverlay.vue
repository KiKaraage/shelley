<!-- Right-side overlay showing the TaskList without leaving the chat. Slides
     in from the right over a backdrop; closed by clicking the backdrop. -->
<template>
  <Teleport to="body">
    <Transition name="tasklist" :duration="300">
      <div v-if="isOpen" class="tasklist-overlay-root">
        <div class="tasklist-overlay-backdrop" @click="emit('close')" />
        <aside class="tasklist-overlay" role="dialog" aria-label="Tasks">
          <div class="tasklist-overlay-header">
            <h2 class="app-bar-title tasklist-overlay-title">Tasks</h2>
            <Button
              class="btn-icon"
              text
              severity="secondary"
              :aria-label="t('closeTaskList')"
              v-tooltip.top="t('closeTaskList')"
              @click="emit('close')"
            >
              <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  :stroke-width="2"
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </Button>
            <Button
              class="btn-icon"
              text
              severity="secondary"
              :aria-label="t('newTask')"
              v-tooltip.top="t('newTask')"
              @click="emit('open-create')"
            >
              <i class="pi pi-pen-to-square chat-icon-1rem" aria-hidden="true" />
            </Button>
          </div>
          <div class="tasklist-overlay-body">
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
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import Button from "primevue/button";
import { useI18n } from "../composables/i18n";
import TaskList from "./TaskList.vue";
import type { Task, TaskDirectories } from "../../services/api_tasks";

const { t } = useI18n();

defineProps<{
  isOpen: boolean;
  tasks: Task[];
  directories: TaskDirectories;
}>();

const emit = defineEmits<{
  (e: "close"): void;
  (e: "open-create"): void;
  (e: "edit", task: Task): void;
  (e: "start-thread", task: Task): void;
  (e: "delete", task: Task): void;
  (e: "open-thread", task: Task): void;
}>();
</script>
