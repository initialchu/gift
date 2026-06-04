<template>
  <div class="cards">
    <!-- grid视图-->
     <template v-if="view==='grid'">
       <div class="page-header" >
          <h2>人情卡片</h2>
          <el-input v-model="search" placeholder="搜索人名..." clearable></el-input>
          
       </div>
       <div  class="card-container" v-loading="loading">
          <el-card  @click="handledetail(card.card_id)" class="card" v-for="card in filterTableData" :key="card.card_id" >
            
            <template #header class="card-header">
             
             <h3>{{ card.person_name }}</h3>
            </template>
             <div class="card-content">
                  <span class="data">来：{{ card.received_count }} 次</span>
                
                 <span class="data">共计：{{ card.received_amount }} 元 </span>
                 <span class="data">去：{{ card.given_count }} 次</span>
                 <span class="data">共计：{{ card.given_amount }} 元 </span>
                 
                 <span :class="card.net_amount >= 0 ? 'positive' : 'negative'">

                     差值：{{ card.net_amount >= 0 ? '+' : '' }}{{ card.net_amount }}
                  </span>
              
              </div>
          </el-card>
       </div>
     </template>
     <!--详细视图-->
     <template v-if="view==='detail'">
        <div class="page-header">
          <el-button @click="handledetail(-1)">返回</el-button>
          <h2>详细信息</h2>
          
        </div>
        <div class="detail-content">
          
          <h2>{{ cards.find(card => card.card_id === id)?.person_name }}</h2>
          <hr>
          <el-table :data="cardDetails" style="width: 100%">
            <el-table-column label="事件" prop="event_name" />
            <el-table-column label="方向" prop="direction"  />
            <el-table-column label="礼金" prop="amount" />
            <el-table-column label="地址" prop="address" />
            <el-table-column label="备注" prop="gift_note" />
            <el-table-column align="right">
      <template #header>
        <el-input v-model="search" size="small" placeholder="Type to search" />
      </template>
      <template #default="scope">
       
        <el-button
          size="small"
          type="primary"
          @click="handleGift(Number(scope.row.gift_book_id))"
        >
          礼薄查看
        </el-button>
      </template>
    </el-table-column>
  </el-table>

        </div>
     </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue'
import axios from '../axios'
import { ElButton, ElMessage, ElTableColumn } from 'element-plus'
import {onMounted} from 'vue'
import {useRouter} from 'vue-router'
const router = useRouter()
const view = ref('grid')
const loading = ref(false)
// 搜索关键词
const search = ref('')
const filterTableData = computed(() =>
  cards.value.filter(
    (data) =>
      !search.value ||
      data.person_name.toLowerCase().includes(search.value.toLowerCase())
  )
)
// 处理卡片点击事件
const id = ref<number | null>(null)
const handledetail = (cardId: number) =>{
  if(view.value === 'detail'){
    view.value = 'grid'

  }else if(view.value === 'grid'){
    view.value = 'detail'
    id.value = cardId
    fetchCardDetails(cardId)
  }

}
//卡片模型
interface CardSumary {
  card_id:number
  person_name:string
  received_count:number
  received_amount:number
  given_count:number
  given_amount:number
  net_amount:number
}
// 获取卡片数据
const cards = ref<CardSumary[]>([])
const fetchCards = async () =>{
  try{
    const res = await axios.get('/cards')
      console.log(res.data)
      cards.value = res.data.cards
      
  }catch(err:any){
    const msg = err.response?.data?.error || '获取卡片数据失败'
    ElMessage.error(msg)
  }finally{
    loading.value = false
  }
}
//详细信息相关
interface CardDetail {
  id: number
  gift_book_id: number
  person_name: string
  amount: number
  address: string
  gift_note: string
  event_name: string
  event_date: string
  direction: string
}

const cardDetails = ref<CardDetail[]>([])
const handleGift = (id:number) => {
  
  router.push(`/giftbooks/${id}`)
}

const fetchCardDetails = async (cardId: number) => {
  try {
    const res = await axios.get(`/cards/detail?card_id=${cardId}`)
    console.log(res.data)
    cardDetails.value = res.data.records
  } catch (err: any) {
    const msg = err.response?.data?.error || '获取卡片详情失败'
    ElMessage.error(msg)
  }
}

onMounted(() => {
  loading.value = true
  fetchCards()
})
</script>
<style scoped>
.card-container{
  display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 5px;
    padding: 10px;
    margin-top:20px;
    cursor:pointer;
}

.card-content{
  display:flex;
    flex-direction:column;
    justify-content:center;
    
}
.card{
  padding:5px;
    margin:5px;
    overflow-y:hidden;
    width:90%;
    border-radius:10px;
    
}
.data{
  margin:5px;
}
.positive { color: #67c23a; font-weight: bold; }  /* 绿色：净收 */
.negative { color: #f56c6c; font-weight: bold; }  /* 红色：净出 */

.detail-content{
  padding:20px;
  
}
</style>