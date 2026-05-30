

<template>
  <div class="demo-type">
     <el-dropdown>
    <span class="el-dropdown-link">
     <div class="avatar">
      <el-avatar
        src="https://cube.elemecdn.com/0/88/03b0d39583f48206768a7534e55bcpng.png"
      />
    </div>
    
    </span>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item >个人中心</el-dropdown-item>
        <el-dropdown-item @click="handleLogout">退出登录</el-dropdown-item>
        
      </el-dropdown-menu>
    </template>
  </el-dropdown>
    
   
  </div>
</template>

<script setup lang="ts" name="Avatar">
import {useAuthStore} from '../stores/auth'
const authStore = useAuthStore()
const handleLogout = () => {
  authStore.logout()
}

</script>

<style scoped>
.demo-type {
  display: flex;
}
.demo-type > div {
  flex: 1;
  text-align: center;
}

.demo-type > div:not(:last-child) {
  border-right: 1px solid var(--el-border-color);
}
</style>