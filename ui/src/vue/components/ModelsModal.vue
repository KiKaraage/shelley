<!-- Manage Models: lists built-in + custom models and owns duplicate/delete +
     refresh + import. The list is a compact PrimeVue DataTable (size="small" +
     modelsTableDt tokens) grouped by source via rowGroupMode="subheader": in
     the default single-gateway install this collapses what used to be two
     full columns of the same repeated hostname/URL into one quiet group
     header line. Endpoints render per-row only when they differ from the
     group's common endpoint. Only custom rows carry a `model` and thus the
     edit/duplicate/delete/toggle actions. Adding/editing opens ModelFormModal;
     importing opens ImportModelsModal — separate stacked dialogs layered on
     top of this one.

     A sticky checkpoint bar appears when custom models are selected via
     checkboxes, offering bulk disable/enable and delete. Toggle and delete
     operate optimistically — local state updates instantly without a full
     list reload, and server errors roll back only the affected rows. -->
<template>
  <Modal
    :is-open="isOpen"
    :title="t('manageModels')"
    class-name="modal-xwide"
    @close="emit('close')"
  >
    <template #title-right>
      <div class="models-header-actions">
        <Button
          :label="refreshing ? t('refreshingModels') : t('refreshModels')"
          severity="secondary"
          size="small"
          :disabled="refreshing || loading"
          @click="handleRefreshModels"
        />
        <Button severity="secondary" size="small" @click="importOpen = true">
          {{ t("importModels") }}
        </Button>
        <Button size="small" @click="handleAddNew">+ {{ t("addModel") }}</Button>
      </div>
    </template>

    <div class="models-modal" :class="{ 'models-modal-list': showList }">
      <div v-if="error" class="models-error">
        {{ error }}
        <button class="models-error-dismiss" @click="error = null">×</button>
      </div>

      <div v-if="loading" class="models-loading">
        <div class="spinner"></div>
        <span>{{ t("loadingModels") }}</span>
      </div>

      <!-- Empty state -->
      <div v-else-if="builtInModels.length === 0 && models.length === 0" class="models-empty">
        <p>{{ t("noModelsConfigured") }}</p>
        <p class="models-empty-hint">{{ t("noModelsHint") }}</p>
      </div>

      <!-- Model List -->
      <DataTable
        v-else
        :value="tableRows"
        data-key="key"
        size="small"
        scrollable
        scroll-height="flex"
        :dt="modelsTableDt"
        class="models-datatable"
        row-group-mode="subheader"
        group-rows-by="groupKey"
        :pt="{ rowGroupHeaderCell: { colspan: 7 } }"
      >
        <template #groupheader="{ data }">
          <span class="models-group-name">{{ data.groupLabel }}</span>
          <span class="models-group-count">({{ groupCounts[data.groupKey] }})</span>
          <span v-if="data.groupEndpoint" class="models-group-endpoint">{{
            data.groupEndpoint
          }}</span>
        </template>
        <Column
          v-if="customModels.length > 0"
          selection-mode="multiple"
          header-style="width: 3rem"
          body-style="width: 3rem; text-align: center"
          :pt="{ headerCell: { style: 'padding:0' }, bodyCell: { style: 'padding:0' } }"
        >
          <template #header>
            <Checkbox
              :model-value="allCustomSelected"
              :indeterminate="someCustomSelected && !allCustomSelected"
              :binary="true"
              @update:model-value="toggleSelectAll"
            />
          </template>
          <template #body="{ data }">
            <Checkbox
              v-if="data.model"
              :model-value="selectedKeys.has(data.key)"
              :binary="true"
              @update:model-value="toggleSelect(data.key)"
            />
          </template>
        </Column>
        <Column :header="t('columnName')" field="name">
          <template #body="{ data }">
            <span class="models-cell-name">{{ data.name }}</span>
            <span v-for="tag in data.tags" :key="tag" class="models-cell-tag">{{ tag }}</span>
          </template>
        </Column>
        <Column :header="t('columnModelId')" field="modelId">
          <template #body="{ data }">
            <span class="models-cell-mono">{{ data.modelId }}</span>
            <div v-if="data.endpoint" class="models-cell-endpoint" :title="data.endpoint">
              {{ data.endpoint }}
            </div>
          </template>
        </Column>
        <Column :header="t('columnProvider')" field="apiShape">
          <template #body="{ data }">
            <span :class="{ 'models-cell-muted': !data.apiShape }">{{ data.apiShape || "—" }}</span>
          </template>
        </Column>
        <Column :header="t('columnImages')" field="supportsImages" class="models-col-images">
          <template #body="{ data }">
            <span
              :class="data.supportsImages ? 'models-table-image-yes' : 'models-table-image-no'"
              role="img"
              :title="data.imageTitle"
              :aria-label="data.imageTitle"
              >{{ data.supportsImages ? "✓" : "✕"
              }}<span v-if="data.imageAuto" class="models-table-image-auto-tag">{{
                t("imageSupportAutoShort")
              }}</span></span
            >
          </template>
        </Column>
        <Column class="models-col-enabled">
          <template #header>
            <span class="sr-only">{{ t("enabled") }}</span>
          </template>
          <template #body="{ data }">
            <ToggleSwitch
              v-if="data.model"
              :model-value="data.model.enabled"
              :disabled="data.model._pending"
              @update:model-value="(v: boolean) => handleToggleEnabled(data.model!, v)"
            />
            <span v-else class="models-cell-muted">—</span>
          </template>
        </Column>
        <Column class="models-col-actions">
          <template #header>
            <span class="sr-only">{{ t("columnActions") }}</span>
          </template>
          <template #body="{ data }">
            <div v-if="data.model" class="models-cell-actions">
              <Button
                class="btn-icon"
                text
                severity="secondary"
                v-tooltip.top="t('duplicate')"
                :aria-label="t('duplicate')"
                :disabled="data.model._pending"
                @click="handleDuplicate(data.model)"
              >
                <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" width="16" height="16">
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    :stroke-width="2"
                    d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
                  />
                </svg>
              </Button>
              <Button
                class="btn-icon"
                text
                severity="secondary"
                v-tooltip.top="t('editModel')"
                :aria-label="t('editModel')"
                :disabled="data.model._pending"
                @click="handleEdit(data.model)"
              >
                <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" width="16" height="16">
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    :stroke-width="2"
                    d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                  />
                </svg>
              </Button>
              <Button
                class="btn-icon btn-danger"
                text
                severity="danger"
                v-tooltip.top="t('delete_')"
                :aria-label="t('delete_')"
                :disabled="data.model._pending"
                @click="handleDelete(data.model.model_id)"
              >
                <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" width="16" height="16">
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    :stroke-width="2"
                    d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                  />
                </svg>
              </Button>
            </div>
          </template>
        </Column>
      </DataTable>

      <!-- Checkpoint bar: sticky toolbar when custom models are selected -->
      <div v-if="selectedKeys.size > 0" class="models-checkpoint">
        <span class="models-checkpoint-count">
          {{ selectedKeys.size }} {{ selectedKeys.size === 1 ? "model" : "models" }} selected
        </span>
        <div class="models-checkpoint-actions">
          <Button
            severity="secondary"
            size="small"
            :disabled="bulkPending"
            @click="bulkToggleEnabled(false)"
          >
            {{ t("disable") }}
          </Button>
          <Button
            severity="secondary"
            size="small"
            :disabled="bulkPending"
            @click="bulkToggleEnabled(true)"
          >
            {{ t("enable") }}
          </Button>
          <Button
            severity="danger"
            size="small"
            :disabled="bulkPending"
            @click="bulkDelete"
          >
            {{ t("delete_") }}
          </Button>
        </div>
      </div>
    </div>
  </Modal>

  <!-- Stacked add/edit dialog, layered on top of the list. -->
  <ModelFormModal
    :is-open="formOpen"
    :edit-model="editModel"
    @saved="handleFormSaved"
    @close="formOpen = false"
  />

  <!-- Stacked import dialog. -->
  <ImportModelsModal
    :is-open="importOpen"
    @imported="handleImported"
    @close="importOpen = false"
  />
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import DataTable from "primevue/datatable";
import Column from "primevue/column";
import Button from "primevue/button";
import Checkbox from "primevue/checkbox";
import ToggleSwitch from "primevue/toggleswitch";
import Modal from "./Modal.vue";
import ModelFormModal from "./ModelFormModal.vue";
import ImportModelsModal from "./ImportModelsModal.vue";
import { modelsTableDt } from "./modelsTableDt";
import { prettyModelLabels } from "../../utils/modelNames";
import { API_TYPE_LABELS, PROVIDER_LABELS } from "./customModelConstants";
import { useI18n } from "../composables/i18n";
import { api, customModelsApi, type AvailableModel, type CustomModel } from "../../services/api";

const props = defineProps<{ isOpen: boolean }>();
const emit = defineEmits<{ (e: "close"): void; (e: "modelsChanged"): void }>();

const { t } = useI18n();

// Extend CustomModel with a local pending flag for optimistic UI.
interface TrackedModel extends CustomModel {
  _pending?: boolean;
}

const models = ref<TrackedModel[]>([]);
const loading = ref(true);
const refreshing = ref(false);
const error = ref<string | null>(null);
const builtInModels = ref<AvailableModel[]>([]);

// Checkpoint: selection state.
const selectedKeys = reactive(new Set<string>());
const bulkPending = ref(false);

// Stacked add/edit dialog state.
const formOpen = ref(false);
const editModel = ref<TrackedModel | null>(null);

// Stacked import dialog state.
const importOpen = ref(false);

const builtInModelsFiltered = computed(() =>
  builtInModels.value.filter((m) => m.id !== "predictable"),
);

const showList = computed(
  () => !loading.value && (builtInModels.value.length > 0 || models.value.length > 0),
);

// Custom-model-only subset for the select-all checkbox.
const customModels = computed(() => models.value);
const allCustomSelected = computed(
  () => customModels.value.length > 0 && customModels.value.every((m) => selectedKeys.has(`custom:${m.model_id}`)),
);
const someCustomSelected = computed(
  () => customModels.value.some((m) => selectedKeys.has(`custom:${m.model_id}`)),
);

function toggleSelectAll(v: boolean | Event) {
  const on = v === true;
  for (const m of customModels.value) {
    const key = `custom:${m.model_id}`;
    if (on) selectedKeys.add(key);
    else selectedKeys.delete(key);
  }
}

function toggleSelect(key: string) {
  if (selectedKeys.has(key)) selectedKeys.delete(key);
  else selectedKeys.add(key);
}

interface TableRow {
  key: string;
  groupKey: string;
  groupLabel: string;
  groupEndpoint: string;
  name: string;
  modelId: string;
  apiShape: string | null;
  endpoint: string;
  tags: string[];
  supportsImages: boolean;
  imageTitle: string;
  imageAuto: boolean;
  model: TrackedModel | null;
}

const tableRows = computed<TableRow[]>(() => {
  const labels = prettyModelLabels(builtInModelsFiltered.value);
  const groups = new Map<string, AvailableModel[]>();
  for (const m of builtInModelsFiltered.value) {
    const src = m.source || "";
    if (!groups.has(src)) groups.set(src, []);
    groups.get(src)!.push(m);
  }
  const rows: TableRow[] = [];
  for (const [src, group] of groups) {
    const endpoints = new Set(group.map((m) => m.base_url || ""));
    const common = endpoints.size === 1 ? (group[0].base_url ?? "") : "";
    for (const m of group) {
      rows.push({
        key: `builtin:${m.id}`,
        groupKey: `builtin:${src}`,
        groupLabel: src,
        groupEndpoint: common,
        name: labels.get(m.id) || m.id,
        modelId: m.id,
        apiShape: (m.api_type && API_TYPE_LABELS[m.api_type]) || null,
        endpoint: common ? "" : m.base_url || "",
        tags: [],
        supportsImages: m.supports_images ?? true,
        imageTitle: (m.supports_images ?? true) ? t("imageSupportYes") : t("imageSupportNo"),
        imageAuto: false,
        model: null,
      });
    }
  }
  for (const m of models.value) {
    rows.push({
      key: `custom:${m.model_id}`,
      groupKey: "custom",
      groupLabel: t("customModelsGroup"),
      groupEndpoint: "",
      name: m.display_name,
      modelId: m.model_name,
      apiShape: PROVIDER_LABELS[m.provider_type],
      endpoint: m.endpoint,
      tags: (m.tags || "")
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean),
      supportsImages: customModelSupportsImages(m),
      imageTitle: customModelImageTitle(m),
      imageAuto: (m.image_support ?? "auto") === "auto",
      model: m,
    });
  }
  return rows;
});

const groupCounts = computed<Record<string, number>>(() => {
  const counts: Record<string, number> = {};
  for (const row of tableRows.value) {
    counts[row.groupKey] = (counts[row.groupKey] || 0) + 1;
  }
  return counts;
});

function customModelSupportsImages(model: CustomModel): boolean {
  const setting = model.image_support ?? "auto";
  if (setting === "yes") return true;
  if (setting === "no") return false;
  return model.supports_images ?? true;
}

function customModelImageTitle(model: CustomModel): string {
  const label = customModelSupportsImages(model) ? t("imageSupportYes") : t("imageSupportNo");
  if ((model.image_support ?? "auto") === "auto") {
    return `${t("imageSupportAuto")} \u2014 ${label}`;
  }
  return label;
}

async function loadModels() {
  try {
    loading.value = true;
    error.value = null;
    models.value = await customModelsApi.getCustomModels();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Failed to load models";
  } finally {
    loading.value = false;
  }
}

function setBuiltInFromModelList(modelList: AvailableModel[]) {
  builtInModels.value = modelList.filter((m) => m.source && m.source !== "custom");
}

function handleAddNew() {
  editModel.value = null;
  formOpen.value = true;
}

function handleEdit(model: TrackedModel) {
  editModel.value = model;
  formOpen.value = true;
}

async function handleFormSaved() {
  await loadModels();
  emit("modelsChanged");
}

async function handleDuplicate(model: TrackedModel) {
  try {
    error.value = null;
    await customModelsApi.duplicateCustomModel(model.model_id);
    await loadModels();
    emit("modelsChanged");
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Failed to duplicate model";
  }
}

/** Optimistic delete: remove from list immediately, roll back on error. */
async function handleDelete(modelId: string) {
  const idx = models.value.findIndex((m) => m.model_id === modelId);
  if (idx === -1) return;
  const removed = models.value.splice(idx, 1)[0];
  selectedKeys.delete(`custom:${modelId}`);
  try {
    error.value = null;
    await customModelsApi.deleteCustomModel(modelId);
    emit("modelsChanged");
  } catch (err) {
    models.value.splice(idx, 0, removed);
    error.value = err instanceof Error ? err.message : "Failed to delete model";
  }
}

async function handleRefreshModels() {
  try {
    refreshing.value = true;
    error.value = null;
    const refreshedModels = await api.refreshModels();
    if (window.__SHELLEY_INIT__) {
      window.__SHELLEY_INIT__.models = refreshedModels;
    }
    setBuiltInFromModelList(refreshedModels);
    emit("modelsChanged");
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Failed to refresh models";
  } finally {
    refreshing.value = false;
  }
}

/** Optimistic toggle: flip enabled in place, roll back on error. */
async function handleToggleEnabled(model: TrackedModel, enabled: boolean) {
  const previous = model.enabled;
  model._pending = true;
  model.enabled = enabled;
  try {
    error.value = null;
    await customModelsApi.updateCustomModel(model.model_id, { enabled });
    emit("modelsChanged");
  } catch (err) {
    model.enabled = previous;
    error.value = err instanceof Error ? err.message : "Failed to update model";
  } finally {
    model._pending = false;
  }
}

async function handleImported() {
  await loadModels();
  emit("modelsChanged");
}

// ---- Bulk actions (checkpoint bar) ----

/** Bulk toggle enabled: optimistic for all selected, roll back any failures. */
async function bulkToggleEnabled(enabled: boolean) {
  bulkPending.value = true;
  const ids = models.value
    .filter((m) => selectedKeys.has(`custom:${m.model_id}`))
    .map((m) => m.model_id);
  const previous = new Map<string, boolean>();
  for (const m of models.value) {
    if (ids.includes(m.model_id)) previous.set(m.model_id, m.enabled);
  }
  for (const m of models.value) {
    if (ids.includes(m.model_id)) {
      m._pending = true;
      m.enabled = enabled;
    }
  }
  try {
    error.value = null;
    await Promise.all(ids.map((id) => customModelsApi.updateCustomModel(id, { enabled })))
    emit("modelsChanged");
  } catch (err) {
    for (const m of models.value) {
      const prev = previous.get(m.model_id);
      if (prev !== undefined) m.enabled = prev;
    }
    error.value = err instanceof Error ? err.message : "Failed to update models";
  } finally {
    for (const m of models.value) m._pending = false;
    bulkPending.value = false;
  }
}

/** Bulk delete: optimistic, remove all selected, roll back on error. */
async function bulkDelete() {
  bulkPending.value = true;
  const ids = models.value
    .filter((m) => selectedKeys.has(`custom:${m.model_id}`))
    .map((m) => m.model_id);
  const removed = new Map<string, { model: TrackedModel; index: number }>();
  for (const id of ids) {
    const idx = models.value.findIndex((m) => m.model_id === id);
    if (idx !== -1) removed.set(id, { model: models.value[idx], index: idx });
  }
  // Remove in reverse index order so splice indices stay valid.
  const sorted = [...removed.values()].sort((a, b) => b.index - a.index);
  for (const { index } of sorted) models.value.splice(index, 1);
  for (const id of ids) selectedKeys.delete(`custom:${id}`);
  try {
    error.value = null;
    await Promise.all(ids.map((id) => customModelsApi.deleteCustomModel(id)))
    emit("modelsChanged");
  } catch (err) {
    const restored = [...removed.values()].sort((a, b) => a.index - b.index);
    for (const { model, index } of restored) models.value.splice(index, 0, model);
    error.value = err instanceof Error ? err.message : "Failed to delete models";
  } finally {
    bulkPending.value = false;
  }
}

watch(
  () => props.isOpen,
  (open) => {
    if (open) {
      selectedKeys.clear();
      loadModels();
      const initData = window.__SHELLEY_INIT__;
      if (initData?.models) {
        setBuiltInFromModelList(initData.models);
      }
    }
  },
  { immediate: true },
);
</script>
