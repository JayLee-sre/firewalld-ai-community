import { createApp } from 'vue'
import { ElButton, ElForm, ElFormItem, ElIcon, ElInput } from 'element-plus'
import 'element-plus/dist/index.css'
import './styles/global.css'
import App from './App.vue'
import router from './router'

const app = createApp(App)

const components = [ElButton, ElForm, ElFormItem, ElIcon, ElInput]
for (const component of components) {
  app.use(component)
}
app.use(router)
app.mount('#app')
