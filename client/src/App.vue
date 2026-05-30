<template>
  <el-menu
    :default-active="activeIndex"
    class="el-menu-demo"
    mode="horizontal"
    :ellipsis="false"
    @select="handleSelect"
  >
   
    <el-menu-item index="1">首页</el-menu-item>
    <el-menu-item index="2">人情卡片</el-menu-item>
    <el-menu-item index="3">礼薄</el-menu-item>
  </el-menu>
</template>

<script lang="ts" setup>
import { ref } from 'vue'

const activeIndex = ref('1')
const handleSelect = (key: string, keyPath: string[]) => {
  console.log(key, keyPath)
}
</script>

<style scoped>
/* .el-menu--horizontal > .el-menu-item:nth-child(1) {
  margin-right: auto;
} */
</style>
