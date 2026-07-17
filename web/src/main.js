import './styles.css'
import './product-styles.css'
import { createApiClient } from './api.js'

const api = createApiClient()

const demoAppointments = [
  { id: 'BK-0716-082', patient: '杭州星河家庭', department: '深度保洁', doctor: '林然 · 服务顾问', scheduledAt: '2026-07-16T09:30:00+08:00', status: '候诊中' },
  { id: 'BK-0716-081', patient: '苏州云杉门店', department: '空调清洗', doctor: '沈宁 · 服务顾问', scheduledAt: '2026-07-16T09:45:00+08:00', status: '已确认' },
  { id: 'BK-0716-080', patient: '上海岸线公寓', department: '家电安装', doctor: '赵然 · 服务顾问', scheduledAt: '2026-07-16T10:00:00+08:00', status: '已完成' },
  { id: 'BK-0716-079', patient: '南京微光社区', department: '管道疏通', doctor: '林然 · 服务顾问', scheduledAt: '2026-07-16T10:15:00+08:00', status: '待确认' },
  { id: 'BK-0716-078', patient: '成都山海公寓', department: '家政钟点', doctor: '周宁 · 服务顾问', scheduledAt: '2026-07-16T10:30:00+08:00', status: '待确认' },
]

const demoFollowups = [
  { id: 'TASK-0716-012', patient: '杭州星河家庭', summary: '确认上门地址与服务物料', dueAt: '今天 16:00', status: '待完成' },
  { id: 'TASK-0716-011', patient: '苏州云杉门店', summary: '回访服务体验并邀请评价', dueAt: '今天 17:30', status: '待完成' },
  { id: 'TASK-0716-010', patient: '上海岸线公寓', summary: '确认续费会员权益', dueAt: '明天 09:30', status: '待完成' },
  { id: 'TASK-0715-009', patient: '南京微光社区', summary: '完成昨日订单归档', dueAt: '已完成', status: '已完成' },
]

const demoDashboard = { todayAppointments: 86, averageWaitMinutes: 12, completed: 58, checkedIn: 42, pendingFollowups: 12 }
const demoServices = [
  { id: 'svc-cleaning', name: '深度保洁', category: '家政保洁', description: '厨房、卫生间重点清洁，服务前确认清单', durationMinutes: 120, priceCents: 19900, active: true },
  { id: 'svc-aircon', name: '空调清洗', category: '家电清洗', description: '拆洗滤网与蒸发器，清新一夏', durationMinutes: 90, priceCents: 12900, active: true },
  { id: 'svc-install', name: '家电安装', category: '上门维修', description: '水电、门锁、小家电快速排障', durationMinutes: 120, priceCents: 8900, active: true },
]
const demoSlots = [
  { id: 'slot-cleaning-0900', serviceId: 'svc-cleaning', startsAt: '2026-07-20T09:00:00+08:00', endsAt: '2026-07-20T11:00:00+08:00', remaining: 1, capacity: 1, status: '可预约' },
  { id: 'slot-cleaning-1300', serviceId: 'svc-cleaning', startsAt: '2026-07-20T13:00:00+08:00', endsAt: '2026-07-20T15:00:00+08:00', remaining: 2, capacity: 2, status: '可预约' },
]
const demoBookings = [
  { id: 'BK-DEMO-001', serviceId: 'svc-cleaning', serviceName: '深度保洁', slotId: 'slot-cleaning-0900', customerName: '杭州星河家庭', startsAt: '2026-07-20T09:00:00+08:00', endsAt: '2026-07-20T11:00:00+08:00', status: '已预约', paymentStatus: '已支付', amountCents: 19900 },
  { id: 'BK-DEMO-002', serviceId: 'svc-aircon', serviceName: '空调清洗', slotId: 'slot-aircon-1000', customerName: '苏州云杉门店', startsAt: '2026-07-20T10:00:00+08:00', endsAt: '2026-07-20T11:30:00+08:00', status: '服务中', paymentStatus: '已支付', amountCents: 12900 },
  { id: 'BK-DEMO-003', serviceId: 'svc-install', serviceName: '家电安装', slotId: 'slot-install-1500', customerName: '上海岸线公寓', startsAt: '2026-07-20T15:00:00+08:00', endsAt: '2026-07-20T17:00:00+08:00', status: '已完成', paymentStatus: '已支付', amountCents: 8900 },
]
const statusColors = { 待确认: 'coral', 已确认: 'indigo', 候诊中: 'amber', 处理中: 'green', 已完成: 'green', 已取消: 'gray', 已预约: 'indigo', 已签到: 'amber', 服务中: 'green' }
const nav = [
  ['overview', '运营总览', '⌂'],
  ['queue', '预约队列', '▤'],
  ['doctors', '服务人员排班', '◉'],
  ['patients', '会员档案', '♧'],
  ['followups', '服务跟进', '✓'],
  ['services', '服务目录', '⌘'],
  ['bookings', '预约订单', '▣'],
  ['mobile', '移动端体验', '⌁'],
]

let appointments = demoAppointments.map((item) => ({ ...item }))
let followupTasks = demoFollowups.map((item) => ({ ...item }))
let dashboard = { ...demoDashboard }
let page = 'overview'
let toast = ''
let toastTimer
let dataSource = '演示数据'
let isSyncing = false
let services = demoServices.map((item) => ({ ...item }))
let slots = demoSlots.map((item) => ({ ...item }))
let bookings = demoBookings.map((item) => ({ ...item }))
let selectedBooking = null
let bookingEvents = []

function displayCopy(root) {
  const rules = [['候诊', '待到店'], ['健康回访', '到店跟进'], ['回访', '服务跟进'], ['临床', '预约'], ['科室', '服务类型'], ['人次', '单'], ['位客户', '个订单'], ['诊断', '真实服务'], ['林负责人', '林然 · 服务顾问'], ['沈负责人', '沈宁 · 服务顾问'], ['赵负责人', '赵然 · 服务顾问'], ['周负责人', '周宁 · 服务顾问'], ['陈负责人', '陈敏 · 服务顾问'], ['王负责人', '王可 · 服务顾问'], ['全科门诊', '深度保洁'], ['皮肤科', '空调清洗'], ['康复理疗', '家电安装'], ['营养咨询', '管道疏通'], ['就诊', '服务'], ['CF-', 'CUS-']]
  const walker = document.createTreeWalker(root, 4)
  while (walker.nextNode()) rules.forEach(([from, to]) => { walker.currentNode.nodeValue = walker.currentNode.nodeValue.replaceAll(from, to) })
}

function timeLabel(value) {
  const match = String(value ?? '').match(/T(\d{2}:\d{2})/)
  return match?.[1] || String(value ?? '').slice(0, 5) || '--:--'
}

function normalizeAppointment(item) {
  return {
    id: item.id,
    patientId: item.patientId,
    patient: item.patient || '未命名客户',
    department: item.department || '待分诊',
    doctor: item.doctor || '待安排',
    scheduledAt: item.scheduledAt || '',
    status: item.status || '待确认',
  }
}

function normalizeFollowup(item) {
  return {
    id: item.id,
    patientId: item.patientId,
    patient: item.patient || '未命名客户',
    summary: item.summary || '健康回访任务',
    dueAt: item.dueAt || '--',
    status: item.status || '待完成',
  }
}

function showToast(message) {
  toast = message
  render()
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => {
    toast = ''
    render()
  }, 2200)
}

function appointmentAction(appointment) {
  if (appointment.status === '待确认') return `<button class="text-action" data-action="checkin" data-appointment-id="${appointment.id}">确认</button>`
  if (appointment.status === '已确认') return `<button class="text-action" data-action="status" data-next-status="候诊中" data-appointment-id="${appointment.id}">进入候诊</button>`
  if (appointment.status === '候诊中') return `<button class="text-action" data-action="status" data-next-status="处理中" data-appointment-id="${appointment.id}">开始处理</button>`
  if (appointment.status === '处理中') return `<button class="text-action" data-action="status" data-next-status="已完成" data-appointment-id="${appointment.id}">完成处理</button>`
  return '<button class="text-action" data-toast="该预约已完成，无需重复操作">查看详情</button>'
}

function header(title) {
  return `<header><span>工作台　/　<strong>${title}</strong></span><span class="header-tools"><span>2026 年 7 月 16 日</span><span class="data-source ${dataSource === 'API 数据' ? 'remote' : ''}">● ${isSyncing ? '同步中' : dataSource}</span><button class="refresh" data-refresh ${isSyncing ? 'disabled' : ''}>↻ 刷新</button></span></header>`
}

function render() {
  const title = nav.find((item) => item[0] === page)?.[1] || '运营总览'
  const content = page === 'overview' ? overview() : page === 'queue' ? queue() : page === 'doctors' ? doctors() : page === 'patients' ? patients() : page === 'followups' ? followups() : page === 'services' ? servicesView() : page === 'bookings' ? bookingsView() : mobileView()
  document.querySelector('#app').innerHTML = `<div class="shell"><aside><div class="brand"><span>◷</span><div><strong>BookingFlow</strong><small>预约排班运营中心</small></div></div><div class="clinic">● 上海静安联合预约中心　⌄</div><p class="caption">临床运营</p><nav>${nav.map((item) => `<button class="${page === item[0] ? 'active' : ''}" data-page="${item[0]}"><i>${item[2]}</i>${item[1]}${item[0] === 'queue' ? '<em>8</em>' : ''}</button>`).join('')}</nav><div class="user"><b>许</b><span><strong>许汝林</strong><small>运营管理员</small></span></div></aside><main>${header(title)}<section class="heading"><div><p>THURSDAY, JUL 16 · BOOKINGFLOW</p><h1>${title} <i>✦</i></h1><label>让预约、排班与到店服务，在一个节奏里完成。</label></div><button class="primary" data-action="create-appointment">＋ 新建预约</button></section>${content}<footer>BookingFlow 预约排班运营 · 免费开源 · 演示数据不含诊断与真实客户信息</footer><div class="toast" ${toast ? '' : 'hidden'}>${toast}</div></main></div>`
  const root = document.querySelector('#app')
  displayCopy(root)
  bind()
}

function overview() {
  return `<section class="metrics"><article class="metric dark"><span>今日预约</span><strong>${dashboard.todayAppointments}</strong><small>↗ 较昨日 +14.6%</small></article><article class="metric"><span>平均候诊</span><strong>${dashboard.averageWaitMinutes}<small> 分钟</small></strong><small class="good">较上周 -3 分钟</small></article><article class="metric"><span>今日完成</span><strong>${dashboard.completed}<small> 人次</small></strong><div class="progress"><i style="width:68%"></i></div></article><article class="metric warm"><span>待回访</span><strong>${dashboard.pendingFollowups}<small> 条</small></strong><small class="coral">今日需完成</small></article></section><section class="grid"><article class="panel calendar"><div class="panel-head"><div><h2>今日预约队列</h2><p>7 月 16 日 · 周四 · 共 ${dashboard.todayAppointments} 位客户</p></div><button class="link" data-page="queue">查看队列 →</button></div><div class="timeline">${appointments.slice(0, 4).map((appointment) => `<div class="time-row"><span>${timeLabel(appointment.scheduledAt)}</span><i class="time-dot ${statusColors[appointment.status] || 'indigo'}"></i><div><strong>${appointment.patient}</strong><small>${appointment.department} · ${appointment.status}</small></div><b class="status ${statusColors[appointment.status] || 'indigo'}">${appointment.status}</b></div>`).join('')}</div></article><article class="panel"><div class="panel-head"><div><h2>科室处理负载</h2><p>当前时段排班利用率</p></div><button class="link" data-page="doctors">排班管理 →</button></div><div class="load-list">${[['全科门诊', '32 / 40', '80%', 'indigo'], ['皮肤科', '18 / 24', '75%', 'coral'], ['康复理疗', '12 / 18', '67%', 'green'], ['营养咨询', '8 / 12', '66%', 'amber']].map((item) => `<div class="load"><div><strong>${item[0]}</strong><span>${item[1]}</span></div><div class="load-bar"><i class="${item[3]}" style="width:${item[2]}"></i></div><b>${item[2]}</b></div>`).join('')}</div></article></section><section class="grid lower"><article class="panel"><div class="panel-head"><div><h2>回访完成趋势</h2><p>近 7 日任务完成率</p></div><span class="legend">本周平均 84%</span></div><div class="spark"><i style="height:38%"></i><i style="height:58%"></i><i style="height:46%"></i><i style="height:74%"></i><i style="height:66%"></i><i style="height:88%"></i><i class="today" style="height:80%"></i></div><div class="days"><span>周五</span><span>周六</span><span>周日</span><span>周一</span><span>周二</span><span>周三</span><span>今天</span></div></article><article class="panel tasks"><div class="panel-head"><div><h2>待办提醒</h2><p>需要运营人员跟进的事项</p></div></div><div class="task"><span class="task-icon coral">!</span><div><strong>3 位客户需要改约</strong><small>预约队列 · 10 分钟前</small></div><button data-page="queue">处理</button></div><div class="task"><span class="task-icon amber">✓</span><div><strong>${dashboard.pendingFollowups} 条回访今日到期</strong><small>健康回访 · 32 分钟前</small></div><button data-page="followups">查看</button></div></article></section>`
}

function queue() {
  return `<section class="panel full"><div class="panel-head"><div><h2>预约队列</h2><p>${dataSource === 'API 数据' ? 'API 实时预约' : '20 条演示预约'} · 支持确认、候诊、处理和完成</p></div><span class="chip">今天　⌄</span></div><div class="table"><div class="th"><span>预约编号 / 客户</span><span>科室</span><span>时间</span><span>状态</span><span>操作</span></div>${appointments.concat(dataSource === 'API 数据' ? [] : appointments.slice(0, 3)).map((appointment) => `<div class="tr"><span><strong>${appointment.id}</strong><small>${appointment.patient}</small></span><span>${appointment.department}</span><span>${timeLabel(appointment.scheduledAt)}</span><b class="status ${statusColors[appointment.status] || 'indigo'}">${appointment.status}</b><span>${appointmentAction(appointment)}</span></div>`).join('')}</div></section>`
}

function doctors() {
  return `<section class="panel full"><div class="panel-head"><div><h2>负责人排班</h2><p>8 位负责人 · 今日 42 个可预约时段</p></div><button class="primary small" data-toast="排班编辑器已打开">编辑排班</button></div><div class="doctor-grid">${[['林负责人', '全科门诊', '32 号候诊', 'indigo'], ['沈负责人', '皮肤科', '18 号候诊', 'coral'], ['赵负责人', '康复理疗', '处理中', 'green'], ['周负责人', '营养咨询', '8 号候诊', 'amber'], ['陈负责人', '全科门诊', '午间休息', 'gray'], ['王负责人', '心理咨询', '6 号候诊', 'indigo']].map((doctor) => `<article><div class="doctor-avatar ${doctor[3]}">${doctor[0][0]}</div><div><strong>${doctor[0]}</strong><small>${doctor[1]}</small></div><span>${doctor[2]}</span><div class="schedule-line"><i style="width:78%"></i></div></article>`).join('')}</div></section>`
}

function patients() {
  return `<section class="panel full"><div class="panel-head"><div><h2>客户档案</h2><p>30 条虚构档案 · 仅用于界面演示</p></div><button class="link" data-toast="导出任务已创建">导出列表 ↓</button></div><div class="table"><div class="th"><span>客户 / 编号</span><span>最近科室</span><span>最近就诊</span><span>回访状态</span><span>操作</span></div>${[['林晓雨', 'CF-2038', '全科门诊', '07/16', '待回访'], ['沈明远', 'CF-2037', '皮肤科', '07/15', '进行中'], ['赵思涵', 'CF-2036', '康复理疗', '07/14', '已完成'], ['周子昂', 'CF-2035', '全科门诊', '07/13', '待回访'], ['许安然', 'CF-2034', '营养咨询', '07/12', '已完成']].map((patient) => `<div class="tr"><span><strong>${patient[0]}</strong><small>${patient[1]}</small></span><span>${patient[2]}</span><span>${patient[3]}</span><b class="status ${patient[4] === '已完成' ? 'green' : 'coral'}">${patient[4]}</b><button class="text-action" data-toast="${patient[0]} 档案已打开">查看档案</button></div>`).join('')}</div></section>`
}

function followups() {
  return `<section class="panel full"><div class="panel-head"><div><h2>回访任务</h2><p>${dataSource === 'API 数据' ? 'API 实时回访' : '12 条待跟进任务'} · 由负责人/护士确认后记录</p></div><span class="chip">全部任务　⌄</span></div><div class="follow-list">${followupTasks.map((item) => `<article><span class="task-icon ${item.status === '已完成' ? 'green' : 'coral'}">✓</span><div><strong>${item.id} · ${item.patient}</strong><p>${item.summary}</p><small>${item.dueAt} · ${dataSource === 'API 数据' ? 'API 数据' : '演示任务'}</small></div>${item.status === '已完成' ? '<button class="text-action" data-toast="该回访已经完成">查看</button>' : `<button class="text-action" data-action="complete-followup" data-followup-id="${item.id}">完成任务</button>`}</article>`).join('')}</div></section>`
}

function money(cents) {
  return `¥${(Number(cents || 0) / 100).toFixed(0)}`
}

function bookingStatusAction(booking) {
  const next = { 待确认: '已预约', 已预约: '已签到', 已签到: '服务中', 服务中: '已完成' }[booking.status]
  const advance = next ? `<button class="text-action" data-action="booking-status" data-booking-id="${booking.id}" data-next-status="${next}">${next === '已预约' ? '确认预约' : next === '已签到' ? '确认到店' : next === '服务中' ? '开始服务' : '完成服务'}</button>` : ''
  const cancel = ['待确认', '已预约', '已签到'].includes(booking.status) ? `<button class="text-action coral" data-action="booking-status" data-booking-id="${booking.id}" data-next-status="已取消">取消</button>` : ''
  return advance + cancel
}

function servicesView() {
  return `<section class="panel full"><div class="panel-head"><div><h2>服务目录</h2><p>${services.length} 项可预约服务 · 先选服务，再锁定时段</p></div><span class="chip">家政保洁　⌄</span></div><div class="service-grid">${services.length ? services.map((service) => `<article class="service-card"><span class="service-card__tag">${service.category}</span><h3>${service.name}</h3><p>${service.description}</p><div class="service-card__meta"><strong>${money(service.priceCents)}</strong><span>${service.durationMinutes} 分钟</span></div><div class="slot-list">${slots.filter((slot) => slot.serviceId === service.id).map((slot) => `<button class="slot" data-action="create-booking" data-service-id="${service.id}" data-slot-id="${slot.id}" data-starts-at="${slot.startsAt}" data-ends-at="${slot.endsAt}" ${slot.remaining < 1 ? 'disabled' : ''}>${timeLabel(slot.startsAt)} · ${slot.remaining > 0 ? `余 ${slot.remaining} 个` : '已满'}</button>`).join('') || '<span class="empty-state">暂无可预约时段</span>'}</div></article>`).join('') : '<div class="empty-state">暂无服务，请稍后刷新</div>'}</div></section>`
}

function bookingsView() {
  if (selectedBooking) {
    const events = bookingEvents.length ? bookingEvents : [{ eventType: 'created', toStatus: BookingStatusLabel(selectedBooking.status), actor: '客户', createdAt: selectedBooking.createdAt || '2026-07-20T08:00:00Z' }]
    return `<section class="grid"><article class="panel"><div class="panel-head"><div><h2>订单 ${selectedBooking.id}</h2><p>${selectedBooking.customerName} · ${selectedBooking.serviceName}</p></div><button class="link" data-action="close-booking-detail">返回订单列表</button></div><div class="booking-summary"><b class="status ${statusColors[selectedBooking.status] || 'indigo'}">${selectedBooking.status}</b><strong>${money(selectedBooking.amountCents)}</strong><span>${timeLabel(selectedBooking.startsAt)}—${timeLabel(selectedBooking.endsAt)} · ${selectedBooking.paymentStatus || '待支付'}</span></div><div class="booking-actions">${bookingStatusAction(selectedBooking)}<button class="text-action" data-action="reschedule-booking" data-booking-id="${selectedBooking.id}">改约时段</button><button class="text-action coral" data-action="refund-booking" data-booking-id="${selectedBooking.id}">申请退款</button><button class="text-action" data-action="review-booking" data-booking-id="${selectedBooking.id}">记录评价</button></div></article><article class="panel"><div class="panel-head"><div><h2>服务时间线</h2><p>每次状态变化均保留操作记录</p></div></div><div class="booking-timeline">${events.map((event) => `<div><i></i><span><strong>${event.eventType === 'created' ? '创建订单' : event.eventType === 'reviewed' ? '完成评价' : event.toStatus || event.eventType}</strong><small>${event.actor || '系统'} · ${String(event.createdAt || '').replace('T', ' ').slice(0, 16)}</small></span></div>`).join('')}</div></article></section>`
  }
  return `<section class="panel full"><div class="panel-head"><div><h2>预约订单</h2><p>${bookings.length} 条订单 · 支持筛选、改约、退款与评价</p></div><div><button class="chip" data-toast="订单筛选：全部状态">全部状态　⌄</button><button class="primary small" data-page="services">新建预约</button></div></div>${bookings.length ? `<div class="table booking-table"><div class="th"><span>订单 / 客户</span><span>服务</span><span>时间</span><span>状态</span><span>操作</span></div>${bookings.map((booking) => `<div class="tr"><span><strong>${booking.id}</strong><small>${booking.customerName}</small></span><span>${booking.serviceName}</span><span>${timeLabel(booking.startsAt)}</span><b class="status ${statusColors[booking.status] || 'indigo'}">${booking.status}</b><span><button class="text-action" data-action="view-booking" data-booking-id="${booking.id}">查看</button>${bookingStatusAction(booking)}</span></div>`).join('')}</div>` : '<div class="empty-state">暂无订单，先去服务目录创建第一笔预约</div>'}</section>`
}

function BookingStatusLabel(status) { return status || '待确认' }

function mobileView() {
  return `<section class="mobile-panel"><div class="mobile-panel__hero"><span>BOOKINGFLOW MOBILE</span><h2>我的就诊与回访</h2><p>客户端可在同一套闭环 API 中完成确认、候诊、处理和回访确认。</p><button class="primary" data-action="create-appointment">＋ 创建演示预约</button></div><div class="mobile-list"><h3>今日预约</h3>${appointments.slice(0, 4).map((appointment) => `<article class="mobile-card"><div><small>${timeLabel(appointment.scheduledAt)} · ${appointment.department}</small><strong>${appointment.patient}</strong><span>${appointment.doctor} · ${appointment.status}</span></div><b class="status ${statusColors[appointment.status] || 'indigo'}">${appointment.status}</b>${appointmentAction(appointment)}</article>`).join('')}</div><div class="mobile-list"><h3>我的回访</h3>${followupTasks.slice(0, 3).map((item) => `<article class="mobile-card"><div><small>${item.dueAt}</small><strong>${item.summary}</strong><span>${item.patient} · ${item.status}</span></div>${item.status === '已完成' ? '<b class="status green">已完成</b>' : `<button class="text-action" data-action="complete-followup" data-followup-id="${item.id}">完成回访</button>`}</article>`).join('')}</div></section>`
}

async function refreshFromApi({ quiet = false } = {}) {
  if (isSyncing) return
  isSyncing = true
  render()
  try {
    const [nextDashboard, nextAppointments, nextFollowups] = await Promise.all([
      api.getDashboard(),
      api.listAppointments({ page: 1, pageSize: 20 }),
      api.listFollowups({ page: 1, pageSize: 20 }),
    ])
    dashboard = { ...demoDashboard, ...nextDashboard }
    appointments = (nextAppointments?.list || []).map(normalizeAppointment)
    followupTasks = (nextFollowups?.list || []).map(normalizeFollowup)
    const [serviceResult, slotResult, bookingResult] = await Promise.allSettled([
      api.listServices({ active: true }),
      api.listAvailability({ date: '2026-07-20' }),
      api.listBookings({ page: 1, pageSize: 50 }),
    ])
    if (serviceResult.status === 'fulfilled' && serviceResult.value?.list?.length) services = serviceResult.value.list
    if (slotResult.status === 'fulfilled' && slotResult.value?.list?.length) slots = slotResult.value.list
    if (bookingResult.status === 'fulfilled' && bookingResult.value?.list?.length) bookings = bookingResult.value.list
    dataSource = 'API 数据'
    if (!quiet) toast = '已从 BookingFlow API 刷新数据'
  } catch (error) {
    dataSource = '演示数据'
    if (!quiet) toast = `API 暂不可用，继续使用演示数据：${error.message}`
  } finally {
    isSyncing = false
    render()
  }
}

async function createBooking(button) {
  const service = services.find((item) => item.id === button.dataset.serviceId)
  if (!service) return
  const input = {
    serviceId: service.id,
    slotId: button.dataset.slotId,
    customerId: 'CUS-DEMO-001',
    customerName: '杭州星河家庭',
    startsAt: button.dataset.startsAt,
    endsAt: button.dataset.endsAt,
  }
  try {
    const created = await api.createBooking(input)
    bookings = [created, ...bookings]
    dataSource = 'API 数据'
    showToast('预约已创建，时段已锁定')
  } catch (error) {
    const fallback = { ...input, id: `BK-DEMO-${Date.now()}`, serviceName: service.name, amountCents: service.priceCents, status: '待确认', paymentStatus: '待支付' }
    bookings = [fallback, ...bookings]
    dataSource = '演示数据'
    showToast(`接口暂不可用，已保留演示订单：${error.message}`)
  }
}

async function openBooking(button) {
  const id = button.dataset.bookingId
  selectedBooking = bookings.find((item) => item.id === id) || null
  bookingEvents = []
  page = 'bookings'
  render()
  try {
    const [detail, events] = await Promise.all([api.getBooking(id), api.listBookingEvents(id)])
    selectedBooking = detail
    bookingEvents = events?.list || []
    dataSource = 'API 数据'
    render()
  } catch {
    // Demo orders intentionally keep the local timeline so the UI remains usable offline.
  }
}

async function updateBookingStatus(button) {
  const id = button.dataset.bookingId
  const booking = bookings.find((item) => item.id === id)
  if (!booking) return
  try {
    const updated = await api.updateBookingStatus(id, button.dataset.nextStatus, '运营人员')
    bookings = bookings.map((item) => item.id === id ? updated : item)
    selectedBooking = selectedBooking?.id === id ? updated : selectedBooking
    dataSource = 'API 数据'
    showToast(`${booking.customerName} 已更新为${updated.status}`)
  } catch (error) {
    showToast(`状态暂不可更新：${error.message}`)
  }
}

async function rescheduleBooking(button) {
  const booking = bookings.find((item) => item.id === button.dataset.bookingId)
  const nextSlot = slots.find((slot) => slot.serviceId === booking?.serviceId && slot.id !== booking.slotId && slot.remaining > 0)
  if (!booking || !nextSlot) return showToast('暂无可改约时段')
  try {
    const updated = await api.rescheduleBooking(booking.id, { slotId: nextSlot.id, startsAt: nextSlot.startsAt, endsAt: nextSlot.endsAt })
    bookings = bookings.map((item) => item.id === updated.id ? updated : item)
    selectedBooking = updated
    showToast('已改约至新的服务时段')
  } catch (error) {
    showToast(`改约失败：${error.message}`)
  }
}

async function refundBooking(button) {
  try {
    const updated = await api.refundBooking(button.dataset.bookingId)
    bookings = bookings.map((item) => item.id === updated.id ? updated : item)
    selectedBooking = updated
    showToast('退款申请已提交')
  } catch (error) {
    showToast(`退款失败：${error.message}`)
  }
}

async function reviewBooking(button) {
  try {
    await api.createBookingReview(button.dataset.bookingId, { rating: 5, content: '服务准时，沟通顺畅' })
    showToast('评价已记录')
  } catch (error) {
    showToast(`评价失败：${error.message}`)
  }
}

function replaceAppointment(updated) {
  appointments = appointments.map((item) => item.id === updated.id ? normalizeAppointment(updated) : item)
}

async function advanceAppointment(button) {
  const id = button.dataset.appointmentId
  const appointment = appointments.find((item) => item.id === id)
  if (!appointment) return
  const nextStatus = button.dataset.nextStatus
  try {
    const updated = button.dataset.action === 'checkin'
      ? await api.checkinAppointment(id)
      : await api.updateAppointmentStatus(id, nextStatus, '运营人员')
    replaceAppointment(updated)
    dataSource = 'API 数据'
    showToast(`${appointment.patient} 已更新为${updated.status}`)
  } catch (error) {
    dataSource = '演示数据'
    showToast(`接口暂不可用，已保留演示数据：${error.message}`)
  }
}

async function completeFollowup(button) {
  const id = button.dataset.followupId
  const task = followupTasks.find((item) => item.id === id)
  if (!task) return
  try {
    const updated = await api.completeFollowup(id)
    followupTasks = followupTasks.map((item) => item.id === id ? normalizeFollowup(updated) : item)
    dataSource = 'API 数据'
    showToast(`${task.patient} 的回访已完成`)
  } catch (error) {
    dataSource = '演示数据'
    showToast(`接口暂不可用，已保留演示任务：${error.message}`)
  }
}

async function createAppointment() {
  try {
    const created = await api.createAppointment({ patient: '移动端演示客户', patientId: 'CUS-MOBILE-DEMO', department: '深度保洁', doctor: '林然 · 服务顾问', scheduledAt: new Date().toISOString() })
    appointments = [normalizeAppointment(created), ...appointments]
    dataSource = 'API 数据'
    showToast('预约已创建，可继续在移动端完成确认')
  } catch (error) {
    dataSource = '演示数据'
    showToast(`API 暂不可用，保留演示预约：${error.message}`)
  }
}

function bind() {
  document.querySelectorAll('[data-page]').forEach((element) => element.addEventListener('click', () => {
    page = element.dataset.page
    render()
  }))
  document.querySelectorAll('[data-toast]').forEach((element) => element.addEventListener('click', () => showToast(element.dataset.toast)))
  document.querySelectorAll('[data-refresh]').forEach((element) => element.addEventListener('click', () => refreshFromApi()))
  document.querySelectorAll('[data-action]').forEach((element) => element.addEventListener('click', () => {
    if (element.dataset.action === 'checkin' || element.dataset.action === 'status') return advanceAppointment(element)
    if (element.dataset.action === 'complete-followup') return completeFollowup(element)
    if (element.dataset.action === 'create-appointment') return createAppointment()
    if (element.dataset.action === 'create-booking') return createBooking(element)
    if (element.dataset.action === 'view-booking') return openBooking(element)
    if (element.dataset.action === 'close-booking-detail') { selectedBooking = null; bookingEvents = []; return render() }
    if (element.dataset.action === 'booking-status') return updateBookingStatus(element)
    if (element.dataset.action === 'reschedule-booking') return rescheduleBooking(element)
    if (element.dataset.action === 'refund-booking') return refundBooking(element)
    if (element.dataset.action === 'review-booking') return reviewBooking(element)
    return undefined
  }))
}

render()
refreshFromApi({ quiet: true })
