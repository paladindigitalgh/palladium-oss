<script setup lang="ts">
import { onMounted, ref } from 'vue'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import WorkspaceActions from '@/components/workspace/WorkspaceActions.vue'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import SimpleTable, { type SimpleTableColumn } from '@/components/data-display/SimpleTable.vue'
import OLTModelFormDialog from '@/components/dialogs/OLTModelFormDialog.vue'
import { listOLTModels, deleteOLTModel } from '@/services/oltModels/oltModelRepository'
import { ApiError } from '@/services/api/httpClient'
import type { OLTModel } from '@/types/oltModel'

/**
 * The Hardware page, reached from the Administration landing hub
 * (AdministrationView.vue). Named generically rather than "OLT Models"
 * because OLT chassis types (internal/oltmodel) are only the first
 * physical-equipment catalog administered here -- other hardware
 * configuration this page grows to hold later (e.g. ONU models) belongs
 * alongside it, not on a differently-named page. Today it manages exactly
 * one catalog: the OLT chassis type and how many PON ports it has --
 * creating an OLT against one of these auto-creates that many PON ports
 * (see internal/olt/service.OLTService's own doc comment). Independent
 * of Providers/Plans and Users, split into its own route
 * (/administration/hardware), the same "one page per administered
 * resource" pattern AdministrationProvidersView.vue and
 * AdministrationUsersView.vue already establish.
 *
 * There is no inline edit and no Edit dialog, matching
 * AdministrationProvidersView.vue's own Provider section (create-only):
 * correcting a mistake means deleting and recreating the entry, which
 * the RESTRICT foreign key on olts.olt_model_id already blocks once a
 * real OLT references it -- an operator who hits that conflict is being
 * told, correctly, that this model is already in use.
 */
const models = ref<OLTModel[]>([])
const loading = ref(true)
const loadError = ref<string | null>(null)

async function load() {
  loading.value = true
  loadError.value = null
  try {
    models.value = await listOLTModels()
  } catch {
    loadError.value = 'OLT models could not be loaded.'
  } finally {
    loading.value = false
  }
}

onMounted(load)

const columns: SimpleTableColumn[] = [
  { key: 'vendor', label: 'Vendor' },
  { key: 'name', label: 'Name' },
  { key: 'ponPortCount', label: 'PON Ports' },
  { key: 'description', label: 'Description' },
  { key: 'actions', label: '' },
]

const showForm = ref(false)

function handleCreated(model: OLTModel) {
  showForm.value = false
  models.value = [...models.value, model]
}

const deleteError = ref<string | null>(null)
const deletePending = ref<string | null>(null)

async function handleDelete(model: OLTModel) {
  deleteError.value = null
  deletePending.value = model.id
  try {
    await deleteOLTModel(model.id)
    models.value = models.value.filter((m) => m.id !== model.id)
  } catch (err) {
    deleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This OLT model is still in use by an OLT -- it cannot be deleted.'
        : 'This OLT model could not be deleted.'
  } finally {
    deletePending.value = null
  }
}
</script>

<template>
  <div class="administration-hardware-view">
    <WorkspaceHeader title="Hardware" :breadcrumbs="[{ label: 'Administration', to: '/administration' }]">
      <template #actions>
        <WorkspaceActions>
          <template #primary>
            <BaseButton variant="primary" size="sm" @click="showForm = true">New OLT Model</BaseButton>
          </template>
        </WorkspaceActions>
      </template>
    </WorkspaceHeader>

    <p class="page-description">
      A catalog of physical OLT chassis types and how many PON ports each one has. Creating an OLT against one of
      these auto-creates that many PON ports.
    </p>

    <OLTModelFormDialog :open="showForm" @close="showForm = false" @created="handleCreated" />

    <p v-if="deleteError" class="olt-models-error" role="alert">{{ deleteError }}</p>

    <BaseCard>
      <div v-if="loading" class="page-status">
        <BaseLoadingState :lines="3" />
      </div>

      <div v-else-if="loadError" class="page-status" role="alert">
        <p class="olt-models-error">{{ loadError }}</p>
        <BaseButton variant="secondary" size="sm" @click="load">Retry</BaseButton>
      </div>

      <SimpleTable
        v-else
        :columns="columns"
        :rows="models"
        :row-key="(model) => model.id"
        empty-icon="settings"
        empty-title="No OLT models yet"
      >
        <template #cell-vendor="{ row }">{{ row.vendor }}</template>
        <template #cell-name="{ row }">{{ row.name }}</template>
        <template #cell-ponPortCount="{ row }">{{ row.ponPortCount }}</template>
        <template #cell-description="{ row }">{{ row.description || '—' }}</template>
        <template #cell-actions="{ row }">
          <BaseButton variant="secondary" size="sm" :disabled="deletePending === row.id" @click="handleDelete(row)">
            {{ deletePending === row.id ? 'Deleting…' : 'Delete' }}
          </BaseButton>
        </template>
      </SimpleTable>
    </BaseCard>
  </div>
</template>

<style scoped>
.administration-hardware-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.page-description {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.page-status {
  padding: var(--space-4) 0;
}

.olt-models-error {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--color-error);
}
</style>
