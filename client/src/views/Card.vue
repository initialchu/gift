<template>
  <div class="cards">
    <!-- grid视图-->
     <template v-if="view==='grid'">
       <div class="page-header" >
          <h2>人情卡片</h2>
          <el-input v-model="search" placeholder="搜索人名..." clearable></el-input>
          
       </div>
       <div :data="filterTableData" class="card-container" v-loading="loading">
          <el-card  class="card" v-for="card in filterTableData" :key="card.card_id" >
            
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
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue'
import axios from '../axios'
import { ElMessage } from 'element-plus'
import {onMounted} from 'vue'
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
</style>