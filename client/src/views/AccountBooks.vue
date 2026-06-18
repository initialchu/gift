<template>
  <div class="accountbooks">
    <el-header class="gift-header">
      
      <el-button type="primary" @click="dialogVisible=true">新建账本</el-button>
    </el-header>

    <el-dialog v-model="dialogVisible" title="新建账本" width="50%">
      <el-form :model="form">
        <el-form-item class="label" label="账本事件">

          <el-input v-model="form.event_name" placeholder="请输入账本事件描述"></el-input>
          </el-form-item>
        
          <el-form-item label="日期">
        <el-date-picker v-model="form.event_date" type="date" />
      </el-form-item>
        <el-button type="primary" @click="createGiftBook">新建</el-button>
      </el-form>
      </el-dialog>
     <!-- 卡片网格 -->
    <div class="books">
    <Books ref="booksRef" direction="去"/>
      </div>
  </div>
</template>

<script lang="ts" setup>

import {ref} from 'vue'
import axios from '../axios'
import Books from '../components/Books.vue'
import { ElMessage } from 'element-plus'
const dialogVisible = ref(false)
const form = ref({
 event_name: '',
 event_date:'',
  direction:'去',
})
// 子组件 Books 的 ref，用于创建成功后刷新列表
const booksRef = ref<InstanceType<typeof Books>>()

// 这里是新增账本的函数
const createGiftBook = async () => {
  try {
    await axios.post('/admin/giftbook', form.value)
    ElMessage.success('创建成功')
    dialogVisible.value = false
    booksRef.value?.fetchGiftBooks()
  } catch (err: any) {
    const msg = err.response?.data?.error || '创建失败'
    ElMessage.error(msg)
  }
}

</script>
<style scoped>
.gift-header{
    display:flex;
    justify-content:space-between;
    align-items:center;
    height:60px;
}
.books{
    padding:20px;
    height:calc(100vh - 60px - 40px);
    overflow-y:auto;
    width:100%;
    
}
</style>