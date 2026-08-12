<!-- TurnBand.vue — collapsible band for auto-expand mode.
     Shows a centered "Run for Xm YYs" toggle with dashed bottom border.
     No chevron, transparent background. -->
<template>
  <div class="turn-band" data-testid="turn-band">
    <button
      type="button"
      class="turn-toggle"
      :aria-expanded="expanded"
      @click="onToggle"
    >
      <span class="turn-toggle-label">Run for {{ duration }}</span>
    </button>
    <div v-if="expanded" class="turn-band-body">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  duration: string;
  expanded: boolean;
  onToggle: () => void;
}>();
</script>
