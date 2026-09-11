import 'vue-router'

// `meta.title`/`meta.description`/`meta.navId` are already used by
// workspaceRoutes in index.ts without a declared type; `public` is the
// one new field the auth guard reads, declared here so it type-checks.
// `breadcrumb` is the static, per-route directory-trail fallback the
// router's beforeEach guard hands to useBreadcrumb.ts's setBreadcrumb on
// every navigation -- see that composable's own doc comment.
declare module 'vue-router' {
  interface RouteMeta {
    public?: boolean
    navId?: string
    title?: string
    description?: string
    breadcrumb?: { label: string; to?: RouteLocationRaw }[]
  }
}
