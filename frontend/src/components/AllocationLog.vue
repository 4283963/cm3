<template>
  <div class="allocation-log card" style="height: 340px; display: flex; flex-direction: column;">
    <div class="card-header">
      <el-icon><Document /></el-icon>
      <span>功率分配日志</span>
      <el-tag size="small" type="info" style="margin-left: auto;">
        {{ store.allocationHistory.length }} 条记录
      </el-tag>
    </div>
    <div class="card-body log-body">
      <div v-if="store.allocationHistory.length === 0" class="empty">
        <el-empty description="暂无分配记录" :image-size="60" />
      </div>
      <div v-else class="log-list">
        <div
          v-for="(log, idx) in store.allocationHistory"
          :key="idx"
          class="log-item"
        >
          <div class="log-time">
            <el-icon><Timer /></el-icon>
            {{ log.time }}
          </div>
          <div class="log-content">
            <div
              v-for="(item, key) in log.data"
              :key="key"
              class="alloc-row"
            >
              <span class="charger">{{ key }}号桩</span>
              <span class="power">{{ Number(item.allocatedPower).toFixed(1) }} kW</span>
              <el-progress
                :percentage="Math.min(100, (item.allocatedPower / 120) * 100)"
                :stroke-width="4"
                :show-text="false"
                color="#67c23a"
                style="width: 60px;"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { Document, Timer } from '@element-plus/icons-vue'
import { useStationStore } from '../stores/station'

const store = useStationStore()
</script>

<style lang="scss" scoped>
.allocation-log {
  flex-shrink: 0;
}

.log-body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  padding: 8px 12px;
}

.empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}

.log-list {
  height: 100%;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.log-item {
  background: rgba(64, 158, 255, 0.04);
  border: 1px solid rgba(64, 158, 255, 0.08);
  border-radius: 6px;
  padding: 8px 10px;

  .log-time {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 11px;
    color: #409eff;
    font-family: 'Consolas', monospace;
    margin-bottom: 6px;
    padding-bottom: 4px;
    border-bottom: 1px dashed rgba(64, 158, 255, 0.1);
  }
}

.alloc-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 2px 0;
  font-size: 12px;

  .charger {
    width: 50px;
    color: #c0c4cc;
    font-weight: 500;
  }

  .power {
    width: 70px;
    color: #67c23a;
    font-family: 'Consolas', monospace;
    font-weight: 600;
  }
}
</style>
