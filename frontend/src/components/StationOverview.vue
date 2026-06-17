<template>
  <div class="overview card">
    <div class="card-header">
      <el-icon><DataAnalysis /></el-icon>
      <span>电站总览</span>
    </div>
    <div class="card-body">
      <div class="total-power">
        <div class="power-label">当前总功率</div>
        <div class="power-value">
          <span class="num">{{ currentPower.toFixed(1) }}</span>
          <span class="unit">kW</span>
          <span class="divider">/</span>
          <span class="total">{{ totalMaxPower.toFixed(0) }} kW</span>
        </div>
        <el-progress
          :percentage="powerPercent"
          :stroke-width="12"
          :show-text="false"
          :color="progressColor"
        />
        <div class="power-info">
          <span>使用率</span>
          <span :style="{ color: progressColor }">{{ powerPercent.toFixed(1) }}%</span>
        </div>
      </div>

      <div class="stats-grid">
        <div class="stat-item charging">
          <el-icon :size="24"><Charge /></el-icon>
          <div class="stat-info">
            <div class="stat-num">{{ status.activeChargers }}</div>
            <div class="stat-label">充电中</div>
          </div>
        </div>
        <div class="stat-item idle">
          <el-icon :size="24"><Power /></el-icon>
          <div class="stat-info">
            <div class="stat-num">{{ status.idleChargers }}</div>
            <div class="stat-label">空闲</div>
          </div>
        </div>
        <div class="stat-item fault">
          <el-icon :size="24"><Warning /></el-icon>
          <div class="stat-info">
            <div class="stat-num">{{ status.faultChargers }}</div>
            <div class="stat-label">故障</div>
          </div>
        </div>
        <div class="stat-item total">
          <el-icon :size="24"><Van /></el-icon>
          <div class="stat-info">
            <div class="stat-num">{{ status.totalChargingVehicles }}</div>
            <div class="stat-label">服务车辆</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useStationStore } from '../stores/station'
import { DataAnalysis, Charge, Power, Warning, Van } from '@element-plus/icons-vue'

const store = useStationStore()

const status = computed(() => store.stationStatus)
const currentPower = computed(() => store.stationStatus.currentTotalPower)
const totalMaxPower = computed(() => store.stationStatus.totalMaxPower)
const powerPercent = computed(() => store.powerUsagePercent)

const progressColor = computed(() => {
  const p = powerPercent.value
  if (p < 60) return '#67c23a'
  if (p < 85) return '#e6a23c'
  return '#f56c6c'
})
</script>

<style lang="scss" scoped>
.overview {
  flex-shrink: 0;
}

.total-power {
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid rgba(64, 158, 255, 0.15);

  .power-label {
    font-size: 13px;
    color: #909399;
    margin-bottom: 8px;
  }

  .power-value {
    display: flex;
    align-items: baseline;
    margin-bottom: 12px;

    .num {
      font-size: 42px;
      font-weight: 700;
      color: #409eff;
      font-family: 'Consolas', 'Monaco', monospace;
      line-height: 1;
    }

    .unit {
      font-size: 18px;
      color: #409eff;
      margin: 0 4px 0 2px;
    }

    .divider {
      color: #606266;
      margin: 0 8px;
    }

    .total {
      font-size: 20px;
      color: #606266;
      font-weight: 500;
    }
  }

  .power-info {
    display: flex;
    justify-content: space-between;
    margin-top: 6px;
    font-size: 12px;
    color: #909399;
  }
}

.stats-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.05);
  transition: all 0.3s;

  &:hover {
    transform: translateY(-2px);
    background: rgba(255, 255, 255, 0.04);
  }

  &.charging {
    color: #67c23a;
    border-color: rgba(103, 194, 58, 0.2);
  }

  &.idle {
    color: #909399;
    border-color: rgba(144, 147, 153, 0.2);
  }

  &.fault {
    color: #f56c6c;
    border-color: rgba(245, 108, 108, 0.2);
  }

  &.total {
    color: #409eff;
    border-color: rgba(64, 158, 255, 0.2);
  }

  .stat-info {
    .stat-num {
      font-size: 24px;
      font-weight: 700;
      line-height: 1.2;
      font-family: 'Consolas', 'Monaco', monospace;
    }

    .stat-label {
      font-size: 12px;
      opacity: 0.8;
    }
  }
}
</style>
