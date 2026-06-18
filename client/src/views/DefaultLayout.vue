<template>
  <div class="layout">
    <el-container>
      <el-header class="header">
        <el-menu
          :default-active="activeIndex"
          class="el-menu-demo"
          mode="horizontal"
          :ellipsis="false"
          @select="handleSelect"
        >
          <el-menu-item index="home">首页</el-menu-item>
          <el-menu-item index="card">人情卡片</el-menu-item>
          <el-menu-item index="giftbooks">礼薄</el-menu-item>
          <el-menu-item index="account">账本</el-menu-item>
          
        </el-menu>
        <div class="avatar">
          <Avatar />
        </div>
      </el-header>
      <el-main>
        <RouterView />
      </el-main>
      <el-footer>Copyright © 2026 Gift</el-footer>
    </el-container>
  </div>
</template>

<script lang="ts" setup>
import { ref, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import Avatar from '../components/Avatar.vue'
const router = useRouter()
const route = useRoute()

const activeIndex = ref((route.name as string) || 'home')

const handleSelect = (key: string) => {
  router.push({ name: key })
}

watch(
  () => route.name,
  (name) => {
    activeIndex.value = (name as string) || 'home'
  },
)
</script>

<style scoped>
.el-footer {
  text-align: center;
  color: #909399;
  padding: 16px 0;
}
.header {
 
  display: flex;
  align-items: center;
}
.header .el-menu {
  flex: 1;
 
  
}
.avatar{
  margin-right: 20px;
}
body {
  padding-right: 0 !important;
}
</style>
