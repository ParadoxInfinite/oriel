import { invoke } from './invoke.js'
import { confirm } from './confirm.svelte.js'
import { containersForImage, suggestTag } from './containers.svelte.js'
import { refreshImages } from './resources.svelte.js'
import { t, tn } from './locale.svelte.js'

// Headless controller for the image tag / used-by / remove flows. Both editions
// instantiate it and bind their own markup, so the behaviour lives in one place.
export class ImageActions {
  tagImage = $state(null) // pinned image being tagged
  tagRef = $state('')
  tagging = $state(false)
  usedByImage = $state(null) // image whose "used by" list is open

  get usingContainers() {
    return this.usedByImage ? containersForImage(this.usedByImage.id) : []
  }

  openTag(img) {
    this.tagImage = img
    this.tagRef = suggestTag(img)
  }

  async applyTag() {
    if (!this.tagRef.trim() || this.tagging) return
    this.tagging = true
    const ok = await invoke('image.tag', { id: this.tagImage.id, ref: this.tagRef.trim() }, { success: t('imageActions.tagged', { ref: this.tagRef.trim() }) })
    this.tagging = false
    if (ok) {
      this.tagImage = null
      refreshImages()
    }
  }

  async untag(img, tag) {
    const last = img.tags.length <= 1
    const ok = await confirm({
      title: t('imageActions.untag.title'),
      message: t('imageActions.untag.msg', { tag }) + (last ? t('imageActions.untag.last') : t('imageActions.untag.kept')),
      confirmLabel: t('imageActions.untag.confirm'),
    })
    if (!ok) return
    await invoke('image.remove', { id: tag, force: false }, { success: t('imageActions.removed', { name: tag }) })
    refreshImages()
  }

  async removeImage(img) {
    const inUse = img.containers > 0
    const many = img.tags.length > 1
    const subject = many
      ? t('imageActions.removeImage.subjectMany', { count: img.tags.length })
      : t('imageActions.removeImage.subjectOne', { tag: img.tags[0] })
    const ok = await confirm({
      title: t('imageActions.removeImage.title'),
      message: t('imageActions.removeImage.deleted', { subject }) + (inUse ? tn('imageActions.removeImage.inUse', img.containers) : ''),
      confirmLabel: t('action.remove'),
    })
    if (!ok) return
    await invoke('image.remove', { id: img.id, force: inUse }, { success: t('imageActions.removed', { name: img.tags[0] }) })
    refreshImages()
  }
}
