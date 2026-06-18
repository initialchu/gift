 <!-- 这里是展示礼薄的卡片网格 -->
  
 

<template>
<div class="card-grid" v-loading="loading">
    <el-card
      class="books"
      v-for="book in giftbooks"
      :key="book.ID"
      shadow="hover"
      @click="goDetail(book.ID)"
    >
        <template #header >


            <div class="card-header">
                <el-dropdown @click.stop>
                    <img @click.stop class="option" src="@/assets/threedot.svg" alt="" />
                    <template #dropdown >
                        <el-dropdown-menu>
                            <el-dropdown-item @click="openEditDialog(book)">编辑</el-dropdown-item>
                            <el-dropdown-item @click="confirmDelete(book.ID)">删除</el-dropdown-item>
                        </el-dropdown-menu>
                    </template>
                </el-dropdown>
                <div class="header-content"> 
                    <span>{{ book.event_name }}</span>
                    <span>{{ formatDate(book.event_date) }}</span>
                    <el-tag class="direction-tag" :type="book.direction==='来'? 'success':'danger'">{{ book.direction }}</el-tag>
                </div>
                
            </div>
           
            
        </template>
         <div class="card-content" >
                <img
                    src="../assets/test.png"
                    style="width: 100% "
                />
                <span style="margin-top:10px;">CreatedBy: {{ book.created_by }}</span>
                <span>创建时间: {{ formatDate(book.CreatedAt) }}</span>
                <span >更新时间: {{ formatDate(book.UpdatedAt) }}</span>
            </div>
        
    </el-card>
</div>

    <!-- 编辑礼薄弹窗 -->
    <el-dialog class="edit-dialog" v-model="editDialogVisible" title="编辑礼薄">
      <el-form :model="editForm" label-width="80px">
        <el-form-item label="事件名称">
          <el-input v-model="editForm.event_name" placeholder="请输入礼薄事件" />
        </el-form-item>
        <el-form-item label="方向">
          <el-radio-group v-model="editForm.direction">
            <el-radio value="来">来</el-radio>
            <el-radio value="去">去</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="日期">
          <el-date-picker v-model="editForm.event_date" type="date" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitEdit">确定</el-button>
      </template>
    </el-dialog>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import axios from '../axios'

const router = useRouter()
const loading = ref(false)
const props = defineProps<{direction?:string}>()
interface GiftBook {
  ID: number
  CreatedAt: string
  UpdatedAt: string
  created_by: string
  event_name: string
  event_date: string
  direction: string
}
// 这里是获取礼薄列表的函数
const giftbooks = ref<GiftBook[]>([])

const fetchGiftBooks = async ()=>{
    loading.value= true
    try{
      const url = props.direction ? `/giftbooks?direction=${props.direction}`:'/giftbooks'
        const res = await axios.get(url)
        console.log(res.data)
        giftbooks.value =res.data.giftbooks
    }catch(err:any){
        const msg = err.response?.data?.error || '获取礼薄列表失败'
        ElMessage.error(msg)
    }finally{
        loading.value= false
    }
    
}
//导出函数供外部调用
defineExpose({
    fetchGiftBooks
})

// --- 编辑礼薄相关 ---
const editDialogVisible = ref(false)
const editingBookId = ref<number | null>(null)
const editForm = ref({
  event_name: '',
  event_date: '',
  direction: '来',
})

// 打开编辑弹窗，预填当前卡片数据
const openEditDialog = (book: GiftBook) => {
  editingBookId.value = book.ID
  editForm.value = {
    event_name: book.event_name,
    event_date: book.event_date,
    direction: book.direction,
  }
  editDialogVisible.value = true
}

// 提交编辑
const submitEdit = async () => {
  if (!editingBookId.value) return
  try {
    await axios.post(`/admin/giftbook/edit/${editingBookId.value}`, editForm.value)
    ElMessage.success('修改成功')
    editDialogVisible.value = false
    fetchGiftBooks()
  } catch (err: any) {
    const msg = err.response?.data?.error || '修改失败'
    ElMessage.error(msg)
  }
}

// 点击卡片跳转详情页
const goDetail = (id: number) => {
  if(props.direction){
    if(props.direction ==='来')
    router.push(`/giftbooks/${id}`)
    else{
    router.push(`/account/${id}`)
  }
  }
 
}

// 组件加载时获取礼薄列表
onMounted(()=>{
    fetchGiftBooks()
})
//改日期格式
const formatDate = (iso: string): string => {
  if (!iso) return ''
  return String(iso.split('T')[0])
}
// 这里是删除礼薄的函数
const confirmDelete = async (id: number) => {
  try {
    await ElMessageBox.confirm('确定要删除这个礼薄吗？', '确认删除', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
       lockScroll: false,
    })
    // 用户点了确定才执行删除
    deletebook(id)
  } catch {
    // 用户点了取消，什么都不做
  }
}
const deletebook = async(id:number)=>{
    try{
        await axios.post(`/admin/giftbook/${id}`)

        ElMessage.success('删除成功')
        // 删除成功后重新获取礼薄列表
        fetchGiftBooks()
    }catch(err:any){
        const msg = err.response?.data?.error || '删除失败'
        ElMessage.error(msg)
    }
}
</script>
<style scoped>

.card-grid{
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 20px;
    padding: 20px;
}

.books{
    padding:5px;
    margin:20px;
    overflow-y:hidden;
    width:80%;
    border-radius:10px;
}

.card-content{
    
    display:flex;
    flex-direction:column;
    justify-content:center;
    align-items:center;
}
.option{
    margin-bottom: 5px;
    width:20px;
    height:20px;
    margin-left: auto;
    cursor:pointer;
    
    
}
.card-header {
    
    display: flex;
    flex-direction: column;
    
    
}
.header-content{
    display: flex;
    justify-content: space-between;
   
}
.direction-tag{
    border-radius:50%;
    height:25px;
    width:25px;
    
}
.edit-dialog{
    width:500px;
}
@media (max-width: 768px) {
  .edit-dialog {
   min-width:unset;
   
    
  }
}
</style>