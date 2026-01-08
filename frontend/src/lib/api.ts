import axios from 'axios'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'

const api = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Add auth token to requests
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Handle 401 errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

// Auth
export const authAPI = {
  login: (email: string, password: string) =>
    api.post('/api/auth/login', { email, password }),
  register: (email: string, password: string) =>
    api.post('/api/auth/register', { email, password }),
  me: () => api.get('/api/auth/me'),
}

// Wallets
export const walletsAPI = {
  list: () => api.get('/api/wallets'),
  get: (id: string) => api.get(`/api/wallets/${id}`),
  create: (data: any) => api.post('/api/wallets', data),
  update: (id: string, data: any) => api.put(`/api/wallets/${id}`, data),
  delete: (id: string) => api.delete(`/api/wallets/${id}`),
  import: (data: any) => api.post('/api/wallets/import', data),
  sync: (id: string) => api.post(`/api/wallets/${id}/sync`),
  groups: () => api.get('/api/wallets/groups'),
  tags: () => api.get('/api/wallets/tags'),
}

// Accounts
export const accountsAPI = {
  list: () => api.get('/api/accounts'),
  get: (id: string) => api.get(`/api/accounts/${id}`),
  create: (data: any) => api.post('/api/accounts', data),
  update: (id: string, data: any) => api.put(`/api/accounts/${id}`, data),
  delete: (id: string) => api.delete(`/api/accounts/${id}`),
  linkWallet: (id: string, walletId: string) =>
    api.post(`/api/accounts/${id}/link-wallet`, { wallet_id: walletId }),
}

// Campaigns
export const campaignsAPI = {
  list: () => api.get('/api/campaigns'),
  get: (id: string) => api.get(`/api/campaigns/${id}`),
  create: (data: any) => api.post('/api/campaigns', data),
  update: (id: string, data: any) => api.put(`/api/campaigns/${id}`, data),
  delete: (id: string) => api.delete(`/api/campaigns/${id}`),
  tasks: (id: string) => api.get(`/api/campaigns/${id}/tasks`),
  execute: (id: string, walletIds: string[]) =>
    api.post(`/api/campaigns/${id}/execute`, { wallet_ids: walletIds }),
  progress: (id: string) => api.get(`/api/campaigns/${id}/progress`),
}

// Tasks
export const tasksAPI = {
  execute: (id: string, walletId: string) =>
    api.post(`/api/tasks/${id}/execute`, { wallet_id: walletId }),
  continue: (executionId: string, data?: any) =>
    api.post(`/api/tasks/executions/${executionId}/continue`, data),
  executions: (id: string) => api.get(`/api/tasks/${id}/executions`),
}

// Browser
export const browserAPI = {
  profiles: () => api.get('/api/browser/profiles'),
  createProfile: (data: any) => api.post('/api/browser/profiles', data),
  deleteProfile: (id: string) => api.delete(`/api/browser/profiles/${id}`),
  sessions: () => api.get('/api/browser/sessions'),
  createSession: (profileId: string) =>
    api.post('/api/browser/sessions', { profile_id: profileId }),
  closeSession: (id: string) => api.delete(`/api/browser/sessions/${id}`),
  action: (sessionId: string, action: any) =>
    api.post(`/api/browser/sessions/${sessionId}/action`, action),
  screenshot: (sessionId: string) =>
    api.get(`/api/browser/sessions/${sessionId}/screenshot`),
}

// Content
export const contentAPI = {
  generate: (data: any) => api.post('/api/content/generate', data),
  drafts: () => api.get('/api/content/drafts'),
  createDraft: (data: any) => api.post('/api/content/drafts', data),
  updateDraft: (id: string, data: any) =>
    api.put(`/api/content/drafts/${id}`, data),
  deleteDraft: (id: string) => api.delete(`/api/content/drafts/${id}`),
  schedule: (id: string, data: any) =>
    api.post(`/api/content/drafts/${id}/schedule`, data),
  engagementPlan: (data: any) => api.post('/api/content/engagement-plan', data),
}

// Jobs
export const jobsAPI = {
  list: () => api.get('/api/jobs'),
  get: (id: string) => api.get(`/api/jobs/${id}`),
  create: (data: any) => api.post('/api/jobs', data),
  update: (id: string, data: any) => api.put(`/api/jobs/${id}`, data),
  delete: (id: string) => api.delete(`/api/jobs/${id}`),
  start: (id: string) => api.post(`/api/jobs/${id}/start`),
  stop: (id: string) => api.post(`/api/jobs/${id}/stop`),
  logs: (id: string) => api.get(`/api/jobs/${id}/logs`),
}

// Proxies
export const proxiesAPI = {
  list: () => api.get('/api/proxies'),
  create: (data: any) => api.post('/api/proxies', data),
  update: (id: string, data: any) => api.put(`/api/proxies/${id}`, data),
  delete: (id: string) => api.delete(`/api/proxies/${id}`),
  test: (id: string) => api.post(`/api/proxies/${id}/test`),
  bulkCreate: (proxies: string[]) =>
    api.post('/api/proxies/bulk', { proxies }),
}

// Dashboard
export const dashboardAPI = {
  stats: () => api.get('/api/dashboard/stats'),
  activity: () => api.get('/api/dashboard/activity'),
  campaigns: () => api.get('/api/dashboard/campaigns'),
}

export default api
