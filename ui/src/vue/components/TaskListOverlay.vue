<!-- Right-side overlay showing the TaskList without leaving the chat. Slides
     in from the right over a backdrop; closed by clicking the backdrop. -->
<template>
  <Teleport to="body">
    <Transition name="tasklist" :duration="400">
      <div v-if="isOpen" class="tasklist-overlay-root">
        <div class="tasklist-overlay-backdrop" @click="emit('close')" />
        <aside class="tasklist-overlay" role="dialog" aria-label="Tasks">
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
import TaskList from "./TaskList.vue";
import type { Task, TaskDirectories } from "../../services/api_tasks";

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
