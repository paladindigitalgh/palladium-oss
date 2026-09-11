<script setup lang="ts">
import { computed, ref } from 'vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseEmptyState from '@/components/base/BaseEmptyState.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import { formatDisplayDateTime } from '@/lib/dates'
import { formatDisplayName } from '@/lib/users'
import type { Note } from '@/types/note'

/**
 * Body content meant to sit inside a `<SectionCard title="Notes">` --
 * the same "SectionCard owns the card chrome and collapse behavior,
 * this component owns only what's inside" split TimelineEntries.vue
 * documents for itself. Unlike TimelineEntries, this is not read-only:
 * an operator can add a note (docs/03-DOMAIN-MODEL.md's Note domain is
 * client-writable, unlike Event), so this component also owns the add
 * form and its own submit-in-progress/error display.
 *
 * `notes` is the full, already-fetched array for one entity (see
 * noteRepository.ts's listNotes -- no server-side pagination, the same
 * "fetch once, slice client-side" shape every other list in this
 * codebase uses); pagination here is purely a local presentation
 * concern, not something a parent needs to know about or control, so
 * `page` is never exposed as a prop.
 */
const props = withDefaults(
  defineProps<{
    notes?: Note[]
    loading?: boolean
    submitting?: boolean
    error?: string | null
  }>(),
  { notes: () => [], loading: false, submitting: false, error: null },
)

const emit = defineEmits<{
  (event: 'submit', body: string): void
}>()

const PAGE_SIZE = 5

const draft = ref('')
const page = ref(1)

const totalPages = computed(() => Math.max(1, Math.ceil(props.notes.length / PAGE_SIZE)))

const pageItems = computed(() => {
  const start = (page.value - 1) * PAGE_SIZE
  return props.notes.slice(start, start + PAGE_SIZE)
})

function handleSubmit() {
  const body = draft.value.trim()
  if (!body) return

  // Optimistic: the point of submitting is to see the new note, and it
  // will land on page 1 (notes are always newest first -- see
  // noteRepository.ts's listNotes doc comment), so jump there now rather
  // than waiting on a round trip before the page the operator is looking
  // at actually changes.
  page.value = 1
  draft.value = ''
  emit('submit', body)
}
</script>

<template>
  <div class="notes-section">
    <form class="notes-section__form" @submit.prevent="handleSubmit">
      <textarea
        v-model="draft"
        class="notes-section__textarea"
        placeholder="Add a note…"
        rows="3"
        :disabled="submitting"
      />
      <p v-if="error" class="notes-section__error" role="alert">{{ error }}</p>
      <div class="notes-section__form-actions">
        <BaseButton type="submit" variant="primary" size="sm" :disabled="submitting || !draft.trim()">
          {{ submitting ? 'Adding…' : 'Add Note' }}
        </BaseButton>
      </div>
    </form>

    <BaseLoadingState v-if="loading" :lines="3" />
    <BaseEmptyState
      v-else-if="notes.length === 0"
      icon="notes"
      title="No notes yet"
      description="Add the first note above to start a history for this record."
    />
    <template v-else>
      <ol class="notes-section__list">
        <li v-for="n in pageItems" :key="n.id" class="notes-section__note">
          <div class="notes-section__note-header">
            <span class="notes-section__author">{{
              formatDisplayName({ firstName: n.authorFirstName, lastName: n.authorLastName, email: n.authorEmail })
            }}</span>
            <span class="notes-section__timestamp">{{ formatDisplayDateTime(n.createdAt) }}</span>
          </div>
          <p class="notes-section__body">{{ n.body }}</p>
        </li>
      </ol>

      <div v-if="totalPages > 1" class="notes-section__pagination">
        <BaseButton variant="ghost" size="sm" :disabled="page <= 1" @click="page--">Previous</BaseButton>
        <span class="notes-section__page-indicator">Page {{ page }} of {{ totalPages }}</span>
        <BaseButton variant="ghost" size="sm" :disabled="page >= totalPages" @click="page++">Next</BaseButton>
      </div>
    </template>
  </div>
</template>

<style scoped>
.notes-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.notes-section__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.notes-section__textarea {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background-color: var(--color-surface);
  color: var(--color-text-primary);
  font: inherit;
  font-size: var(--font-size-sm);
  resize: vertical;
}

.notes-section__textarea:hover {
  border-color: var(--color-text-muted);
}

.notes-section__textarea:focus-visible {
  outline: 2px solid var(--color-brand);
  outline-offset: 2px;
}

.notes-section__textarea:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.notes-section__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.notes-section__form-actions {
  display: flex;
  justify-content: flex-end;
}

.notes-section__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.notes-section__note {
  padding-top: var(--space-3);
  border-top: 1px solid var(--color-border);
}

.notes-section__note:first-child {
  padding-top: 0;
  border-top: none;
}

.notes-section__note-header {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: var(--space-2);
  margin-bottom: var(--space-1);
}

.notes-section__author {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-primary);
}

.notes-section__timestamp {
  font-size: var(--font-size-xs);
  color: var(--color-text-muted);
}

.notes-section__body {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
  white-space: pre-wrap;
  overflow-wrap: break-word;
}

.notes-section__pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-3);
}

.notes-section__page-indicator {
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}
</style>
