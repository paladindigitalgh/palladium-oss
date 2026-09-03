import 'vue-router'

// `meta.title`/`meta.description`/`meta.navId` are already used by
// workspaceRoutes in index.ts without a declared type; `public` is the
// one new field the auth guard reads, declared here so it type-checks.
declare module 'vue-router' {
  interface RouteMeta {
    public?: boolean
    navId?: string
    title?: string
    description?: string
  }
}
