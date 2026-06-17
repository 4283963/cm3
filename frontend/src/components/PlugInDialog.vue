<template>
  <el-dialog
    v-model="dialogVisible"
    title="模拟车辆插桩充电"
    width="500px"
    :close-on-click-modal="false"
    destroy-on-close
  >
    <el-form :model="form" label-width="100px" ref="formRef">
      <el-form-item label="充电桩" prop="chargerId" :rules="[{ required: true, message: '请选择充电桩' }]">
        <el-select v-model="form.chargerId" placeholder="请选择充电桩">
          <el-option
            v-for="c in availableChargers"
            :key="c.id"
            :label="`${c.name}号桩 - (空闲)`"
            :value="c.id"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="车牌号" prop="licensePlate" :rules="[{ required: true, message: '请输入车牌号' }]">
        <el-input v-model="form.licensePlate" placeholder="如: 京A88888" maxlength="10" />
      </el-form-item>

      <el-form-item label="车架号VIN" prop="vin" :rules="[{ required: true, message: '请输入车架号' }]">
        <el-input v-model="form.vin" placeholder="17位车架号" maxlength="17" />
      </el-form-item>

      <el-form-item label="电池容量" prop="batteryCapacity" :rules="[{ required: true, message: '请输入电池容量' }]">
        <el-input-number
          v-model="form.batteryCapacity"
          :min="20"
          :max="200"
          :step="5"
          style="width: 100%;"
        />
        <span style="font-size: 12px; color: #909399; margin-left: 8px;">kWh</span>
      </el-form-item>

      <el-form-item label="当前电量SOC" prop="currentSOC" :rules="[{ required: true, message: '请输入当前电量' }]">
        <el-slider
          v-model="form.currentSOC"
          :min="1"
          :max="99"
          :marks="{ 20: '20%', 50: '50%', 80: '80%' }"
          show-input
          style="padding-right: 100px;"
        />
      </el-form-item>

      <el-form-item label="目标电量" prop="targetSOC" :rules="[{ required: true, message: '请输入目标电量' }]">
        <el-slider
          v-model="form.targetSOC"
          :min="form.currentSOC"
          :max="100"
          :marks="{ 80: '80%', 90: '90%', 100: '100%' }"
          show-input
          style="padding-right: 100px;"
        />
      </el-form-item>

      <el-form-item label="最高接受功率" prop="maxAcceptPower" :rules="[{ required: true, message: '请输入最大接受功率' }]">
        <el-radio-group v-model="form.maxAcceptPower">
          <el-radio :value="60">60 kW</el-radio>
          <el-radio :value="90">90 kW</el-radio>
          <el-radio :value="120">120 kW</el-radio>
          <el-radio :value="150">150 kW</el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="车主等级">
        <el-radio-group v-model="form.isVip">
          <el-radio :value="false">普通车主</el-radio>
          <el-radio :value="true">
            <el-tag type="warning" effect="dark" size="small" style="margin-right: 4px;">VIP</el-tag>
            VIP 套餐车主（电网限电时优先保电）
          </el-radio>
        </el-radio-group>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" @click="submitForm" :loading="submitting">
        开始充电
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, reactive, watch } from 'vue'
import { useStationStore } from '../stores/station'

const props = defineProps({
  visible: { type: Boolean, default: false },
})
const emit = defineEmits(['update:visible', 'success'])

const store = useStationStore()
const formRef = ref(null)
const submitting = ref(false)

const dialogVisible = computed({
  get: () => props.visible,
  set: (v) => emit('update:visible', v),
})

const form = reactive({
  chargerId: null,
  licensePlate: '',
  vin: '',
  batteryCapacity: 75,
  currentSOC: 30,
  targetSOC: 90,
  maxAcceptPower: 120,
  isVip: false,
})

const availableChargers = computed(() => {
  return store.chargers.filter((c) => c.status === 'idle')
})

watch(
  () => form.currentSOC,
  (val) => {
    if (form.targetSOC <= val) {
      form.targetSOC = Math.min(100, val + 10)
    }
  }
)

watch(
  () => props.visible,
  (v) => {
    if (v && availableChargers.value.length > 0) {
      form.chargerId = availableChargers.value[0].id
      form.licensePlate = `京A${Math.floor(10000 + Math.random() * 90000)}`
      form.vin = `LFM${Date.now().toString().slice(-14)}`
    }
  }
)

const submitForm = async () => {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      const res = await store.plugIn({ ...form })
      if (res.success) {
        emit('success')
      }
    } finally {
      submitting.value = false
    }
  })
}
</script>
