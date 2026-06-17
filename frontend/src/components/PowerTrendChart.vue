<template>
  <div class="power-trend card" style="flex: 1; display: flex; flex-direction: column;">
    <div class="card-header">
      <el-icon><TrendCharts /></el-icon>
      <span>功率分配趋势</span>
      <el-select v-model="timeRange" size="small" style="margin-left: auto; width: 90px;">
        <el-option label="最近1小时" :value="1" />
        <el-option label="最近6小时" :value="6" />
        <el-option label="最近24小时" :value="24" />
      </el-select>
    </div>
    <div class="card-body" style="flex: 1; min-height: 0; padding: 8px;">
      <v-chart :option="chartOption" autoresize style="height: 100%; width: 100%;" />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { use } from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent, GridComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { TrendCharts } from '@element-plus/icons-vue'
import { useStationStore } from '../stores/station'

use([LineChart, TooltipComponent, LegendComponent, GridComponent, CanvasRenderer])

const store = useStationStore()
const timeRange = ref(6)

const COLORS = [
  '#409eff', '#67c23a', '#e6a23c', '#f56c6c', '#909399',
  '#9b59b6', '#1abc9c', '#e91e63', '#00bcd4', '#ff9800',
]

const chartOption = computed(() => {
  const history = store.powerHistory
  const timeMap = new Map()
  const chargerData = {}

  for (let i = 1; i <= 10; i++) {
    chargerData[i] = []
  }

  history.forEach((r) => {
    const t = new Date(r.timestamp)
    const timeKey = `${t.getHours().toString().padStart(2, '0')}:${t.getMinutes().toString().padStart(2, '0')}:${t.getSeconds().toString().padStart(2, '0')}`
    timeMap.set(timeKey, 1)

    if (chargerData[r.chargerId]) {
      chargerData[r.chargerId].push({ time: timeKey, value: r.allocatedPower })
    }
  })

  const times = Array.from(timeMap.keys()).sort()

  const series = []
  for (let id = 1; id <= 10; id++) {
    const data = chargerData[id]
    if (data.length === 0) continue

    const filledData = times.map((t) => {
      const found = data.find((d) => d.time === t)
      return found ? found.value : 0
    })

    series.push({
      name: `${id}号桩`,
      type: 'line',
      smooth: true,
      symbol: 'none',
      stack: undefined,
      lineStyle: { width: 2, color: COLORS[id - 1] },
      areaStyle: {
        opacity: 0.08,
        color: COLORS[id - 1],
      },
      emphasis: { focus: 'series' },
      data: filledData,
    })
  }

  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(13, 33, 55, 0.95)',
      borderColor: 'rgba(64, 158, 255, 0.3)',
      textStyle: { color: '#e4e7ed', fontSize: 11 },
      axisPointer: {
        type: 'cross',
        lineStyle: { color: 'rgba(64, 158, 255, 0.3)' },
      },
    },
    legend: {
      type: 'scroll',
      bottom: 0,
      textStyle: { color: '#909399', fontSize: 10 },
      pageTextStyle: { color: '#909399' },
      itemWidth: 12,
      itemHeight: 8,
      pageIconSize: 10,
    },
    grid: {
      left: 45,
      right: 15,
      top: 10,
      bottom: 40,
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: times.length ? times : ['00:00:00', '00:05:00'],
      axisLine: { lineStyle: { color: 'rgba(64, 158, 255, 0.15)' } },
      axisLabel: {
        color: '#606266',
        fontSize: 10,
        interval: Math.floor(times.length / 8) || 0,
      },
      splitLine: { show: false },
    },
    yAxis: {
      type: 'value',
      name: 'kW',
      nameTextStyle: { color: '#606266', fontSize: 10 },
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { color: '#606266', fontSize: 10 },
      splitLine: { lineStyle: { color: 'rgba(64, 158, 255, 0.08)' } },
      max: 120,
    },
    series: series.length ? series : [{
      name: '暂无数据',
      type: 'line',
      smooth: true,
      symbol: 'none',
      lineStyle: { width: 2, color: 'rgba(144,147,153,0.3)' },
      data: [],
    }],
  }
})

watch(timeRange, (val) => {
  store.fetchPowerHistory({ hours: val })
})

onMounted(() => {
  store.fetchPowerHistory({ hours: timeRange.value })
})
</script>

<style lang="scss" scoped>
.power-trend {
  min-height: 0;
}
</style>
