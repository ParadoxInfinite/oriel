import { apiGet } from './api.js'

// Popular public registries. `search`/`tags` flag which expose an API we proxy;
// the rest pull fine, we just guide the ref format. Shared by both editions'
// pull dialogs so the source list and behaviour stay identical.
export const REGISTRY_SOURCES = [
  { id: 'dockerhub', label: 'Docker Hub', host: '', search: true, tags: true, hint: 'e.g. nginx:alpine' },
  { id: 'quay', label: 'Quay.io', host: 'quay.io/', search: true, tags: true, hint: 'e.g. quay.io/prometheus/prometheus' },
  { id: 'ecr', label: 'AWS · public.ecr.aws', host: 'public.ecr.aws/', search: true, tags: false, hint: 'e.g. public.ecr.aws/nginx/nginx' },
  { id: 'ghcr', label: 'GitHub · ghcr.io', host: 'ghcr.io/', search: false, tags: false, hint: 'e.g. ghcr.io/cli/cli:latest' },
  { id: 'gcr', label: 'Google · gcr.io', host: 'gcr.io/', search: false, tags: false, hint: 'e.g. gcr.io/distroless/static' },
  { id: 'k8s', label: 'Kubernetes · registry.k8s.io', host: 'registry.k8s.io/', search: false, tags: false, hint: 'e.g. registry.k8s.io/pause:3.9' },
  { id: 'mcr', label: 'Microsoft · mcr.microsoft.com', host: 'mcr.microsoft.com/', search: false, tags: false, hint: 'e.g. mcr.microsoft.com/dotnet/runtime' },
]

export const REGISTRY_HOSTS = REGISTRY_SOURCES.map((s) => s.host).filter(Boolean)

export const searchRegistry = (source, term, limit = 20) =>
  apiGet(`/api/images/search?source=${source}&q=${encodeURIComponent(term)}&limit=${limit}`)

export const listImageTags = (source, repo, limit = 30) =>
  apiGet(`/api/images/tags?source=${source}&repo=${encodeURIComponent(repo)}&limit=${limit}`)
