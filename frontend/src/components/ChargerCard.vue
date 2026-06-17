<template>
  <div
    class="charger-card"
    :class="[charger.status]"
    @click="isCharging && $emit('view-detail', charger)"
  >
    <div class="card-top">
      <div class="charger-id">
        <span class="label">桩号</span>
        <span class="name">{{ charger.name }}</span>
      </div>
      <div class="status-badge">
        <i class="status-dot"></i>
        {{ statusText }}
      </div>
    </div>

    <div v-if="isCharging && charger.currentVehicle" class="charging-info">
      <div class="vehicle-info">
        <el-icon><Van /></el-icon>
        <span class="plate">{{ charger.currentVehicle.licensePlate || charger.currentVehicle.vin }}</span>
      </div>

      <div class="soc-section">
        <div class="soc-header">
          <span>电量 SOC</span>
          <span class="soc-value">{{ socPercent.toFixed(1) }}%</span>
        </div>
        <div class="soc-bar-wrap">
          <div class="soc-bar">
            <div
              class="soc-fill charging-bar"
              :style="{ width: `${socPercent}%`, background: socColor }"
            ></div>
            <div class="soc-target" :style="{ left: `${charger.currentVehicle.targetSOC}%` }">
              <i></i>
              <span>{{ charger.currentVehicle.targetSOC }}%</span>
            </div>
          </div>
        </div>
      </div>

      <div class="power-section">
        <div class="power-item">
          <span class="p-label">分配功率</span>
          <span class="p-value highlight">{{ charger.currentPower?.toFixed(1) }} kW</span>
        </div>
        <div class="power-item">
          <span class="p-label">最高接受</span>
          <span class="p-value">{{ charger.currentVehicle.maxAcceptPower }} kW</span>
        </div>
      </div>

      <div class="battery-info">
        <div class="info-row">
          <span>电池容量</span>
          <span>{{ charger.currentVehicle.batteryCapacity }} kWh</span>
        </div>
        <div class="info-row" v-if="charger.currentVehicle.estimatedEndTime">
          <span>预计结束</span>
          <span>{{ formatTime(charger.currentVehicle.estimatedEndTime) }}</span>
        </div>
      </div>

      <el-button
        type="danger"
        size="small"
        class="plugout-btn"
        @click.stop="$emit('plug-out', charger.id)"
      >
        <el-icon><Switch /></el-icon>
        结束充电
      </el-button>
    </div>

    <div v-else-if="charger.status === 'idle'" class="idle-info">
      <el-icon :size="48" class="idle-icon"><Power /></el-icon>
      <p>空闲可用</p>
      <p class="max-power">最大输出: {{ charger.maxPower }} kW</p>
    </div>

    <div v-else class="fault-info">
      <el-icon :size="48" color="#f56c6c"><Warning /></el-icon>
      <p>设备故障</p>
      <p class="fault-tip">请联系运维</p>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Van, Power, Warning, Switch } from '@element-plus/icons-vue'

const props = defineProps({
  charger: { type: Object, required: true },
})
defineEmits(['plug-out', 'view-detail'])

const STATUS_MAP = {
  charging: '充电中',
  idle: '空闲',
  fault: '故障',
  reserved: '预约中',
}

const isCharging = computed(() => props.charger.status === 'charging')
const statusText = computed(() => STATUS_MAP[props.charger.status] || props.charger.status)

const socPercent = computed(() => {
  if (props.charger.currentVehicle) {
    return props.charger.currentVehicle.currentSOC
  }
  return 0
})

const socColor = computed(() => {
  const soc = socPercent.value
  if (soc < 20) return 'linear-gradient(90deg, #f56c6c, #e6a23c)'
  if (soc < 80) return 'linear-gradient(90deg, #e6a23c, #67c23a)'
  return 'linear-gradient(90deg, #67c23a, #409eff)'
})

const formatTime = (t) => {
  if (!t) return '--'
  const d = new Date(t)
  const pad = (n) => n.toString().padStart(2, '0')
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}
</script>

<style lang="scss" scoped>
.charger-card {
  border-radius: 10px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.3s ease;
  position: relative;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: linear-gradient(145deg, rgba(19, 38, 66, 0.7), rgba(13, 33, 55, 0.85));

  &:hover {
    transform: translateY(-3px);
  }

  &.charging {
    border-color: rgba(103, 194, 58, 0.4);
    background: linear-gradient(145deg, rgba(26, 58, 38, 0.6), rgba(19, 38, 66, 0.85));

    &:hover {
      box-shadow: 0 8px 24px rgba(103, 194, 58, 0.2);
    }
  }

  &.fault {
    border-color: rgba(245, 108, 108, 0.4);
    background: linear-gradient(145deg, rgba(58, 26, 26, 0.6), rgba(19, 38, 66, 0.85));
  }

  &.idle {
    &:hover {
      border-color: rgba(64, 158, 255, 0.3);
      box-shadow: 0 8px 24px rgba(64, 158, 255, 0.15);
    }
  }
}

.card-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 10px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.charger-id {
  display: flex;
  align-items: baseline;
  gap: 6px;

  .label {
    font-size: 11px;
    color: #909399;
  }

  .name {
    font-size: 20px;
    font-weight: 700;
    color: #409eff;
    font-family: 'Consolas', monospace;
  }
}

.status-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  border-radius: 10px;
  font-size: 12px;
  background: rgba(144, 147, 153, 0.15);
  color: #909399;

  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: currentColor;
  }
}

.charging {
  .status-badge {
    background: rgba(103, 194, 58, 0.15);
    color: #67c23a;

    .status-dot {
      animation: pulse 1.5s infinite;
    }
  }
}

.fault {
  .status-badge {
    background: rgba(245, 108, 108, 0.15);
    color: #f56c6c;
  }
}

.vehicle-info {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
  font-size: 14px;
  color: #c0c4cc;

  .plate {
    font-weight: 600;
    color: #8cc5ff;
  }
}

.soc-section {
  margin-bottom: 12px;

  .soc-header {
    display: flex;
    justify-content: space-between;
    font-size: 12px;
    margin-bottom: 6px;
    color: #909399;

    .soc-value {
      font-weight: 700;
      font-family: 'Consolas', monospace;
      color: #e4e7ed;
    }
  }
}

.soc-bar-wrap {
  position: relative;
}

.soc-bar {
  height: 14px;
  background: rgba(144, 147, 153, 0.15);
  border-radius: 7px;
  overflow: visible;
  position: relative;
}

.soc-fill {
  height: 100%;
  border-radius: 7px;
  transition: width 0.8s ease;
}

.soc-target {
  position: absolute;
  top: -18px;
  transform: translateX(-50%);
  display: flex;
  flex-direction: column;
  align-items: center;

  i {
    width: 2px;
    height: 20px;
    background: #e6a23c;
    display: block;
    margin-top: 18px;
  }

  span {
    font-size: 10px;
    color: #e6a23c;
    white-space: nowrap;
  }
}

.power-section {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
  padding: 10px;
  background: rgba(64, 158, 255, 0.06);
  border-radius: 6px;
}

.power-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;

  .p-label {
    font-size: 10px;
    color: #909399;
  }

  .p-value {
    font-size: 14px;
    font-weight: 600;
    font-family: 'Consolas', monospace;
    color: #e4e7ed;

    &.highlight {
      color: #67c23a;
      font-size: 16px;
    }
  }
}

.battery-info {
  margin-bottom: 12px;
  padding: 8px 0;
  border-top: 1px dashed rgba(255, 255, 255, 0.06);

  .info-row {
    display: flex;
    justify-content: space-between;
    font-size: 12px;
    padding: 3px 0;
    color: #909399;

    span:last-child {
      color: #c0c4cc;
      font-family: 'Consolas', monospace;
    }
  }
}

.plugout-btn {
  width: 100%;
}

.idle-info,
.fault-info {
  padding: 20px 12px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;

  p {
    font-size: 14px;
    color: #909399;
    margin: 0;
  }

  .max-power {
    font-size: 12px;
    color: #606266;
  }

  .fault-tip {
    font-size: 12px;
    color: #f56c6c;
  }
}

.idle-icon {
  color: #409eff;
  opacity: 0.6;
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.6; transform: scale(1.2); }
}
</style>
