import Vue from 'vue'
import Vuetify from 'vuetify/lib'
import { library } from '@fortawesome/fontawesome-svg-core'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { fas } from '@fortawesome/free-solid-svg-icons'

import 'vuetify/dist/vuetify.min.css'

Vue.component('font-awesome-icon', FontAwesomeIcon) // Register component globally
library.add(fas) // Include needed icons.

Vue.use(Vuetify);

export default new Vuetify({
  iconfont: 'md',
  options: {
    customProperties: true
  }
})
