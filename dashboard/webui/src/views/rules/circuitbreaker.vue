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
        <el-table-column prop="statIntervalMs" label="统计窗口/毫秒" width="200" />
        <el-table-column prop="retryTimeoutMs" label="熔断时长/毫秒" width="200" />
        <el-table-column prop="threshold" label="阈值" width="100" />
        <el-table-column prop="strategy" label="熔断策略" width="250">
          <template slot-scope="props">
            {{ strategyMap[props.row.strategy] }}
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="100">
          <template slot="header">
            <el-button type="text" size="small" @click="updateAppFlowRule(null)">
              <i class="el-icon-plus" />新建
            </el-button>
          </template>
          <template slot-scope="props">
            <el-button type="text" size="mini" @click="updateAppFlowRule(props.row)">修改</el-button>
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
        <el-form-item label="熔断策略">
          {{ strategyMap[dialogForm.strategy] }}
        </el-form-item>
        <el-form-item
          v-if="!dialogForm.id"
          label="资源名"
          prop="resource"
          :rules="[{ required: true, message: '请填写资源名' }]"
        >
          <el-input v-model="dialogForm.resource" />
        </el-form-item>

        <el-form-item label="统计窗口/毫秒" prop="statIntervalMs">
          <el-input-number v-model="dialogForm.statIntervalMs" :min="100" />
        </el-form-item>
        <el-form-item label="统计窗口/毫秒" prop="retryTimeoutMs">
          <el-input-number v-model="dialogForm.retryTimeoutMs" :min="100" />
        </el-form-item>
        <el-form-item label="阈值" prop="threshold">
          <el-input-number v-model="dialogForm.threshold" :min="0" />
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

import { listCircuitbreakerRules, deleteCircuitbreakerRule, saveOrUpdateCircuitbreakerRule } from '@/api/env'

export default {
  name: 'CircuitbreakerRule',
  data() {
    return {
      rules: [],
      loading: false,
      dialogVisible: false,
      dialogTitle: '',
      dialogForm: {},
      delLoading: false,
      updLoading: false,
      strategyMap: {
        0: '慢调用比例(SlowRequestRatio)',
        1: '错误比例(ErrorRatio)',
        2: '错误数量(ErrorCount)'
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
      this.$confirm('此操作将删除熔断降级规则, 是否继续?', '提示', {
        confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning'
      }).then(() => {
        that.delLoading = true
        deleteCircuitbreakerRule(payload).then(resp => {
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
      const tip = payload.id ? '更新熔断降级规则' : '创建熔断降级规则'
      this.$confirm(`此操作将${tip}, 是否继续?`, '提示', {
        confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning'
      }).then(() => {
        that.updLoading = true
        saveOrUpdateCircuitbreakerRule(payload).then(resp => {
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
          'strategy': 1,
          'retryTimeoutMs': 3000,
          'minRequestAmount': 10,
          'statIntervalMs': 10000,
          'statSlidingWindowBucketCount': 0,
          'maxAllowedRtMs': 0,
          'threshold': 0.3,
          'probeNum': 0
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
      listCircuitbreakerRules(that.appId).then(resp => {
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

