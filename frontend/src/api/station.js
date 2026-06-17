import axios from 'axios'

const request = axios.create({
  baseURL: '/api',
  timeout: 10000,
})

request.interceptors.response.use(
  (response) => response.data,
  (error) => {
    console.error('API Error:', error)
    return Promise.reject(error)
  }
)

export const stationApi = {
  plugIn: (data) => request.post('/station/plug-in', data),
  plugOut: (data) => request.post('/station/plug-out', data),
  getChargers: () => request.get('/station/chargers'),
  getStatus: () => request.get('/station/status'),
  getPowerHistory: (params) => request.get('/station/power-history', { params }),
  updateSOC: (data) => request.post('/station/update-soc', data),
  triggerAllocation: () => request.post('/station/allocate'),
  setGridLimit: (data) => request.post('/station/grid-limit', data),
}

export default request
