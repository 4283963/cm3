# ⚡ 超级充电站动态功率分配系统

新能源汽车超级充电站智能监控与动态功率分配系统。当电网总功率有限时（默认 500kW），根据每辆电车的当前电量（SOC）和电池最高接受功率，智能计算并动态分配电流。

## 🏗️ 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                        前端 Vue3 监控大屏                      │
│  ┌──────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │ 电站总览  │  │ 充电桩状态网格 │  │ 功率趋势曲线 / 分配日志 │   │
│  └──────────┘  └──────────────┘  └──────────────────────┘   │
└─────────────────────────────┬───────────────────────────────┘
                              │ REST API + WebSocket
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Go (Gin) 后端服务                        │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────┐  │
│  │ 插桩感知 API │  │ 功率分配引擎  │  │ WebSocket 推送服务  │  │
│  └─────────────┘  └──────────────┘  └────────────────────┘  │
└─────────────────────────────┬───────────────────────────────┘
                              │ GORM
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                        MySQL 数据库                            │
│  vehicles | chargers | power_allocation_records | status     │
└─────────────────────────────────────────────────────────────┘
```

## ✨ 核心功能

### 🔋 智能动态功率分配算法
- **SOC 权重分配**：电量越低的车辆分配权重越高（紧急优先）
  - SOC < 20%：权重 2.0，紧急系数 1.5x
  - 20% ≤ SOC < 50%：权重 1.5
  - 50% ≤ SOC < 80%：权重 1.0
  - 80% ≤ SOC < 90%：权重 0.6
  - SOC ≥ 90%：权重 0.3
- **电池曲线模拟**：模拟真实电池充电特性
  - 低电量段(0-20%)：全速 95% 功率
  - 中段(20-80%)：恒流阶段 70-90%
  - 高电量段(80%+)：涓流阶段 5-40%
- **盈余再分配**：满足需求后多余功率自动重新分配给其他车辆
- **双重限流**：同时受车辆最大接受功率 + 充电桩最大功率限制

### 🚗 API 接口
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/station/plug-in` | 车辆插桩充电 |
| POST | `/api/station/plug-out` | 车辆拔桩结束 |
| GET | `/api/station/chargers` | 获取所有充电桩状态 |
| GET | `/api/station/status` | 获取电站整体状态 |
| GET | `/api/station/power-history` | 获取功率分配历史曲线 |
| POST | `/api/station/update-soc` | 更新车辆 SOC |
| POST | `/api/station/allocate` | 手动触发功率计算 |
| GET | `/ws` | WebSocket 实时数据推送 |
| GET | `/health` | 健康检查 |

### 📊 监控大屏
- **电站总览**：当前总功率/限额、使用率进度条、充电桩状态统计
- **功率饼图**：各充电桩实时功率占比可视化
- **充电桩网格**：10 个充电桩卡片，SOC 进度条带目标线、充电中/空闲/故障状态
- **趋势曲线**：多时间维度（1/6/24小时）各桩功率分配折线堆叠图
- **分配日志**：实时显示每次功率调整的分配明细
- **WebSocket 实时推送**：5 秒自动触发一次分配 + SOC 模拟增长

## 🚀 快速开始

### 方式一：Docker 启动数据库 + 本地运行

```bash
# 1. 启动 MySQL（需要安装 Docker）
docker compose up -d

# 2. 启动后端服务
cd backend
go mod tidy
go run .
# 服务监听: http://localhost:8080

# 3. 启动前端（新开终端）
cd frontend
npm install
npm run dev
# 访问: http://localhost:3000
```

### 方式二：使用本地 MySQL
1. 创建数据库：
   ```sql
   CREATE DATABASE supercharger CHARACTER SET utf8mb4;
   ```
2. 修改 `backend/config/config.go` 中的数据库连接信息
3. 启动后端（首次运行会自动建表并初始化 10 个充电桩）
4. 启动前端

## 📁 项目结构

```
cm3/
├── backend/                          # Go 后端
│   ├── config/                       # 配置
│   │   └── config.go
│   ├── database/                     # 数据库初始化
│   │   └── database.go
│   ├── handlers/                     # HTTP 控制器
│   │   └── station_handler.go
│   ├── models/                       # 数据模型
│   │   └── models.go
│   ├── routes/                       # 路由注册
│   │   └── routes.go
│   ├── services/                     # 业务服务
│   │   ├── power_engine.go           # 🔥 功率分配核心引擎
│   │   ├── station_manager.go        # 电站管理器
│   │   └── websocket.go              # WebSocket 推送
│   ├── sql/
│   │   └── init.sql
│   ├── main.go
│   └── go.mod
├── frontend/                         # Vue3 前端
│   ├── src/
│   │   ├── api/                      # API 请求
│   │   │   └── station.js
│   │   ├── components/               # UI 组件
│   │   │   ├── AllocationLog.vue     # 分配日志
│   │   ├── ChargerCard.vue           # 充电桩卡片
│   │   │   ├── ChargersGrid.vue      # 充电桩网格
│   │   │   ├── PlugInDialog.vue      # 插桩模拟对话框
│   │   │   ├── PowerTrendChart.vue   # 功率趋势图
│   │   │   ├── PowerUsageChart.vue   # 功率分布饼图
│   │   │   └── StationOverview.vue   # 电站概览
│   │   ├── stores/                   # Pinia 状态管理
│   │   │   └── station.js
│   │   ├── styles/
│   │   │   └── global.scss
│   │   ├── utils/
│   │   │   └── websocket.js          # WebSocket 客户端
│   │   ├── App.vue
│   │   └── main.js
│   ├── index.html
│   ├── package.json
│   └── vite.config.js
├── docker-compose.yml                # MySQL 容器
└── README.md
```

## 🧪 测试流程

1. 打开监控大屏：http://localhost:3000
2. 点击右上角 **「模拟插桩」** 按钮
3. 选择空闲充电桩，输入车辆信息（车牌号、SOC、电池参数等）
4. 提交后观察：
   - 充电桩卡片变为绿色"充电中"状态
   - SOC 进度条动画 + 目标电量指示线
   - 功率饼图出现该充电桩的色块
   - 分配日志出现本次分配记录
5. 多插几辆车（总需求 > 500kW），触发动态分配算法：
   - 低电量车辆获得更高功率倾斜
   - 高电量车辆自动降功率进入涓流
6. 等待或 SOC 达到 100%，观察车辆自动结束充电

## 🔧 配置说明

修改 `backend/config/config.go` 可调整：
```go
StationConfig{
    TotalMaxPower: 500.0,   // 电网限额 (kW)
    ChargerCount:  10,      // 充电桩数量
}
ServerConfig{
    Port: ":8080",          // 后端端口
}
```

## 🛠️ 技术栈

| 层级 | 技术 |
|------|------|
| 前端框架 | Vue 3.3 + Composition API |
| UI 组件 | Element Plus 2.4 |
| 状态管理 | Pinia 2 |
| 图表库 | ECharts 5 + vue-echarts |
| 构建工具 | Vite 5 |
| 后端框架 | Gin 1.10 |
| ORM | GORM 1.25 |
| 数据库 | MySQL 8.0 |
| 实时通信 | Gorilla WebSocket |
| 样式 | SCSS |
