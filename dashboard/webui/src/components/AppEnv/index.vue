<template>
  <div class="header-search">
    <span class="search-icon">

      <el-dropdown @command="handleCommand">
        <span class="el-dropdown-link">
          <el-link> <span>当前APP: </span>
            <el-tag>
              {{ currentApp.name }} （ {{ currentApp.env }} ）
            </el-tag>
          </el-link> <i class="el-icon-arrow-down el-icon--right" />
        </span>
        <el-dropdown-menu slot="dropdown">
          <el-dropdown-item v-for="app in allApp" :key="app.id" :command="app">
            {{ app.name }} （ {{ app.env }} ）
          </el-dropdown-item>
        </el-dropdown-menu>
      </el-dropdown>
    </span>
  </div>
</template>

<script>
// fuse is a lightweight fuzzy-search module
// make search results more in line with expectations

export default {
  name: 'AppEnv',
  data() {
    return {
      dialogVisible: false
    }
  },
  computed: {
    currentApp() {
      return this.$store.getters.currentApp
    },
    allApp() {
      return this.$store.getters.allApp
    }
  },
  // watch: {
  // },
  mounted() {
    this.reloadApps()
  },
  methods: {
    reloadApps() {
      this.$store.dispatch('env/reLoadApps')
    },
    handleCommand(value) {
      const currentId = this.currentApp.id
      if (currentId !== value.id) {
        this.$store.dispatch('env/changeAppEnv', value)
        this.dialogVisible = false
        this.$message({ message: `切换到APP: ${value.name}（${value.env}）`, type: 'success' })
      }
    }
  }
}
</script>

<style lang="scss" scoped>
.header-search {
  font-size: 0 !important;

  .search-icon {
    cursor: pointer;
    font-size: 18px;
    vertical-align: middle;
  }

  .header-search-select {
    font-size: 18px;
    transition: width 0.2s;
    width: 0;
    overflow: hidden;
    background: transparent;
    border-radius: 0;
    display: inline-block;
    vertical-align: middle;

    ::v-deep .el-input__inner {
      border-radius: 0;
      border: 0;
      padding-left: 0;
      padding-right: 0;
      box-shadow: none !important;
      border-bottom: 1px solid #d9d9d9;
      vertical-align: middle;
    }
  }

  &.show {
    .header-search-select {
      width: 210px;
      margin-left: 10px;
    }
  }
}
</style>
