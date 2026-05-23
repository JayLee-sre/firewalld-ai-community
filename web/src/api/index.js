import axios from 'axios'
import { ElMessage } from 'element-plus'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
})

let redirectingToLogin = false
export const TOKEN_KEY = 'zhiyu_waf_token'

export function getAuthToken() {
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function setAuthToken(token) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearAuthToken() {
  localStorage.removeItem(TOKEN_KEY)
}

api.interceptors.request.use(config => {
  const token = getAuthToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  res => res.data,
  err => {
    if (err.response?.status === 401) {
      clearAuthToken()
      if (!redirectingToLogin && window.location.pathname !== '/login') {
        redirectingToLogin = true
        window.location.replace('/login')
      }
      return Promise.reject(err)
    }

    if (err.response?.status === 403 && (err.response?.data?.code === 'professional_required' || err.response?.data?.code === 'feature_not_licensed')) {
      return Promise.reject(err)
    }

    if (!err.config?.suppressError) {
      const msg = err.response?.data?.error || err.message || '请求失败'
      ElMessage.error(msg)
    }
    return Promise.reject(err)
  }
)

export default api
