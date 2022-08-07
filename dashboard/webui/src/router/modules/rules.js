/** When your routing table is too long, you can split it into small modules**/

import Layout from '@/layout'

const rulesRouter = {
  path: '/rules',
  component: Layout,
  redirect: 'noRedirect',
  name: '规则',
  meta: { title: '规则配置', icon: 'component' },
  children: [
    {
      path: 'flow',
      component: () => import('@/views/rules/flow'),
      name: 'Flow',
      meta: { title: '流量控制', noCache: true, icon: 'el-icon-sunrise' }
    },
    {
      path: 'circuitbreaker',
      component: () => import('@/views/rules/circuitbreaker'),
      name: 'Circuitbreaker',
      meta: { title: '熔断降级', noCache: true, icon: 'el-icon-ship' }
    }
  ]
}

export default rulesRouter
