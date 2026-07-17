# BookingFlow Admin

BookingFlow 是免费开源的预约排班与上门服务运营后台，覆盖服务目录、时段容量、预约订单、改约/取消、退款标记、评价、负责人排班和服务跟进。它只做运营流程，不做诊断、处方、真实支付扣款或真实客户数据。

## 闭环流程

1. 前台创建预约（`待确认`）。
2. 客户确认后进入 `已确认`，运营人员推进到 `候诊中`、`处理中`。
3. 服务完成后进入 `已完成`；每次状态变化写入 `appointment_events`，可从详情页回放。
4. 运营人员创建回访任务，执行完成后由 `待完成` 变为 `已完成`。
5. 所有写请求要求 `Idempotency-Key`，Redis 负责幂等结果和时段并发锁，MySQL 8.4 负责持久化。

## 服务订单闭环

服务订单状态为 `待确认 -> 已预约 -> 已签到 -> 服务中 -> 已完成`，任意未完成阶段可取消；订单完成后可提交一条评价。创建订单会原子扣减时段剩余容量，重复幂等键不会重复下单，所有状态、改约、退款和评价都会写入时间线事件。

```text
GET  /api/v1/services
GET  /api/v1/availability?serviceId=svc-cleaning&date=2026-07-20
GET  /api/v1/bookings?page=1&pageSize=20&status=已预约
GET  /api/v1/bookings/:id
GET  /api/v1/bookings/:id/events
POST /api/v1/bookings
POST /api/v1/bookings/:id/status
POST /api/v1/bookings/:id/reschedule
POST /api/v1/bookings/:id/refund
POST /api/v1/bookings/:id/review
```

```bash
# 一键启动 API + MySQL 8.4 + Redis 8（会自动加载合成演示数据）
docker compose -f deploy/docker-compose.yml up --build

# 或仅使用无外部依赖的内存模式运行 API
go run ./server

# 管理后台
cd web && npm install && npm run dev
```

前端默认请求 `/api/v1`，Vite 开发服务器会把 `/api` 和 `/healthz` 代理到 `http://localhost:8080`。部署到独立域名时，可在构建时设置 `VITE_API_BASE_URL=https://api.example.com`；客户端会自动补齐 `/api/v1`，所有创建、确认、状态推进、改约、退款、评价和回访完成请求都会自动生成 `Idempotency-Key`。

后台的“服务目录”“预约订单”“预约队列”和“回访任务”按钮会优先调用真实 API；API 暂不可用时保留内置演示数据并提示当前数据来源。侧栏“移动端体验”提供同一闭环的窄屏客户视图，支持选服务时段、创建订单、确认、签到、服务完成、改约、取消、退款和评价，便于用手机浏览器联调。

## 闭环 API 示例

```bash
# 创建预约（重复发送相同 Idempotency-Key 只会创建一次）
curl -X POST http://localhost:8080/api/v1/appointments \
  -H 'Content-Type: application/json' -H 'Idempotency-Key: demo-create-001' \
  -d '{"patient":"演示客户","department":"全科门诊","doctor":"林负责人","scheduledAt":"2026-07-16T09:00:00+08:00"}'

# 推进状态：待确认 -> 已确认 -> 候诊中 -> 处理中 -> 已完成（将 AP-1001 替换为上一步返回的 id）
curl -X POST http://localhost:8080/api/v1/appointments/AP-1001/checkin -H 'Idempotency-Key: demo-checkin-001'
curl -X POST http://localhost:8080/api/v1/appointments/AP-1001/status \
  -H 'Content-Type: application/json' -H 'Idempotency-Key: demo-waiting-001' -d '{"status":"候诊中"}'

# 查看审计事件
curl http://localhost:8080/api/v1/appointments/AP-1001/events

# 完成回访
curl -X POST http://localhost:8080/api/v1/followups/FW-0716-001/complete -H 'Idempotency-Key: demo-followup-001'

# 创建上门服务订单（替换服务和时段后可重复联调）
curl -X POST http://localhost:8080/api/v1/bookings \
  -H 'Content-Type: application/json' -H 'Idempotency-Key: demo-booking-001' \
  -d '{"serviceId":"svc-cleaning","slotId":"slot-cleaning-0900","customerId":"CUS-001","customerName":"杭州星河家庭","startsAt":"2026-07-20T09:00:00+08:00","endsAt":"2026-07-20T11:00:00+08:00"}'
```

演示数据均为虚构数据；项目不得用于真实医疗诊断、处方、支付或客户隐私存储。

## 产品边界

BookingFlow 是免费开源的预约排班与服务订单样板，覆盖目录、锁时段、创建、确认、签到、履约、完成、改约、取消、退款标记、评价和回访的完整闭环。所有演示数据均为虚构，不接入真实财务、人事或客户隐私。

