import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'
import { initLocale } from './platform/index.js'

initLocale() // resolve the locale + set <html lang> before the first render
const app = mount(App, { target: document.getElementById('app') })

export default app
