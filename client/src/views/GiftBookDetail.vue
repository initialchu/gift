<template>
  <div class="giftbook-detail">
    <el-button text @click="router.push('/giftbooks')">&larr; 返回礼薄列表</el-button>

    <el-card v-loading="loading">
      <template #header>
        <div class="detail-header">
          <h2>{{ book?.event_name }}</h2>
          <el-tag :type="book?.direction === '来' ? 'success' : 'danger'">
            {{ book?.direction }}
          </el-tag>
        </div>
      </template>

      <div v-if="book" class="detail-info">
        <p>日期：{{ formatDate(book.event_date) }}</p>
        <p>创建者：{{ book.created_by }}</p>
      </div>

      <el-empty v-if="!loading && !book" description="礼薄不存在" />
    </el-card>

    <!-- 记录列表区域（后续完善） -->
    <div class="records-section">
      <h3>往来记录</h3>
      <el-empty description="记录列表待实现" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import axios from '../axios'

const router = useRouter()
const route = useRoute()
const loading = ref(false)
const book = ref<any>(null)

const formatDate = (iso: string): string => {
  if (!iso) return ''
  return String(iso.split('T')[0])
}

const fetchDetail = async () => {
  loading.value = true
  try {
    const id = route.params.id
    const res = await axios.get(`/giftbook/${id}`)
    book.value = res.data.giftbook
  } catch (err: any) {
    const msg = err.response?.data?.error || '获取礼薄详情失败'
    ElMessage.error(msg)
  } finally {
    loading.value = false
  }
}

import { ElMessage } from 'element-plus'

onMounted(() => {
  fetchDetail()
})
</script>

<style scoped>
.giftbook-detail {
  padding: 20px;
  max-width: 800px;
  margin: 0 auto;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.detail-info {
  color: #666;
  font-size: 14px;
}

.detail-info p {
  margin: 8px 0;
}

.records-section {
  margin-top: 24px;
}
</style>
