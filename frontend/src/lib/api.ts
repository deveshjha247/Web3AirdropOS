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
    api.post('/api/v1/auth/login', { email, password }),
  register: (email: string, password: string, name: string) =>
    api.post('/api/v1/auth/register', { email, password, name }),
  me: () => api.get('/api/v1/auth/me'),
  refresh: (refreshToken: string) =>
    api.post('/api/v1/auth/refresh', { refresh_token: refreshToken }),
}

// Wallets
export const walletsAPI = {
  list: () => api.get('/api/v1/wallets'),
  get: (id: string) => api.get(`/api/v1/wallets/${id}`),
  create: (data: any) => api.post('/api/v1/wallets', data),
  update: (id: string, data: any) => api.put(`/api/v1/wallets/${id}`, data),
  delete: (id: string) => api.delete(`/api/v1/wallets/${id}`),
  import: (data: any) => api.post('/api/v1/wallets/import', data),
  bulkCreate: (data: any) => api.post('/api/v1/wallets/bulk', data),
  getBalance: (id: string) => api.get(`/api/v1/wallets/${id}/balance`),
  getTransactions: (id: string) => api.get(`/api/v1/wallets/${id}/transactions`),
}

// Wallet Groups
export const walletGroupsAPI = {
  list: () => api.get('/api/v1/wallet-groups'),
  create: (data: any) => api.post('/api/v1/wallet-groups', data),
  update: (id: string, data: any) => api.put(`/api/v1/wallet-groups/${id}`, data),
  delete: (id: string) => api.delete(`/api/v1/wallet-groups/${id}`),
  addWallets: (id: string, walletIds: string[]) =>
    api.post(`/api/v1/wallet-groups/${id}/wallets`, { wallet_ids: walletIds }),
  removeWallets: (id: string, walletIds: string[]) =>
    api.delete(`/api/v1/wallet-groups/${id}/wallets`, { data: { wallet_ids: walletIds } }),
}

// Accounts
export const accountsAPI = {
  list: () => api.get('/api/v1/accounts'),
  get: (id: string) => api.get(`/api/v1/accounts/${id}`),
  create: (data: any) => api.post('/api/v1/accounts', data),
  update: (id: string, data: any) => api.put(`/api/v1/accounts/${id}`, data),
  delete: (id: string) => api.delete(`/api/v1/accounts/${id}`),
  getActivities: (id: string) => api.get(`/api/v1/accounts/${id}/activities`),
  linkWallet: (id: string, walletId: string) =>
    api.post(`/api/v1/accounts/${id}/link-wallet`, { wallet_id: walletId }),
  sync: (id: string) => api.post(`/api/v1/accounts/${id}/sync`),
}

// Campaigns
export const campaignsAPI = {
  list: () => api.get('/api/v1/campaigns'),
  get: (id: string) => api.get(`/api/v1/campaigns/${id}`),
  create: (data: any) => api.post('/api/v1/campaigns', data),
  update: (id: string, data: any) => api.put(`/api/v1/campaigns/${id}`, data),
  delete: (id: string) => api.delete(`/api/v1/campaigns/${id}`),
  getTasks: (id: string) => api.get(`/api/v1/campaigns/${id}/tasks`),
  addTask: (id: string, data: any) => api.post(`/api/v1/campaigns/${id}/tasks`, data),
  execute: (id: string, walletIds: string[]) =>
    api.post(`/api/v1/campaigns/${id}/execute`, { wallet_ids: walletIds }),
  getProgress: (id: string) => api.get(`/api/v1/campaigns/${id}/progress`),
}

// Tasks
export const tasksAPI = {
  get: (id: string) => api.get(`/api/v1/tasks/${id}`),
  update: (id: string, data: any) => api.put(`/api/v1/tasks/${id}`, data),
  execute: (id: string, walletId: string) =>
    api.post(`/api/v1/tasks/${id}/execute`, { wallet_id: walletId }),
  continue: (id: string, data?: any) =>
    api.post(`/api/v1/tasks/${id}/continue`, data),
  getExecutions: (id: string) => api.get(`/api/v1/tasks/${id}/executions`),
}

// Browser
export const browserAPI = {
  listProfiles: () => api.get('/api/v1/browser/profiles'),
  createProfile: (data: any) => api.post('/api/v1/browser/profiles', data),
  deleteProfile: (id: string) => api.delete(`/api/v1/browser/profiles/${id}`),
  listSessions: () => api.get('/api/v1/browser/sessions'),
  getSession: (id: string) => api.get(`/api/v1/browser/sessions/${id}`),
  startSession: (data: any) => api.post('/api/v1/browser/sessions', data),
  stopSession: (id: string) => api.delete(`/api/v1/browser/sessions/${id}`),
  executeAction: (sessionId: string, action: any) =>
    api.post(`/api/v1/browser/sessions/${sessionId}/action`, action),
  getScreenshot: (sessionId: string) =>
    api.get(`/api/v1/browser/sessions/${sessionId}/screenshot`),
  getTerminalOutput: (sessionId: string) =>
    api.get(`/api/v1/browser/sessions/${sessionId}/terminal`),
}

// Content / AI
export const contentAPI = {
  generate: (data: any) => api.post('/api/v1/content/generate', data),
  listDrafts: () => api.get('/api/v1/content/drafts'),
  getDraft: (id: string) => api.get(`/api/v1/content/drafts/${id}`),
  createDraft: (data: any) => api.post('/api/v1/content/drafts', data),
  updateDraft: (id: string, data: any) =>
    api.put(`/api/v1/content/drafts/${id}`, data),
  deleteDraft: (id: string) => api.delete(`/api/v1/content/drafts/${id}`),
  approveDraft: (id: string) => api.post(`/api/v1/content/drafts/${id}/approve`),
  rejectDraft: (id: string) => api.post(`/api/v1/content/drafts/${id}/reject`),
  schedule: (id: string, data: any) =>
    api.post(`/api/v1/content/drafts/${id}/schedule`, data),
  listScheduled: () => api.get('/api/v1/content/scheduled'),
}

// Automation Jobs
export const jobsAPI = {
  list: () => api.get('/api/v1/jobs'),
  get: (id: string) => api.get(`/api/v1/jobs/${id}`),
  create: (data: any) => api.post('/api/v1/jobs', data),
  update: (id: string, data: any) => api.put(`/api/v1/jobs/${id}`, data),
  delete: (id: string) => api.delete(`/api/v1/jobs/${id}`),
  start: (id: string) => api.post(`/api/v1/jobs/${id}/start`),
  stop: (id: string) => api.post(`/api/v1/jobs/${id}/stop`),
  getLogs: (id: string) => api.get(`/api/v1/jobs/${id}/logs`),
}

// Proxies
export const proxiesAPI = {
  list: () => api.get('/api/v1/proxies'),
  get: (id: string) => api.get(`/api/v1/proxies/${id}`),
  create: (data: any) => api.post('/api/v1/proxies', data),
  update: (id: string, data: any) => api.put(`/api/v1/proxies/${id}`, data),
  delete: (id: string) => api.delete(`/api/v1/proxies/${id}`),
  test: (id: string) => api.post(`/api/v1/proxies/${id}/test`),
  bulkCreate: (proxies: string[]) =>
    api.post('/api/v1/proxies/bulk', { proxies }),
}

// Dashboard
export const dashboardAPI = {
  getStats: () => api.get('/api/v1/dashboard/stats'),
  getActivity: () => api.get('/api/v1/dashboard/activity'),
  getCampaigns: () => api.get('/api/v1/dashboard/campaigns'),
  getNotifications: () => api.get('/api/v1/dashboard/notifications'),
}

// Notifications
export const notificationsAPI = {
  list: () => api.get('/api/v1/notifications'),
  markRead: (id: string) => api.post(`/api/v1/notifications/${id}/read`),
  markAllRead: () => api.post('/api/v1/notifications/read-all'),
}

// Settings
export const settingsAPI = {
  get: () => api.get('/api/v1/settings'),
  update: (data: any) => api.put('/api/v1/settings', data),
  updateApiKeys: (data: any) => api.put('/api/v1/settings/api-keys', data),
}

export default api
