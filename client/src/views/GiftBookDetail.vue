<template>
  <div class="giftbook-detail">
    <el-button text @click="router.back()">&larr; 返回</el-button>

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
     <div class="records-header">
        <h3>礼金记录</h3>
         <el-button type="primary" @click="dialogVisible=true">添加记录</el-button>
     </div>
     <el-dialog v-model="dialogVisible" title="添加记录">
        <el-form :model="form">
          <el-form-item label="姓名">
            

            <el-autocomplete 
                v-model="form.person_name"
                :fetch-suggestions="querySearch"
                clearable
                placeholder="请输入姓名"
                @select="handleSelect"
            >

            </el-autocomplete>
           
          </el-form-item>
          <el-form-item label="金额">
            <el-radio-group v-model="selectedAmount">
              <el-radio :value="100">100</el-radio>
              <el-radio :value="200">200</el-radio>
              <el-radio :value="500">500</el-radio>
              <el-radio :value="0">自定义</el-radio>
            </el-radio-group>
            <el-input
              v-if="selectedAmount === 0"
              v-model="customAmount"
              type="number"
              placeholder="请输入金额"
              style="margin-top: 8px; width: 200px ;margin-left: 8px"
            />
          </el-form-item>
          <el-form-item label="地址">
            <el-input v-model="form.address" />
          </el-form-item>
          <el-form-item label="备注">
            <el-input v-model="form.gift_note" />
          </el-form-item>
          <el-button type="primary" @click="submitRecord">提交</el-button>
        </el-form>
     </el-dialog>
    <div class="records-section">
      
      <GiftRecords ref="giftrecords"/>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import axios from '../axios'
import GiftRecords from '../components/GiftRecords.vue'
import { ElMessage } from 'element-plus'
const router = useRouter()
const route = useRoute()
const loading = ref(false)
const book = ref<any>(null)
const selectedAmount = ref(100)
const customAmount = ref<number | null>(null)
const formatDate = (iso: string): string => {
  if (!iso) return ''
  return String(iso.split('T')[0])
}
//搜索建议
interface RetItem{
  value:string
  card_id:number
}
const rets = ref<RetItem[]>([])
const querySearch = async (queryString:string,cb:any)=>{

  const results = queryString ? rets.value.filter(createFilter(queryString)) : rets.value
  cb(results)
  
}
const createFilter = (queryString:string)=>{
  return (ret:RetItem)=>{
    if(!ret.value) return false
    return ret.value.toLowerCase().indexOf(queryString.toLowerCase()) === 0
  }

}
const loadAll =async ()=>{
  try{
      const res = await axios.get(`/cards`)
      
      rets.value = res.data.cards .filter((card: any) => card.person_name).map((card:any)=>({
        value:card.person_name,
        card_id:card.card_id,
      }))
  }catch{
    ElMessage.error('获取卡片列表失败')
  }
  
}
const handleSelect = (item:RetItem)=>{
  form.value.card_id = item.card_id
}
//获取礼薄详情
const fetchDetail = async () => {
  loading.value = true
  try {
    const id = route.params.id
    const res = await axios.get(`/giftbook/${id}`)
    console.log('礼薄详情', res.data)
    book.value = res.data.giftbook
  } catch (err: any) {
    const msg = err.response?.data?.error || '获取礼薄详情失败'
    ElMessage.error(msg)
  } finally {
    loading.value = false
  }
}
//添加记录
const dialogVisible = ref(false)
type Record = {
  gift_book_id : number
  person_name: string
  amount: number
  address: string
  gift_note:string
  card_id?:number
}
const form = ref<Record>({
  gift_book_id:Number(route.params.id),
  person_name: '',
  amount: 0,
  address: '',
  gift_note: '',
  card_id: 0,
})
const submitRecord = async () => {
  try {
    form.value.amount = selectedAmount.value === 0 ? (customAmount.value || 0) : selectedAmount.value
    const res = await axios.post(`/admin/giftrecord/${form.value.gift_book_id}/records`, form.value)
    console.log('添加记录成功', res.data)
    ElMessage.success('记录添加成功')
    dialogVisible.value = false
    fetchRecords()
    loadAll()
    //重置表单
    form.value = {
      gift_book_id:Number(route.params.id),
      person_name: '',
      amount: 0,
      address: '',
      gift_note: '',
      card_id: 0,
    }
    selectedAmount.value = 100
    customAmount.value = null
  } catch (err: any) {
    const msg = err.response?.data?.error || '添加记录失败'
    ElMessage.error(msg)
  }
}
//调用GiftRecords组件中的fetchRecords方法来刷新记录列表
const giftrecords = ref<InstanceType<typeof GiftRecords>|null>(null)
const fetchRecords = ()=>{
  giftrecords.value?.fetchRecords()
}


onMounted(() => {
  fetchDetail()
  loadAll()
})
</script>

<style scoped>
.giftbook-detail {
  padding: 20px;
  max-width: 800px;
  margin: 0 auto;
}
.records-header
{
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 24px;
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
