<script setup lang="ts">
const props = defineProps<{
  path: string
  bundled?: boolean
}>()

const modulePath = `go.loglayer.dev/${props.path}`
const goRefURL = `https://pkg.go.dev/${modulePath}`
const goRefBadge = `https://pkg.go.dev/badge/${modulePath}.svg`

const versionBadge = props.bundled
  ? `https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=v*&label=go.loglayer.dev`
  : `https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=${props.path}/v*&label=version`

const releasesURL = `https://github.com/loglayer/loglayer-go/releases`
const sourceURL = `https://github.com/loglayer/loglayer-go/tree/main/${props.path}`

const changelogURL = props.bundled
  ? `https://github.com/loglayer/loglayer-go/blob/main/CHANGELOG.md`
  : `https://github.com/loglayer/loglayer-go/blob/main/${props.path}/CHANGELOG.md`

const sourceBadge = `https://img.shields.io/badge/source-github-181717?logo=github`
const changelogBadge = `https://img.shields.io/badge/changelog-md-blue`
</script>

<template>
  <p
    style="display: flex; flex-wrap: wrap; gap: 0.4rem; align-items: center; margin-top: 0.5rem;"
  >
    <a :href="goRefURL" target="_blank" rel="noreferrer">
      <img :src="goRefBadge" alt="Go Reference" style="display: inline-block; vertical-align: middle;" />
    </a>
    <a :href="releasesURL" target="_blank" rel="noreferrer">
      <img :src="versionBadge" alt="Version" style="display: inline-block; vertical-align: middle;" />
    </a>
    <a :href="sourceURL" target="_blank" rel="noreferrer">
      <img :src="sourceBadge" alt="Source" style="display: inline-block; vertical-align: middle;" />
    </a>
    <a :href="changelogURL" target="_blank" rel="noreferrer">
      <img :src="changelogBadge" alt="Changelog" style="display: inline-block; vertical-align: middle;" />
    </a>
  </p>
</template>
