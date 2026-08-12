<!--
  ChatOverflowMenu.vue — the top-right "kebab" overflow menu.

  This is the first piece of the UI rebuilt on real PrimeVue *components*
  (the rest of the Vue world so far only consumes the PrimeVue *theme*). It
  replaces a hand-rolled dropdown — a `v-if` panel with a manual document
  `mousedown` outside-click listener and bespoke segmented-toggle rows — with:

    - <Popover>     the dropdown surface (dismiss-on-outside-click + Esc + focus
                    trap come for free, so we delete the manual handlers)
    - <SelectButton> the theme / notifications / markdown segmented toggles
    - <Select>      the language picker

  The e2e DOM/ARIA contract is preserved so the shared Playwright specs keep
  passing in BOTH worlds:
    - root wrapper:  .chat-overflow-menu-wrapper
    - trigger:       button.btn-icon  (aria-label = t('moreOptions'))
    - action items:  button.overflow-menu-item  (matched by visible text)
  See e2e/agents-md-vim.spec.ts and e2e/diff-viewer-find.spec.ts.

  State the menu reads/writes lives in shared composables/services
  (markdownMode, theme, notifications), so this component owns it directly
  instead of taking a dozen props. Conversation-scoped actions (diffs, git
  graph, archive, export, …) are surfaced as events for ChatInterface to wire
  to its existing handlers.
-->
<template>
  <div class="chat-overflow-menu-wrapper">
    <Button
      class="btn-icon"
      text
      severity="secondary"
      :aria-label="t('moreOptions')"
      v-tooltip.top="t('moreOptions')"
      aria-haspopup="true"
      :aria-expanded="open"
      @click="toggle"
    >
      <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          :stroke-width="2"
          d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z"
        />
      </svg>
      <span v-if="hasUpdate" class="version-update-dot" />
    </Button>

    <Popover
      ref="popoverRef"
      :pt="{ root: { class: 'chat-overflow-popover' }, content: { class: 'overflow-menu-panel' } }"
      @show="open = true"
      @hide="open = false"
    >
      <!-- Custom server-provided links (icon is a raw SVG path) -->
      <button
        v-for="(link, index) in links"
        :key="index"
        class="overflow-menu-item"
        @click="onExternalLink(link.url)"
      >
        <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" class="chat-menu-icon">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            :stroke-width="2"
            :d="
              link.icon_svg ||
              'M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14'
            "
          />
        </svg>
        {{ link.title }}
      </button>

      <template v-if="canArchive">
        <div class="overflow-menu-divider" />
        <button class="overflow-menu-item" @click="onArchive">
          <i class="pi pi-inbox chat-menu-icon" aria-hidden="true" />
          {{ t("archiveConversation") }}
          <span class="overflow-menu-shortcut"
            ><kbd>{{ menuShortcutLabel("archive") }}</kbd></span
          >
        </button>
      </template>

      <template v-if="canExport">
        <div class="overflow-menu-divider" />
        <button class="overflow-menu-item" @click="onExport">
          <i class="pi pi-download chat-menu-icon" aria-hidden="true" />
          {{ t("exportConversation") }}
          <span class="overflow-menu-shortcut"
            ><kbd>{{ menuShortcutLabel("export") }}</kbd></span
          >
        </button>
      </template>

      <template v-if="canExport">
        <button
          class="overflow-menu-item"
          v-tooltip.left="gistId ? t('updateGist') : t('exportToGist')"
          @click="onExportGist"
        >
          <svg fill="none" stroke="currentColor" viewBox="0 0 24 24" class="chat-menu-icon">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
          </svg>
          {{ gistId ? t("updateGist") : t("exportToGist") }}
        </button>
      </template>

      <div class="overflow-menu-divider" />
      <button class="overflow-menu-item" @click="onEditAgentsMd">
        <i class="pi pi-pencil chat-menu-icon" aria-hidden="true" />
        {{ t("editUserAgentsMd") }}
        <span class="overflow-menu-shortcut"
          ><kbd>{{ menuShortcutLabel("editAgentsMd") }}</kbd></span
        >
      </button>
      <button class="overflow-menu-item" @click="onEditFile">
        <i class="pi pi-file-edit chat-menu-icon" aria-hidden="true" />
        {{ t("editFile") }}
        <span
          v-tooltip.bottom="editFileShortcutTooltip"
          class="overflow-menu-shortcut"
          :class="{ 'overflow-menu-shortcut-inert': isFirefox }"
          ><kbd>{{ menuShortcutLabel("editFile") }}</kbd></span
        >
      </button>

      <div class="overflow-menu-divider" />
      <button class="overflow-menu-item" @click="onCheckVersion">
        <i class="pi pi-refresh chat-menu-icon" aria-hidden="true" />
        {{ t("checkForNewVersion") }}
        <span v-if="hasUpdate" class="version-menu-dot" />
        <span class="overflow-menu-shortcut"
          ><kbd>{{ menuShortcutLabel("checkVersion") }}</kbd></span
        >
      </button>

      <!-- Compact view/theme/notification controls -->
      <div class="overflow-menu-divider" />
      <div class="overflow-quick-controls">
        <button
          type="button"
          class="overflow-quick-control"
          data-testid="conversation-view-toggle"
          :aria-label="conversationViewLabel"
          :aria-pressed="conversationViewMode === 'end-of-turn'"
          :title="conversationViewLabel"
          @click="toggleConversationView"
        >
          <span class="overflow-quick-label">{{ t("brevity") }}</span>
          <span class="overflow-choice-stage" aria-hidden="true">
            <Transition name="choice-rotate" mode="out-in">
              <svg
                :key="conversationViewMode"
                class="overflow-choice-current"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
              >
                <template v-if="conversationViewMode === 'all'">
                  <path d="M8 6h12M8 12h12M8 18h12" stroke-width="2" stroke-linecap="round" />
                  <circle cx="4" cy="6" r="1.4" fill="currentColor" stroke="none" />
                  <circle cx="4" cy="12" r="1.4" fill="currentColor" stroke="none" />
                  <circle cx="4" cy="18" r="1.4" fill="currentColor" stroke="none" />
                </template>
                <template v-else-if="conversationViewMode === 'end-of-turn'">
                  <path d="M8 7h12M8 17h12" stroke-width="2" stroke-linecap="round" />
                  <circle cx="4" cy="7" r="1.4" fill="currentColor" stroke="none" />
                  <path
                    d="m2.7 17 1.1 1.1 2.3-2.5"
                    stroke-width="1.8"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  />
                </template>
                <template v-else>
                  <path d="M8 6h12M8 12h12M8 18h12" stroke-width="2" stroke-linecap="round" />
                  <circle cx="4" cy="6" r="1.4" fill="currentColor" stroke="none" />
                  <circle cx="4" cy="12" r="1.4" fill="currentColor" stroke="none" />
                  <circle cx="4" cy="18" r="1.4" fill="currentColor" stroke="none" />
                  <path d="M18 10l-2 2 2 2" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
                </template>
              </svg>
            </Transition>
          </span>
          <span class="overflow-choice-alternatives" aria-hidden="true">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
              <template v-if="conversationViewMode === 'all'">
                <path d="M8 7h12M8 17h12" stroke-width="2" stroke-linecap="round" />
                <circle cx="4" cy="7" r="1.4" fill="currentColor" stroke="none" />
                <path
                  d="m2.7 17 1.1 1.1 2.3-2.5"
                  stroke-width="1.8"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </template>
              <template v-else-if="conversationViewMode === 'end-of-turn'">
                <path d="M8 6h12M8 12h12M8 18h12" stroke-width="2" stroke-linecap="round" />
                <circle cx="4" cy="6" r="1.4" fill="currentColor" stroke="none" />
                <circle cx="4" cy="12" r="1.4" fill="currentColor" stroke="none" />
                <circle cx="4" cy="18" r="1.4" fill="currentColor" stroke="none" />
                <path d="M18 10l-2 2 2 2" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
              </template>
              <template v-else>
                <path d="M8 6h12M8 12h12M8 18h12" stroke-width="2" stroke-linecap="round" />
                <circle cx="4" cy="6" r="1.4" fill="currentColor" stroke="none" />
                <circle cx="4" cy="12" r="1.4" fill="currentColor" stroke="none" />
                <circle cx="4" cy="18" r="1.4" fill="currentColor" stroke="none" />
              </template>
            </svg>
          </span>
          <span class="sr-only-label">{{ conversationViewLabel }}</span>
        </button>

        <button
          type="button"
          class="overflow-quick-control"
          data-testid="theme-cycle"
          :aria-label="themeLabel"
          :title="themeLabel"
          @click="cycleTheme"
        >
          <span class="overflow-quick-label">{{ t("look") }}</span>
          <span class="overflow-choice-stage" aria-hidden="true">
            <Transition name="choice-rotate" mode="out-in">
              <i :key="theme" :class="['pi', themeIcon, 'overflow-choice-current']" />
            </Transition>
          </span>
          <span class="overflow-choice-alternatives" aria-hidden="true">
            <i v-for="choice in otherThemes" :key="choice" :class="['pi', themeIconFor(choice)]" />
          </span>
          <span class="sr-only-label">{{ themeLabel }}</span>
        </button>

        <button
          v-if="notificationSupported"
          type="button"
          class="overflow-quick-control"
          data-testid="notification-toggle"
          :disabled="notifBlocked && !notifEnabled"
          :aria-label="notificationLabel"
          :aria-pressed="notifEnabled"
          :title="notificationLabel"
          @click="toggleNotifications"
        >
          <span class="overflow-quick-label">{{ t("notifications") }}</span>
          <span class="overflow-choice-stage" aria-hidden="true">
            <Transition name="choice-rotate" mode="out-in">
              <i
                :key="String(notifEnabled)"
                :class="[
                  'pi',
                  notifEnabled ? 'pi-bell' : 'pi-bell-slash',
                  'overflow-choice-current',
                ]"
              />
            </Transition>
          </span>
          <span class="overflow-choice-alternatives" aria-hidden="true">
            <i :class="['pi', notifEnabled ? 'pi-bell-slash' : 'pi-bell']" />
          </span>
          <span class="sr-only-label">{{ notificationLabel }}</span>
        </button>
      </div>

      <!-- Language -->
      <div class="overflow-menu-divider" />
      <div class="overflow-menu-control">
        <div class="md-toggle-label">{{ t("language") }}</div>
        <Select
          v-model="lang"
          :options="languageOptions"
          option-label="label"
          option-value="locale"
          :aria-label="t('switchLanguage')"
          class="overflow-language-select"
          append-to="self"
          @update:model-value="onLangChange"
        >
          <template #value="{ value }">
            <span class="language-dropdown-flag">{{ languageFor(value).flag }}</span>
            <span>{{ languageFor(value).label }}</span>
          </template>
          <template #option="{ option }">
            <span class="language-dropdown-flag">{{ option.flag }}</span>
            <span>{{ option.label }}</span>
          </template>
        </Select>
      </div>

      <!-- Git Attribution -->
      <div class="overflow-menu-divider" />
      <div class="overflow-menu-control">
        <Select
          v-model="gitAttribution"
          :options="gitAttributionOptions"
          option-label="label"
          option-value="value"
          aria-label="Git Attribution"
          class="overflow-language-select"
          append-to="self"
          @update:model-value="onGitAttributionChange"
        />
      </div>
    </Popover>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import Popover from "primevue/popover";
import Button from "primevue/button";
import Select from "primevue/select";
import type { Link } from "../../types";
import type { Locale } from "../../i18n/types";
import { useI18n } from "../composables/i18n";
import { useConversationView, type ConversationViewMode } from "../composables/conversationView";
import { menuShortcutLabel, isFirefox } from "../../utils/menuShortcuts";
import { type ThemeMode, getStoredTheme, setStoredTheme, applyTheme } from "../../services/theme";
import {
  isChannelEnabled,
  setChannelEnabled,
  getBrowserNotificationState,
  requestBrowserNotificationPermission,
} from "../../services/notifications";
import { api } from "../../services/api";

defineProps<{
  links: Link[];
  canArchive: boolean;
  canExport: boolean;
  hasUpdate: boolean;
  gistId: string | null;
  gistUrl: string | null;
}>();

const emit = defineEmits<{
  (e: "open-external-link", url: string): void;
  (e: "archive"): void;
  (e: "export"): void;
  (e: "export-gist"): void;
  (e: "edit-agents-md"): void;
  (e: "edit-file"): void;
  (e: "check-version"): void;
}>();

const { t, locale, setLocale } = useI18n();
const { conversationViewMode, setConversationViewMode } = useConversationView();
const conversationViewLabel = computed(() => {
  const mode = conversationViewMode.value;
  const current =
    mode === "all" ? t("seeAllMessages") : mode === "end-of-turn" ? t("seeEndOfTurnMessagesOnly") : t("seeAutoExpand");
  const next =
    mode === "all" ? t("seeEndOfTurnMessagesOnly") : mode === "end-of-turn" ? t("seeAutoExpand") : t("seeAllMessages");
  return `${current} → ${next}`;
});
function toggleConversationView() {
  const mode = conversationViewMode.value;
  const next: ConversationViewMode = mode === "all" ? "end-of-turn" : mode === "end-of-turn" ? "auto-expand" : "all";
  setConversationViewMode(next);
}

// Edit File uses Cmd/Ctrl+Shift+P (VS Code parity). Firefox reserves that combo
// for "New Private Window" and never delivers it to the page, so the shortcut
// is inert there; explain that on hover rather than silently misleading users.
const editFileShortcutTooltip = computed(() =>
  isFirefox ? t("editFileShortcutFirefox") : t("editFileShortcut"),
);

const popoverRef = ref<InstanceType<typeof Popover> | null>(null);
const open = ref(false);

function toggle(event: MouseEvent) {
  popoverRef.value?.toggle(event);
}
function hide() {
  popoverRef.value?.hide();
}

// Each action emits its event, then closes the Popover. Kept as explicit
// one-liners (rather than a union-typed helper) so defineEmits' per-event
// overloads type-check cleanly.
const onArchive = () => (emit("archive"), hide());
const onExport = () => (emit("export"), hide());
const onExportGist = () => (emit("export-gist"), hide());
const onEditAgentsMd = () => (emit("edit-agents-md"), hide());
const onEditFile = () => (emit("edit-file"), hide());
const onCheckVersion = () => (emit("check-version"), hide());
function onExternalLink(url: string) {
  emit("open-external-link", url);
  hide();
}

const notificationSupported = typeof Notification !== "undefined";

// ---- Theme cycle (System → Light → Dark) ----
const theme = ref<ThemeMode>(getStoredTheme());
const themeOrder: ThemeMode[] = ["system", "light", "dark"];
const nextTheme = computed(
  () => themeOrder[(themeOrder.indexOf(theme.value) + 1) % themeOrder.length],
);
function themeIconFor(mode: ThemeMode): string {
  if (mode === "light") return "pi-sun";
  if (mode === "dark") return "pi-moon";
  return "pi-desktop";
}
const themeIcon = computed(() => themeIconFor(theme.value));
const otherThemes = computed(() => themeOrder.filter((choice) => choice !== theme.value));
const themeLabel = computed(() => `${t(theme.value)} → ${t(nextTheme.value)}`);
function cycleTheme() {
  theme.value = nextTheme.value;
  setStoredTheme(theme.value);
  applyTheme(theme.value);
}

// ---- Browser notifications (on / off) ----
const notifEnabled = ref<boolean>(isChannelEnabled("browser"));
const notifBlocked = ref(getBrowserNotificationState() === "denied");
const notificationLabel = computed(() => {
  if (notifBlocked.value && !notifEnabled.value) return t("blockedByBrowser");
  return notifEnabled.value ? t("disableNotifications") : t("enableNotifications");
});
async function toggleNotifications() {
  if (notifEnabled.value) {
    setChannelEnabled("browser", false);
    notifEnabled.value = false;
    return;
  }
  notifEnabled.value = await requestBrowserNotificationPermission();
  notifBlocked.value = getBrowserNotificationState() === "denied";
}

// ---- Language picker ----
interface LanguageOption {
  locale: Locale;
  flag: string;
  label: string;
}
const languageOptions: LanguageOption[] = [
  { locale: "en", flag: "\uD83C\uDDFA\uD83C\uDDF8", label: "English" },
  { locale: "ja", flag: "\uD83C\uDDEF\uD83C\uDDF5", label: "\u65E5\u672C\u8A9E" },
  { locale: "fr", flag: "\uD83C\uDDEB\uD83C\uDDF7", label: "Fran\u00E7ais" },
  {
    locale: "ru",
    flag: "\uD83C\uDDF7\uD83C\uDDFA",
    label: "\u0420\u0443\u0441\u0441\u043A\u0438\u0439",
  },
  { locale: "es", flag: "\uD83C\uDDEA\uD83C\uDDF8", label: "Espa\u00F1ol" },
  { locale: "zh-CN", flag: "\uD83C\uDDE8\uD83C\uDDF3", label: "\u7B80\u4F53\u4E2D\u6587" },
  { locale: "zh-TW", flag: "\uD83C\uDDF9\uD83C\uDDFC", label: "\u7E41\u9AD4\u4E2D\u6587" },
  { locale: "vi", flag: "\uD83C\uDDFB\uD83C\uDDF3", label: "Ti\u1EBFng Vi\u1EC7t" },
  { locale: "upgoer5", flag: "\uD83D\uDE80", label: "Up-Goer Five" },
];
const lang = ref<Locale>(locale.value);
function languageFor(l: Locale): LanguageOption {
  return languageOptions.find((o) => o.locale === l) || languageOptions[0];
}
function onLangChange(l: Locale) {
  lang.value = l;
  setLocale(l);
}

// ---- Git Attribution picker ----
interface GitAttributionOption {
  value: string;
  label: string;
}
const gitAttributionOptions: GitAttributionOption[] = [
  { value: "co-author", label: t("coAuthoredByShelley") },
  { value: "assisted-by", label: t("assistedByModelInShelley") },
  { value: "off", label: t("noAgentAttribution") },
];
const gitAttribution = ref<string>("co-author");

// Load current setting on mount
api.getSettings().then((settings) => {
  if (settings["shelley.attribution"]) {
    gitAttribution.value = settings["shelley.attribution"];
  }
}).catch(() => {
  // Leave at default (co-author)
});

function onGitAttributionChange(value: string) {
  gitAttribution.value = value;
  api.setSetting("shelley.attribution", value).catch(() => {});
}
</script>
