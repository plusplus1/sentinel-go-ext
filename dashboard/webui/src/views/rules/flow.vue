<template>
  <div class="app-container" v-loading:="loading">
    <div v-if="rules && rules.length > 0">
      <el-table :data="rules" style="width: 100%">
        <!-- <el-table-column type="expand">
          <template slot-scope="props">
            <pre>{{ JSON.stringify(props.row, null, 4) }}</pre>
          </template>
        </el-table-column> -->
        <el-table-column prop="resource" label="资源名" />
        <el-table-column prop="statIntervalInMs" label="统计间隔/毫秒" width="200" />
        <el-table-column prop="threshold" label="阈值" width="100" />
        <el-table-column prop="tokenCalculateStrategy" label="计算策略" width="200">
          <template slot-scope="props">
            {{ tokenCalculateStrategyMap[props.row.tokenCalculateStrategy] }}
          </template>
        </el-table-column>
        <el-table-column prop="controlBehavior" label="控制策略" width="200">
          <template slot-scope="props">
            {{ controlBehaviorMap[props.row.controlBehavior] }}
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="100">

          <template slot="header">
            <el-button type="text" size="small" @click="updateAppFlowRule(null)">
              <i class="el-icon-plus" />新建
            </el-button>
          </template>

          <template slot-scope="props">
            <el-button type="text" size="mini" @click="updateAppFlowRule(props.row)">
              <i class="el-icon-edit" /> 修改
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
    <div v-else>
      <span>暂无规则</span>
      <el-button type="text" @click="updateAppFlowRule(null)">&nbsp;&nbsp;点我&nbsp;&nbsp;</el-button><span>添加</span>
    </div>

    <el-dialog :title="dialogTitle" :visible.sync="dialogVisible" width="40%">
      <el-form label-position="left" label-width="180px" :model="dialogForm">
        <el-form-item label="计算策略">
          <!-- <el-select v-model="dialogForm.tokenCalculateStrategy" readonly placeholder="请选择">
            <el-option v-for="(v, k) in tokenCalculateStrategyMap" :key="k" :label="v" :value="parseInt(k)" />
          </el-select> -->
          {{ tokenCalculateStrategyMap[dialogForm.tokenCalculateStrategy] }}
        </el-form-item>

        <el-form-item label="控制策略">
          <!-- <el-select v-model="dialogForm.controlBehavior" readonly placeholder="请选择">
            <el-option v-for="(v, k) in controlBehaviorMap" :key="k" :label="v" :value="parseInt(k)" />
          </el-select> -->
          {{ controlBehaviorMap[dialogForm.controlBehavior] }}
        </el-form-item>

        <el-form-item
          v-if="!dialogForm.id"
          label="资源名"
          prop="resource"
          :rules="[{ required: true, message: '请填写资源名' }]"
        >
          <el-input v-model="dialogForm.resource" />
        </el-form-item>
        <el-form-item label="统计窗口/毫秒" prop="statIntervalInMs">
          <el-input-number v-model="dialogForm.statIntervalInMs" :min="100" :max="3000" />
        </el-form-item>
        <el-form-item label="阈值" prop="threshold">
          <el-input-number v-model="dialogForm.threshold" :min="1" />
        </el-form-item>
      </el-form>
      <span slot="footer" class="dialog-footer" style="width:100%">
        <el-button v-if="dialogForm.id" :loading="delLoading" type="warning" @click="onDelete">删 除</el-button>
        <el-button @click="dialogVisible = false">取 消</el-button>
        <el-button type="primary" :loading="updLoading" @click="onSaveOrUpdate">更 新</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>

import { listFlowRules, deleteFlowRule, saveOrUpdateFlowRule } from '@/api/env'

export default {
  name: 'FlowRule',
  data() {
    return {
      rules: [],
      dialogVisible: false,
      dialogTitle: '',
      dialogForm: {},
      loading: false,
      delLoading: false,
      updLoading: false,
      tokenCalculateStrategyMap: {
        0: 'Direct',
        1: 'WarmUp',
        2: 'MemoryAdaptive'
      },
      controlBehaviorMap: {
        0: 'Reject',
        1: 'Throttling'
      }
    }
  },
  computed: {
    appId() {
      const app = this.$store.getters.currentApp
      return app && app.id
    }
  },
  mounted() {
    this.safeReload()
  },
  methods: {

    onDelete() {
      const that = this
      const payload = { ...that.dialogForm }
      this.$confirm('此操作将删除流量控制规则, 是否继续?', '提示', {
        confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning'
      }).then(() => {
        that.delLoading = true
        deleteFlowRule(payload).then(resp => {
          setTimeout(() => {
            if (resp && resp.code === 0) {
              that.$notify({ title: '提示', message: `删除成功`, type: 'success' })
              that.dialogVisible = false
              that.delLoading = false
              that.safeReload()
            } else {
              that.$notify({ title: '提示', message: `${resp && resp.message || '删除失败'}`, type: 'error' })
              that.delLoading = false
            }
          }, 500)
        }).catch(() => {
          that.delLoading = false
        })
      })
    },
    onSaveOrUpdate() {
      const payload = { ...this.dialogForm }
      const that = this
      const tip = payload.id ? '更新流量控制规则' : '创建流量控制规则'
      this.$confirm(`此操作将${tip}, 是否继续?`, '提示', {
        confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning'
      }).then(() => {
        that.updLoading = true
        saveOrUpdateFlowRule(payload).then(resp => {
          setTimeout(() => {
            if (resp && resp.code === 0) {
              that.$notify({ title: '提示', message: `${tip}成功！`, type: 'success' })
              that.dialogVisible = false
              that.safeReload()
            } else {
              that.$notify({ title: '提示', message: `${resp && resp.message || (tip + '失败')}`, type: 'error' })
            }
            that.updLoading = false
          }, 500)
        }).catch(() => {
          that.updLoading = false
        })
      })
    },

    updateAppFlowRule(app) {
      if (app) {
        this.dialogTitle = '修改规则：' + app.resource
        this.dialogForm = { appId: this.appId, ...app }
      } else {
        this.dialogTitle = '创建规则'
        this.dialogForm = {
          'appId': this.appId,
          'id': '',
          'resource': '',
          'tokenCalculateStrategy': 0,
          'controlBehavior': 0,
          'threshold': 0,
          'relationStrategy': 0,
          'refResource': '',
          'maxQueueingTimeMs': 0,
          'warmUpPeriodSec': 0,
          'warmUpColdFactor': 0,
          'statIntervalInMs': 1000,
          'lowMemUsageThreshold': 0,
          'highMemUsageThreshold': 0,
          'memLowWaterMarkBytes': 0,
          'memHighWaterMarkBytes': 0
        }
      }
      this.dialogVisible = true
      this.delLoading = false
      this.updLoading = false
    },
    sleep(time) {
      return new Promise(resolve => setTimeout(resolve, time))
    },
    safeReload() {
      const that = this
      if (!this.appId) {
        this.$store.dispatch('env/reLoadApps').then(() => { that.doLoad() })
      } else {
        that.doLoad()
      }
    },
    doLoad() {
      const that = this
      that.loading = true
      listFlowRules(that.appId).then(resp => {
        if (resp && resp.code === 0) {
          that.rules = resp.data || []
        }
        that.loading = false
      }).catch(() => { that.loading = false })
    }
  }
}

</script>

<style scoped>
</style>

