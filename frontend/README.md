# Palladium OSS Frontend

Vue 3 + TypeScript + Vite frontend for Palladium OSS. This milestone
establishes the permanent application shell, routing, workspace layout
primitives, and base component library described in
`docs/07-UI-ARCHITECTURE.md` and `docs/11-COMPONENT-ARCHITECTURE.md`. It
intentionally contains no business functionality or API integration yet.

## Development

```
npm install
npm run dev      # start the dev server
npm run build    # type-check (vue-tsc) and production build
npm run preview  # preview a production build locally
```

## Structure

```
src/
├── components/
│   ├── app/         Shell-level chrome: AppShell, GlobalSearch, NotificationCenter, UserMenu
│   ├── base/         Foundational UI primitives (BaseButton, BaseCard, ...)
│   ├── navigation/   AppSidebar, TopNavigation, Breadcrumbs
│   └── workspace/     Reusable workspace layout primitives
├── composables/       useTheme, useSidebar, useDisclosure
├── router/            Route table and the shared navigation item list
├── styles/            Design tokens and base element styles
└── views/             Route-level components
```

See the codebase's conversation history / commit message for this
milestone's full architectural rationale.
