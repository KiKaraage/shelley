<!-- Import models from an OpenAI-compatible /v1/models endpoint.
     Simple stacked dialog: endpoint URL + API key, then import.
     Reports counts after completion. -->
<template>
  <Modal :is-open="isOpen" :title="t('importModels')" @close="emit('close')">
    <div class="import-form">
      <div v-if="errMsg" class="models-error">
        {{ errMsg }}
        <button class="models-error-dismiss" @click="errMsg = null">×</button>
      </div>

      <div v-if="result" class="import-result">
        <p>
          {{ result.imported }} {{ t("importedModelsCount") }}
          <span v-if="result.skipped > 0"
            >{{ result.skipped }} {{ t("skippedModelsCount") }}</span
          >
        </p>
        <div class="import-result-buttons">
          <Button :label="t('done')" @click="emit('close')" />
        </div>
      </div>

      <template v-else>
        <div class="form-group">
          <label>{{ t("endpoint") }}</label>
          <InputText
            v-model="endpoint"
            placeholder="https://api.openai.com/v1"
            fluid
            :dt="inputFieldDt"
            autocomplete="off"
          />
        </div>

        <div class="form-group">
          <label>{{ t("apiKey") }}</label>
          <InputText
            v-model="apiKey"
            :placeholder="t('enterApiKey')"
            fluid
            :dt="inputFieldDt"
            autocomplete="off"
          />
        </div>

        <div class="import-buttons">
          <Button
            :label="imp ? t('importing') : t('importModels')"
            :disabled="imp || !endpoint || !apiKey"
            @click="handleImport"
          />
        </div>
      </template>
    </div>
  </Modal>
</template>

<script setup lang="ts">
import { ref } from "vue";
import Modal from "./Modal.vue";
import InputText from "primevue/inputtext";
import Button from "primevue/button";
import { inputFieldDt } from "./configFieldDt";
import { useI18n } from "../composables/i18n";
import { customModelsApi, type ImportModelsResponse } from "../../services/api";

defineProps<{ isOpen: boolean }>();
const emit = defineEmits<{
  (e: "close"): void;
  (e: "imported"): void;
}>();

const { t } = useI18n();

const endpoint = ref("");
const apiKey = ref("");
const imp = ref(false);
const errMsg = ref<string | null>(null);
const result = ref<ImportModelsResponse | null>(null);

async function handleImport() {
  try {
    imp.value = true;
    errMsg.value = null;
    result.value = await customModelsApi.importModels({
      provider_type: "openai",
      endpoint: endpoint.value.trim(),
      api_key: apiKey.value.trim(),
    });
    emit("imported");
  } catch (err) {
    errMsg.value = err instanceof Error ? err.message : "Failed to import models";
  } finally {
    imp.value = false;
  }
}
</script>
