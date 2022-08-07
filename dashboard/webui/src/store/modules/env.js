import { listApps } from '@/api/env'

const state = {
  currentApp: {},
  allApp: []
}

const mutations = {
  SET_ENV: (state, app) => {
    state.currentApp = { ...app }
  },
  SET_ALL: (state, apps) => {
    state.allApp = [...apps]
  }
}

const actions = {
  reLoadApps({ commit, state }) {
    return new Promise((resolve, reject) => {
      listApps().then(response => {
        const { data } = response
        commit('SET_ALL', data)

        // set current app
        let currentApp = state.currentApp
        let isValid = false

        if (currentApp && data) {
          data.forEach(element => {
            if (element.id === currentApp.id) {
              isValid = true
            }
          })
        }

        if ((!isValid || !currentApp) && (data && data.length > 0)) {
          currentApp = data[0]
          commit('SET_ENV', currentApp)
        }

        resolve()
      }).catch(error => {
        reject(error)
      })
    })
  },

  async changeAppEnv({ commit, dispatch }, app) {
    commit('SET_ENV', app)
  }
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
