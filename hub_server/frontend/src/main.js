import { createApp } from 'vue'
import { createPinia } from 'pinia'
import PrimeVue from 'primevue/config'
import Aura from '@primevue/themes/aura'
import ToastService from 'primevue/toastservice'
import ConfirmationService from 'primevue/confirmationservice'
import 'primeicons/primeicons.css'

import App from './App.vue'
import router from './router'
import './style.css'

const app = createApp(App)

app.use(PrimeVue, {
    theme: {
        preset: Aura,
        options: {
            darkModeSelector: '.dark'
        }
    }
})

// Global Component Registration
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Dialog from 'primevue/dialog'
import Toast from 'primevue/toast'
import ConfirmDialog from 'primevue/confirmdialog'
import Tag from 'primevue/tag'
import Avatar from 'primevue/avatar'
import SelectButton from 'primevue/selectbutton'
import Skeleton from 'primevue/skeleton'
import Paginator from 'primevue/paginator'

app.component('Button', Button)
app.component('InputText', InputText)
app.component('Dialog', Dialog)
app.component('Toast', Toast)
app.component('ConfirmDialog', ConfirmDialog)
app.component('Tag', Tag)
app.component('Avatar', Avatar)
app.component('SelectButton', SelectButton)
app.component('Skeleton', Skeleton)
app.component('Paginator', Paginator)

import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import ToggleSwitch from 'primevue/toggleswitch'
import Tooltip from 'primevue/tooltip'

app.component('DataTable', DataTable)
app.component('Column', Column)
app.component('ToggleSwitch', ToggleSwitch)
app.directive('tooltip', Tooltip)

import Card from 'primevue/card'
import Password from 'primevue/password'
import FloatLabel from 'primevue/floatlabel'
import Message from 'primevue/message'
import Checkbox from 'primevue/checkbox'
import IconField from 'primevue/iconfield'
import InputIcon from 'primevue/inputicon'

app.component('Card', Card)
app.component('Password', Password)
app.component('FloatLabel', FloatLabel)
app.component('Message', Message)
app.component('Checkbox', Checkbox)
app.component('IconField', IconField)
app.component('InputIcon', InputIcon)

const pinia = createPinia()
app.use(pinia)
app.use(router)
app.use(ToastService)
app.use(ConfirmationService)

// Wait for router to be ready before mounting to prevent FOUC/Layout flash
router.isReady().then(() => {
    app.mount('#app')
})
