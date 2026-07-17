import test from 'node:test'
import assert from 'node:assert/strict'

import { createApiClient } from '../src/api.js'

function response(data) {
  return { ok: true, status: 200, async json() { return { code: 0, message: 'ok', data } } }
}

test('预约闭环 API 暴露目录、时段、订单时间线与售后操作', async () => {
  const requests = []
  const client = createApiClient({
    fetchImpl: async (url, init = {}) => {
      requests.push({ url, init })
      return response({ id: 'BK-1', list: [], total: 0 })
    },
  })

  await client.listServices({ category: '家政保洁' })
  await client.listAvailability({ serviceId: 'svc-cleaning', date: '2026-07-20' })
  await client.listBookings({ page: 1, status: '已预约' })
  await client.getBooking('BK-1')
  await client.listBookingEvents('BK-1')
  await client.createBooking({ serviceId: 'svc-cleaning', slotId: 'slot-cleaning-0900', customerName: '杭州星河家庭', startsAt: '2026-07-20T09:00:00+08:00', endsAt: '2026-07-20T11:00:00+08:00' }, 'booking-1')
  await client.updateBookingStatus('BK-1', '已签到', '客户', 'status-1')
  await client.rescheduleBooking('BK-1', { slotId: 'slot-cleaning-1300', startsAt: '2026-07-20T13:00:00+08:00', endsAt: '2026-07-20T15:00:00+08:00' }, 'reschedule-1')
  await client.refundBooking('BK-1', 'refund-1')
  await client.createBookingReview('BK-1', { rating: 5, content: '服务准时' }, 'review-1')

  assert.deepEqual(requests.map(({ url }) => url), [
    '/api/v1/services?category=%E5%AE%B6%E6%94%BF%E4%BF%9D%E6%B4%81',
    '/api/v1/availability?serviceId=svc-cleaning&date=2026-07-20',
    '/api/v1/bookings?page=1&status=%E5%B7%B2%E9%A2%84%E7%BA%A6',
    '/api/v1/bookings/BK-1',
    '/api/v1/bookings/BK-1/events',
    '/api/v1/bookings',
    '/api/v1/bookings/BK-1/status',
    '/api/v1/bookings/BK-1/reschedule',
    '/api/v1/bookings/BK-1/refund',
    '/api/v1/bookings/BK-1/review',
  ])
  assert.equal(requests[5].init.headers['Idempotency-Key'], 'booking-1')
  assert.equal(requests[6].init.headers['Idempotency-Key'], 'status-1')
})
