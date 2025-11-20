import { createApp } from 'vue'

// Vuetify
import 'vuetify/styles'
import '@mdi/font/css/materialdesignicons.css'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'

// Components
import App from './App.vue'

const myTheme = {
  dark: false,
  colors: {
    'background': '#004D40', // Cor de fundo principal - teal-darken-4
  }
}

const vuetify = createVuetify({
  components,
  directives,
  theme: {
    defaultTheme: 'myTheme',
    themes: {
      myTheme,
    }
  },
  icons: { 
    defaultSet: 'mdi', 
  },
})

createApp(App).use(vuetify).mount('#app')
