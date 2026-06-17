<template>
  <div class="chargers-grid card" style="height: 100%; display: flex; flex-direction: column;">
    <div class="card-header">
      <el-icon><Connection /></el-icon>
      <span>充电桩实时状态</span>
      <div class="legend">
        <span class="legend-item"><i class="dot green"></i>充电中</span>
        <span class="legend-item"><i class="dot gray"></i>空闲</span>
        <span class="legend-item"><i class="dot red"></i>故障</span>
      </div>
    </div>
    <div class="card-body grid-body">
      <div class="grid">
        <ChargerCard
          v-for="charger in chargers"
          :key="charger.id"
          :charger="charger"
          @plug-out="(id) => $emit('plug-out', id)"
          @view-detail="(c) => $emit('view-detail', c)"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useStationStore } from '../stores/station'
import ChargerCard from './ChargerCard.vue'
import { Connection } from '@element-plus/icons-vue'

defineEmits(['plug-out', 'view-detail'])

const store = useStationStore()
const chargers = computed(() => store.chargers)
</script>

<style lang="scss" scoped>
.chargers-grid {
  min-height: 0;
}

.grid-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 12px;
}

.card-header {
  justify-content: flex-start;
  position: relative;

  .legend {
    position: absolute;
    right: 16px;
    display: flex;
    gap: 16px;
    font-size: 12px;
    color: #909399;
    font-weight: normal;
  }

  .legend-item {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    display: inline-block;

    &.green { background: #67c23a; box-shadow: 0 0 6px #67c23a; }
    &.gray { background: #606266; }
    &.red { background: #f56c6c; box-shadow: 0 0 6px #f56c6c; }
  }
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 16px;
}
</style>
