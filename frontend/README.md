# Palladium OSS Frontend

Vue 3 + TypeScript + Vite frontend for Palladium OSS, backed by the real
API (see `"../docs/10-IMPLEMENTATION PLAN.md"` for the full stack/pattern
writeup, and `docs/07-UI-ARCHITECTURE.md`/`docs/11-COMPONENT-ARCHITECTURE.md`
for the design system and component conventions this frontend follows).

## Development

```
cp .env.example .env.local   # point VITE_API_URL at your local API, if not http://localhost:8080/api/v1
npm install
npm run dev      # start the dev server
npm run build    # type-check (vue-tsc) and production build
npm run preview  # preview a production build locally
npm run test     # run the Vitest suite
```

## Structure

```
src/
├── components/
│   ├── app/           Shell-level chrome: AppShell, UserMenu
│   ├── base/           Foundational UI primitives (BaseButton, BaseInput, BaseModal, ...)
│   ├── dashboard/       Dashboard-only widgets
│   ├── data-display/     DataTable, SimpleTable, FactGrid, RelationshipCard, TimelineEntries, ...
│   ├── dialogs/          One *FormDialog.vue per create/edit flow, plus the shared ConfirmationDialog
│   ├── navigation/        AppSidebar, TopNavigation, Breadcrumbs
│   └── workspace/          DetailWorkspace, WorkspaceHeader, WorkspaceActions
├── composables/        useAuth, useTheme, useSidebar, useDisclosure, use*Collection (per Collection Workspace)
├── router/             Route table (index.ts) and the shared navigation item list (navigation.ts)
├── services/
│   ├── <domain>/        One folder per domain, each holding a *Repository.ts (the only place apiFetch is called for that domain)
│   └── api/              httpClient.ts — the single apiFetch seam every repository builds on
├── styles/             Design tokens and base element styles
├── types/              One file per domain type
└── views/              *CollectionView.vue / *DetailView.vue per workspace, plus LoginView/DashboardView
```

Every `*Repository.ts`/`use*Collection.ts`/`dialogs/*FormDialog.vue` has a
co-located `*.test.ts` — see `services/customers/customerRepository.test.ts`,
`composables/useCustomerCollection.test.ts`, and
`components/dialogs/DeviceFormDialog.test.ts` as the reference pattern for
each of those three layers before adding a new one.
