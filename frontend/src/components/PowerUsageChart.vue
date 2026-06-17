<template>
  <div class="power-usage card" style="flex: 1; display: flex; flex-direction: column;">
    <div class="card-header">
      <el-icon><PieChart /></el-icon>
      <span>充电桩功率分布</span>
    </div>
    <div class="card-body" style="flex: 1; min-height: 0; display: flex; flex-direction: column;">
      <div class="chart-container">
        <v-chart :option="chartOption" autoresize />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { use } from 'echarts/core'
import { PieChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { PieChart as PieIcon } from '@element-plus/icons-vue'
import { useStationStore } from '../stores/station'

use([PieChart, TooltipComponent, LegendComponent, CanvasRenderer])

const store = useStationStore()

const COLORS = [
  '#409eff',
  '#67c23a',
  '#e6a23c',
  '#f56c6c',
  '#909399',
  '#9b59b6',
  '#1abc9c',
  '#e91e63',
  '#00bcd4',
  '#ff9800',
]

const chartOption = computed(() => {
  const activeChargers = store.chargers.filter(c => c.status === 'charging')
  const totalPower = activeChargers.reduce((sum, c) => sum + (c.currentPower || 0), 0)
  const idlePower = Math.max(0, 500 - totalPower)

  const data = activeChargers.map((c, i) => ({
    name: `${c.name}号桩`,
    value: Number(c.currentPower?.toFixed(2)) || 0,
    itemStyle: { color: COLORS[c.id - 1] || COLORS[i % 10] },
  }))

  if (idlePower > 0) {
    data.push({
      name: '剩余容量',
      value: Number(idlePower.toFixed(2)),
      itemStyle: { color: 'rgba(144, 147, 153, 0.3)' },
    })
  }

  return {
    tooltip: {
      trigger: 'item',
      backgroundColor: 'rgba(13, 33, 55, 0.95)',
      borderColor: 'rgba(64, 158, 255, 0.3)',
      textStyle: { color: '#e4e7ed' },
      formatter: (params) => `${params.name}<br/>功率: <b>${params.value.toFixed(2)}</b> kW (${params.percent}%)`,
    },
    legend: {
      type: 'scroll',
      orient: 'vertical',
      right: 4,
      top: 'center',
      itemWidth: 10,
      itemHeight: 10,
      textStyle: { color: '#909399', fontSize: 11 },
      pageTextStyle: { color: '#909399' },
    },
    series: [
      {
        name: '功率分配',
        type: 'pie',
        radius: ['45%', '72%'],
        center: ['35%', '50%'],
        avoidLabelOverlap: true,
        itemStyle: {
          borderRadius: 4,
          borderColor: 'rgba(10, 22, 40, 0.8)',
          borderWidth: 2,
        },
        label: {
          show: false,
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 12,
            fontWeight: 'bold',
            color: '#fff',
          },
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)',
          },
        },
        data: data.length ? data : [{ name: '暂无充电', value: 500, itemStyle: { color: 'rgba(144, 147, 153, 0.2)' } }],
      },
    ],
  }
})
</script>

<style lang="scss" scoped>
.chart-container {
  width: 100%;
  height: 100%;
  min-height: 280px;
}
</style>
