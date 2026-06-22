import { invoke } from './invoke.js'
import { confirm } from './confirm.svelte.js'
import { containersForImage, suggestTag } from './containers.svelte.js'
import { refreshImages } from './resources.svelte.js'

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
    const ok = await invoke('image.tag', { id: this.tagImage.id, ref: this.tagRef.trim() }, { success: `Tagged ${this.tagRef.trim()}` })
    this.tagging = false
    if (ok) {
      this.tagImage = null
      refreshImages()
    }
  }

  async untag(img, tag) {
    const last = img.tags.length <= 1
    const ok = await confirm({
      title: 'Remove this tag?',
      message: `“${tag}” will be removed.${last ? ' It is the only tag, so the image itself will be deleted.' : ' The image and its other tags stay.'}`,
      confirmLabel: 'Remove tag',
    })
    if (!ok) return
    await invoke('image.remove', { id: tag, force: false }, { success: `Removed ${tag}` })
    refreshImages()
  }

  async removeImage(img) {
    const inUse = img.containers > 0
    const many = img.tags.length > 1
    const ok = await confirm({
      title: 'Remove image?',
      message: `${many ? `All ${img.tags.length} tags of this image` : `“${img.tags[0]}”`} will be deleted.${inUse ? ` It is used by ${img.containers} container(s) and will be force-removed.` : ''}`,
      confirmLabel: 'Remove',
    })
    if (!ok) return
    await invoke('image.remove', { id: img.id, force: inUse }, { success: `Removed ${img.tags[0]}` })
    refreshImages()
  }
}
