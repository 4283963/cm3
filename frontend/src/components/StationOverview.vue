<template>
  <div class="overview card">
    <div v-if="status.gridLimitMode" class="grid-limit-banner">
      <el-icon :size="18"><Warning /></el-icon>
      <span class="banner-text">
        电网限电响应中 · 总功率上限 <b>{{ status.currentLimitPower.toFixed(0) }} kW</b>
        （已强制削峰 <b>{{ cutPercent }}%</b>） · <em>VIP 车主功率优先保障</em>
      </span>
      <el-tag type="danger" effect="dark" size="small" class="banner-tag">
        {{ status.vipChargers }} 辆 VIP 保电中
      </el-tag>
    </div>

    <div class="card-header">
      <el-icon><DataAnalysis /></el-icon>
      <span>电站总览</span>
      <div class="header-actions">
        <span class="grid-limit-label">
          <el-icon v-if="status.gridLimitMode"><Lightning /></el-icon>
          <el-icon v-else><Connection /></el-icon>
          电网限电响应
        </span>
        <el-switch
          :model-value="status.gridLimitMode"
          :loading="store.gridLimitSwitching"
          active-color="#f56c6c"
          inactive-color="#67c23a"
          @change="onGridLimitChange"
        />
      </div>
    </div>

    <div class="card-body">
      <div class="total-power">
        <div class="power-label">
          当前总功率
          <span v-if="status.gridLimitMode" class="limit-tag">(上限 {{ status.currentLimitPower.toFixed(0) }} kW)</span>
        </div>
        <div class="power-value">
          <span class="num" :class="{ 'is-limit': status.gridLimitMode }">
            {{ currentPower.toFixed(1) }}
          </span>
          <span class="unit">kW</span>
          <span class="divider">/</span>
          <span class="total">{{ basePower.toFixed(0) }} kW</span>
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

        <div v-if="status.gridLimitMode" class="vip-summary">
          <div class="vip-row">
            <el-icon color="#e6a23c"><Medal /></el-icon>
            <span>VIP 保电功率</span>
            <span class="vip-val">{{ status.vipProtectedPower.toFixed(1) }} kW</span>
          </div>
          <div class="vip-row cut">
            <el-icon color="#f56c6c"><TrendCharts /></el-icon>
            <span>已削减普通车</span>
            <span class="vip-val">−{{ normalReduced.toFixed(1) }} kW</span>
          </div>
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
        <div class="stat-item vip">
          <el-icon :size="24"><Medal /></el-icon>
          <div class="stat-info">
            <div class="stat-num">{{ status.vipChargers }}</div>
            <div class="stat-label">VIP 保电</div>
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
        <div class="stat-item total wide">
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
import {
  DataAnalysis,
  Charge,
  Power,
  Warning,
  Van,
  Medal,
  Lightning,
  Connection,
  TrendCharts,
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const store = useStationStore()

const status = computed(() => store.stationStatus)
const currentPower = computed(() => store.stationStatus.currentTotalPower)
const totalMaxPower = computed(() => store.stationStatus.totalMaxPower)
const powerPercent = computed(() => store.powerUsagePercent)
const cutPercent = computed(() => store.gridLimitCutPercent)

const basePower = computed(() =>
  status.value.gridLimitMode ? status.value.currentLimitPower : totalMaxPower.value
)
const normalReduced = computed(() => status.value.normalReducedPower || 0)

const progressColor = computed(() => {
  const p = powerPercent.value
  if (p < 60) return '#67c23a'
  if (p < 85) return '#e6a23c'
  return '#f56c6c'
})

async function onGridLimitChange(newVal) {
  if (newVal) {
    try {
      await ElMessageBox.confirm(
        '开启后电网总功率上限将强制砍半（50%），1秒内完成功率调整。\nVIP 车主将优先保障充电功率，仅削减普通车主功率。是否继续？',
        '电网限电响应',
        {
          confirmButtonText: '确认开启削峰',
          cancelButtonText: '取消',
          type: 'warning',
          confirmButtonClass: 'el-button--danger',
        }
      )
    } catch (_) {
      return
    }
    const res = await store.setGridLimitMode(true, 0.5)
    if (res.success) {
      ElMessage.success('电网限电模式已开启，VIP 优先保障策略已生效')
    } else {
      ElMessage.error(res.message || '开启失败')
    }
  } else {
    const res = await store.setGridLimitMode(false)
    if (res.success) {
      ElMessage.success('电网限电已解除，所有充电桩恢复全功率运行')
    } else {
      ElMessage.error(res.message || '关闭失败')
    }
  }
}
</script>

<style lang="scss" scoped>
.overview {
  flex-shrink: 0;
}

.grid-limit-banner {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  margin: -16px -20px 16px;
  background: linear-gradient(90deg, rgba(245, 108, 108, 0.15), rgba(245, 108, 108, 0.05));
  border-bottom: 1px dashed rgba(245, 108, 108, 0.4);
  color: #f56c6c;
  font-size: 13px;

  .banner-text {
    flex: 1;

    b {
      color: #f56c6c;
      font-weight: 600;
      margin: 0 2px;
    }

    em {
      color: #e6a23c;
      font-style: normal;
      margin-left: 6px;
    }
  }

  .banner-tag {
    margin-left: auto;
  }
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 16px;
  color: #e6a23c;
  padding-bottom: 14px;
  margin-bottom: 16px;
  border-bottom: 1px solid rgba(64, 158, 255, 0.15);

  .header-actions {
    margin-left: auto;
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 13px;
    color: #909399;
    font-weight: 500;

    .grid-limit-label {
      display: inline-flex;
      align-items: center;
      gap: 4px;
    }
  }
}

.total-power {
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid rgba(64, 158, 255, 0.15);

  .power-label {
    font-size: 13px;
    color: #909399;
    margin-bottom: 8px;

    .limit-tag {
      color: #f56c6c;
      margin-left: 6px;
    }
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

      &.is-limit {
        color: #e6a23c;
      }
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

.vip-summary {
  margin-top: 14px;
  padding: 10px 12px;
  background: linear-gradient(90deg, rgba(230, 162, 60, 0.1), rgba(64, 158, 255, 0.05));
  border-radius: 6px;
  border-left: 3px solid #e6a23c;
  display: flex;
  flex-direction: column;
  gap: 6px;

  .vip-row {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: #c0c4cc;

    .vip-val {
      margin-left: auto;
      font-family: 'Consolas', 'Monaco', monospace;
      font-weight: 600;
      color: #e6a23c;
    }

    &.cut .vip-val {
      color: #f56c6c;
    }
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

  &.vip {
    color: #e6a23c;
    border-color: rgba(230, 162, 60, 0.25);
    background: linear-gradient(135deg, rgba(230, 162, 60, 0.08), rgba(255, 255, 255, 0.02));
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

  &.wide {
    grid-column: span 2;
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
