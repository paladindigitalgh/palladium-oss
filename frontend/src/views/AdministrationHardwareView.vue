<script setup lang="ts">
import { onMounted, ref } from 'vue'
import WorkspaceHeader from '@/components/workspace/WorkspaceHeader.vue'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import BaseLoadingState from '@/components/base/BaseLoadingState.vue'
import SimpleTable, { type SimpleTableColumn } from '@/components/data-display/SimpleTable.vue'
import OLTModelFormDialog from '@/components/dialogs/OLTModelFormDialog.vue'
import DeviceManufacturerFormDialog from '@/components/dialogs/DeviceManufacturerFormDialog.vue'
import DeviceModelFormDialog from '@/components/dialogs/DeviceModelFormDialog.vue'
import { listOLTModels, deleteOLTModel } from '@/services/oltModels/oltModelRepository'
import { listDeviceManufacturers, deleteDeviceManufacturer } from '@/services/deviceManufacturers/deviceManufacturerRepository'
import { listDeviceModels, deleteDeviceModel } from '@/services/deviceModels/deviceModelRepository'
import { ApiError } from '@/services/api/httpClient'
import type { OLTModel } from '@/types/oltModel'
import type { DeviceManufacturer } from '@/types/deviceManufacturer'
import type { DeviceModel } from '@/types/deviceModel'

/**
 * The Hardware page, reached from the Administration landing hub
 * (AdministrationView.vue). Named generically rather than "OLT Models"
 * because OLT chassis types (internal/oltmodel) were only ever the first
 * physical-equipment catalog administered here -- Device Manufacturers
 * and Device Models (internal/devicemanufacturer, internal/devicemodel)
 * are the "e.g. ONU models" this page's own original doc comment
 * predicted, added 2026-09-08 when Device stopped carrying its
 * Manufacturer/Model as free text (see internal/inventory/model.go's
 * Device doc comment). Independent of Providers/Plans and Users, split
 * into its own route (/administration/hardware), the same "one page per
 * administered resource" pattern AdministrationProvidersView.vue and
 * AdministrationUsersView.vue already establish -- one page, three
 * independent catalog sections, since none of the three nests inside
 * another the way Provider/Plan does.
 *
 * None of the three catalogs below has an inline edit or Edit dialog,
 * matching AdministrationProvidersView.vue's own Provider section
 * (create-only): correcting a mistake means deleting and recreating the
 * entry, which each one's RESTRICT foreign key already blocks once a
 * real OLT/Device references it -- an operator who hits that conflict
 * is being told, correctly, that the entry is already in use.
 *
 * Device Models additionally need every DeviceManufacturer to resolve
 * each row's Manufacturer column and to populate
 * DeviceModelFormDialog.vue's picker -- fetched alongside Device Models
 * in loadDeviceModels rather than shared with the Device Manufacturers
 * section's own fetch, so each section reloads independently and a
 * failure in one never blocks the other two.
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

// --- Device Manufacturers ---

const manufacturers = ref<DeviceManufacturer[]>([])
const manufacturersLoading = ref(true)
const manufacturersLoadError = ref<string | null>(null)

async function loadManufacturers() {
  manufacturersLoading.value = true
  manufacturersLoadError.value = null
  try {
    manufacturers.value = await listDeviceManufacturers()
  } catch {
    manufacturersLoadError.value = 'Device manufacturers could not be loaded.'
  } finally {
    manufacturersLoading.value = false
  }
}

onMounted(loadManufacturers)

const manufacturerColumns: SimpleTableColumn[] = [
  { key: 'name', label: 'Name' },
  { key: 'description', label: 'Description' },
  { key: 'actions', label: '' },
]

const showManufacturerForm = ref(false)

function handleManufacturerCreated(manufacturer: DeviceManufacturer) {
  showManufacturerForm.value = false
  manufacturers.value = [...manufacturers.value, manufacturer]
}

const manufacturerDeleteError = ref<string | null>(null)
const manufacturerDeletePending = ref<string | null>(null)

async function handleManufacturerDelete(manufacturer: DeviceManufacturer) {
  manufacturerDeleteError.value = null
  manufacturerDeletePending.value = manufacturer.id
  try {
    await deleteDeviceManufacturer(manufacturer.id)
    manufacturers.value = manufacturers.value.filter((m) => m.id !== manufacturer.id)
  } catch (err) {
    manufacturerDeleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This device manufacturer is still in use by a device model -- it cannot be deleted.'
        : 'This device manufacturer could not be deleted.'
  } finally {
    manufacturerDeletePending.value = null
  }
}

// --- Device Models ---

const deviceModels = ref<DeviceModel[]>([])
const deviceModelManufacturers = ref<DeviceManufacturer[]>([])
const deviceModelsLoading = ref(true)
const deviceModelsLoadError = ref<string | null>(null)

async function loadDeviceModels() {
  deviceModelsLoading.value = true
  deviceModelsLoadError.value = null
  try {
    ;[deviceModels.value, deviceModelManufacturers.value] = await Promise.all([listDeviceModels(), listDeviceManufacturers()])
  } catch {
    deviceModelsLoadError.value = 'Device models could not be loaded.'
  } finally {
    deviceModelsLoading.value = false
  }
}

onMounted(loadDeviceModels)

function manufacturerName(manufacturerId: string): string {
  return deviceModelManufacturers.value.find((m) => m.id === manufacturerId)?.name ?? 'Unknown'
}

const deviceModelColumns: SimpleTableColumn[] = [
  { key: 'manufacturer', label: 'Manufacturer' },
  { key: 'name', label: 'Name' },
  { key: 'description', label: 'Description' },
  { key: 'actions', label: '' },
]

const showDeviceModelForm = ref(false)

function handleDeviceModelCreated(model: DeviceModel) {
  showDeviceModelForm.value = false
  deviceModels.value = [...deviceModels.value, model]
}

const deviceModelDeleteError = ref<string | null>(null)
const deviceModelDeletePending = ref<string | null>(null)

async function handleDeviceModelDelete(model: DeviceModel) {
  deviceModelDeleteError.value = null
  deviceModelDeletePending.value = model.id
  try {
    await deleteDeviceModel(model.id)
    deviceModels.value = deviceModels.value.filter((m) => m.id !== model.id)
  } catch (err) {
    deviceModelDeleteError.value =
      err instanceof ApiError && err.kind === 'conflict'
        ? 'This device model is still in use by a device -- it cannot be deleted.'
        : 'This device model could not be deleted.'
  } finally {
    deviceModelDeletePending.value = null
  }
}
</script>

<template>
  <div class="administration-hardware-view">
    <WorkspaceHeader title="Hardware" :breadcrumbs="[{ label: 'Administration', to: '/administration' }]" />

    <section class="hardware-section">
      <div class="hardware-section__header">
        <div>
          <h2 class="hardware-section__title">OLT Models</h2>
          <p class="page-description">
            A catalog of physical OLT chassis types and how many PON ports each one has. Creating an OLT against one
            of these auto-creates that many PON ports.
          </p>
        </div>
        <BaseButton variant="primary" size="sm" @click="showForm = true">New OLT Model</BaseButton>
      </div>

      <OLTModelFormDialog :open="showForm" @close="showForm = false" @created="handleCreated" />

      <p v-if="deleteError" class="hardware-error" role="alert">{{ deleteError }}</p>

      <BaseCard>
        <div v-if="loading" class="page-status">
          <BaseLoadingState :lines="3" />
        </div>

        <div v-else-if="loadError" class="page-status" role="alert">
          <p class="hardware-error">{{ loadError }}</p>
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
    </section>

    <section class="hardware-section">
      <div class="hardware-section__header">
        <div>
          <h2 class="hardware-section__title">Device Manufacturers</h2>
          <p class="page-description">
            A catalog of Device hardware manufacturers, e.g. "Nokia" or "Calix". Each Device Model below belongs to
            one of these.
          </p>
        </div>
        <BaseButton variant="primary" size="sm" @click="showManufacturerForm = true">New Device Manufacturer</BaseButton>
      </div>

      <DeviceManufacturerFormDialog :open="showManufacturerForm" @close="showManufacturerForm = false" @created="handleManufacturerCreated" />

      <p v-if="manufacturerDeleteError" class="hardware-error" role="alert">{{ manufacturerDeleteError }}</p>

      <BaseCard>
        <div v-if="manufacturersLoading" class="page-status">
          <BaseLoadingState :lines="3" />
        </div>

        <div v-else-if="manufacturersLoadError" class="page-status" role="alert">
          <p class="hardware-error">{{ manufacturersLoadError }}</p>
          <BaseButton variant="secondary" size="sm" @click="loadManufacturers">Retry</BaseButton>
        </div>

        <SimpleTable
          v-else
          :columns="manufacturerColumns"
          :rows="manufacturers"
          :row-key="(manufacturer) => manufacturer.id"
          empty-icon="settings"
          empty-title="No device manufacturers yet"
        >
          <template #cell-name="{ row }">{{ row.name }}</template>
          <template #cell-description="{ row }">{{ row.description || '—' }}</template>
          <template #cell-actions="{ row }">
            <BaseButton
              variant="secondary"
              size="sm"
              :disabled="manufacturerDeletePending === row.id"
              @click="handleManufacturerDelete(row)"
            >
              {{ manufacturerDeletePending === row.id ? 'Deleting…' : 'Delete' }}
            </BaseButton>
          </template>
        </SimpleTable>
      </BaseCard>
    </section>

    <section class="hardware-section">
      <div class="hardware-section__header">
        <div>
          <h2 class="hardware-section__title">Device Models</h2>
          <p class="page-description">
            A catalog of hardware models, each belonging to a Device Manufacturer above. A Device's Manufacturer and
            Model are picked from this catalog rather than typed freely.
          </p>
        </div>
        <BaseButton variant="primary" size="sm" @click="showDeviceModelForm = true">New Device Model</BaseButton>
      </div>

      <DeviceModelFormDialog :open="showDeviceModelForm" @close="showDeviceModelForm = false" @created="handleDeviceModelCreated" />

      <p v-if="deviceModelDeleteError" class="hardware-error" role="alert">{{ deviceModelDeleteError }}</p>

      <BaseCard>
        <div v-if="deviceModelsLoading" class="page-status">
          <BaseLoadingState :lines="3" />
        </div>

        <div v-else-if="deviceModelsLoadError" class="page-status" role="alert">
          <p class="hardware-error">{{ deviceModelsLoadError }}</p>
          <BaseButton variant="secondary" size="sm" @click="loadDeviceModels">Retry</BaseButton>
        </div>

        <SimpleTable
          v-else
          :columns="deviceModelColumns"
          :rows="deviceModels"
          :row-key="(model) => model.id"
          empty-icon="settings"
          empty-title="No device models yet"
        >
          <template #cell-manufacturer="{ row }">{{ manufacturerName(row.manufacturerId) }}</template>
          <template #cell-name="{ row }">{{ row.name }}</template>
          <template #cell-description="{ row }">{{ row.description || '—' }}</template>
          <template #cell-actions="{ row }">
            <BaseButton
              variant="secondary"
              size="sm"
              :disabled="deviceModelDeletePending === row.id"
              @click="handleDeviceModelDelete(row)"
            >
              {{ deviceModelDeletePending === row.id ? 'Deleting…' : 'Delete' }}
            </BaseButton>
          </template>
        </SimpleTable>
      </BaseCard>
    </section>
  </div>
</template>

<style scoped>
.administration-hardware-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-8);
}

.hardware-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.hardware-section__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-4);
}

.hardware-section__title {
  margin: 0 0 var(--space-1);
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-semibold);
}

.page-description {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.page-status {
  padding: var(--space-4) 0;
}

.hardware-error {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--color-error);
}
</style>
