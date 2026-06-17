<template>
  <div class="dashboard">
    <header class="header">
      <div class="header-left">
        <el-icon :size="32" color="#409eff"><Lightning /></el-icon>
        <h1>超级充电站智能监控系统</h1>
      </div>
      <div class="header-center">
        <div class="time-display">
          <el-icon color="#8cc5ff"><Clock /></el-icon>
          <span>{{ currentTime }}</span>
        </div>
        <div class="ws-status" :class="{ connected: store.wsConnected }">
          <span class="status-dot"></span>
          {{ store.wsConnected ? '实时连接中' : '连接断开' }}
        </div>
      </div>
      <div class="header-right">
        <el-button type="primary" @click="showPlugInDialog = true" :icon="Plus">
          模拟插桩
        </el-button>
        <el-button @click="refreshData" :icon="Refresh" :loading="loading">
          刷新
        </el-button>
      </div>
    </header>

    <main class="main-content">
      <aside class="left-panel">
        <StationOverview />
        <PowerUsageChart />
      </aside>

      <section class="center-panel">
        <ChargersGrid
          @plug-out="handlePlugOut"
          @view-detail="handleViewDetail"
        />
      </section>

      <aside class="right-panel">
        <PowerTrendChart />
        <AllocationLog />
      </aside>
    </main>

    <PlugInDialog
      v-model:visible="showPlugInDialog"
      @success="handlePlugInSuccess"
    />
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useStationStore } from './stores/station'
import { connectWebSocket, disconnectWebSocket } from './utils/websocket'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh } from '@element-plus/icons-vue'
import StationOverview from './components/StationOverview.vue'
import PowerUsageChart from './components/PowerUsageChart.vue'
import ChargersGrid from './components/ChargersGrid.vue'
import PowerTrendChart from './components/PowerTrendChart.vue'
import AllocationLog from './components/AllocationLog.vue'
import PlugInDialog from './components/PlugInDialog.vue'
import { stationApi } from './api/station'

const store = useStationStore()
const showPlugInDialog = ref(false)
const loading = ref(false)
const currentTime = ref('')
let timeTimer = null

const updateTime = () => {
  const now = new Date()
  const pad = (n) => n.toString().padStart(2, '0')
  currentTime.value = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`
}

const refreshData = async () => {
  loading.value = true
  try {
    await Promise.all([
      store.fetchChargers(),
      store.fetchStationStatus(),
      store.fetchPowerHistory({ hours: 6 }),
    ])
  } finally {
    loading.value = false
  }
}

const handlePlugInSuccess = () => {
  showPlugInDialog.value = false
  ElMessage.success('车辆插桩成功，系统已启动功率动态分配')
  refreshData()
}

const handlePlugOut = async (chargerId) => {
  try {
    await ElMessageBox.confirm(
      `确认要让 ${chargerId} 号充电桩结束充电？`,
      '拔桩确认',
      {
        confirmButtonText: '确认拔桩',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    const res = await stationApi.plugOut(chargerId)
    if (res.success) {
      ElMessage.success('车辆已安全拔桩')
      refreshData()
    }
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error('拔桩操作失败')
    }
  }
}

const handleViewDetail = (charger) => {
  console.log('View detail:', charger)
}

onMounted(() => {
  updateTime()
  timeTimer = setInterval(updateTime, 1000)
  refreshData()
  setTimeout(() => connectWebSocket(), 500)
})

onUnmounted(() => {
  if (timeTimer) clearInterval(timeTimer)
  disconnectWebSocket()
})
</script>

<style lang="scss" scoped>
.dashboard {
  width: 100vw;
  height: 100vh;
  display: flex;
  flex-direction: column;
  padding: 0;
  overflow: hidden;
}

.header {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  background: linear-gradient(90deg, rgba(19, 38, 66, 0.95) 0%, rgba(13, 33, 55, 0.98) 50%, rgba(19, 38, 66, 0.95) 100%);
  border-bottom: 1px solid rgba(64, 158, 255, 0.3);
  flex-shrink: 0;

  .header-left {
    display: flex;
    align-items: center;
    gap: 12px;

    h1 {
      font-size: 22px;
      font-weight: 700;
      background: linear-gradient(90deg, #8cc5ff 0%, #409eff 50%, #8cc5ff 100%);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
      letter-spacing: 2px;
    }
  }

  .header-center {
    display: flex;
    align-items: center;
    gap: 32px;

    .time-display {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 16px;
      color: #8cc5ff;
      font-family: 'Consolas', 'Monaco', monospace;
    }

    .ws-status {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 13px;
      color: #f56c6c;

      &.connected {
        color: #67c23a;
      }

      .status-dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        background: currentColor;
        animation: pulse 1.5s ease-in-out infinite;
      }
    }
  }

  .header-right {
    display: flex;
    gap: 12px;
  }
}

.main-content {
  flex: 1;
  display: grid;
  grid-template-columns: 360px 1fr 400px;
  gap: 16px;
  padding: 16px;
  min-height: 0;
  overflow: hidden;
}

.left-panel,
.right-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 0;
  overflow: hidden;
}

.center-panel {
  min-height: 0;
  overflow: hidden;
}
</style>
