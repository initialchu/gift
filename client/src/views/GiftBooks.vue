<template>
  <div class="giftbooks">
    <el-header class="gift-header">
      
      <el-button type="primary" @click="dialogVisible=true">新建礼薄</el-button>
    </el-header>

    <el-dialog v-model="dialogVisible" title="新建礼薄" width="50%">
      <el-form :model="form">
        <el-form-item class="label" label="礼薄事件">

          <el-input v-model="form.event_name" placeholder="请输入礼薄事件描述"></el-input>
          </el-form-item>
          <el-form-item class="label" label="礼薄方向">
          <div class="mb-2 ml-4">
            <el-radio-group v-model="form.direction">
            <el-radio value="来" size="large">来</el-radio>
            <el-radio value="去" size="large">去</el-radio>
              
            </el-radio-group>
            </div>
        </el-form-item>
          <el-form-item label="日期">
        <el-date-picker v-model="form.event_date" type="date" />
      </el-form-item>
        <el-button type="primary" @click="createGiftBook">新建</el-button>
      </el-form>
      </el-dialog>
     <!-- 卡片网格 -->
    <div class="books">
    <Books ref="booksRef" />
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
  direction:'来',
})
// 子组件 Books 的 ref，用于创建成功后刷新列表
const booksRef = ref<InstanceType<typeof Books>>()

// 这里是新增礼薄的函数
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