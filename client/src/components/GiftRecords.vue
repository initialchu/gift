<template>
  <el-table :data="filterTableData" style="width: 100%">
    <el-table-column label="姓名" prop="person_name" />
    <el-table-column label="礼金" prop="amount" />
    <el-table-column label="地址" prop="address" />
    <el-table-column label="备注" prop="gift_note" />
    <el-table-column label="时间" prop="gone_at" v-if="direction === '去'" />
    <el-table-column align="right">
      <template #header>
        <el-input v-model="search" size="small" placeholder="Type to search" />
      </template>
      <template #default="scope">
        <el-button size="small" @click="handleEdit(scope.row)">
          编辑
        </el-button>
        <el-button
          size="small"
          type="danger"
          @click="handleDelete(String(route.params.id), scope.row.ID)"
        >
          删除
        </el-button>
      </template>
    </el-table-column>
  </el-table>

  <!-- 编辑记录弹窗 -->
  <el-dialog class="edit-dialog" v-model="editDialogVisible" title="编辑记录" >
    <el-form :model="editForm" label-width="60px">
      <el-form-item label="姓名">
        <span>{{ editForm.person_name }}</span>
      </el-form-item>
      <el-form-item label="金额">
        <el-input v-model.number="editForm.amount" type="number" />
      </el-form-item>
      <el-form-item label="地址">
        <el-input v-model="editForm.address" />
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="editForm.gift_note" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="editDialogVisible = false">取消</el-button>
      <el-button type="primary" @click="submitEdit">确定</el-button>
    </template>
  </el-dialog>
</template>

<script lang="ts" setup>

import { computed, ref } from 'vue'
import axios from '../axios'
import { ElMessage } from 'element-plus'
import {onMounted} from 'vue'
import {useRoute} from 'vue-router'
import { ElMessageBox } from 'element-plus'
const route = useRoute()
defineProps<{direction:string}>()
interface Record {
  ID:number
  gift_book_id : number
  person_name: string
  amount: number
  address: string
  gift_note:string
  card_id: number
  gone_at: string

}

const search = ref('')
const filterTableData = computed(() =>
  tableData.value.filter(
    (data) =>
      !search.value ||
      data.person_name.toLowerCase().includes(search.value.toLowerCase())
  )
)
// 编辑记录
const editDialogVisible = ref(false)
const editingId = ref<number | null>(null)
const editForm = ref({
  person_name: '',
  amount: 0,
  address: '',
  gift_note: '',
  card_id: 0,
  gone_at: '',
})

const handleEdit = (row: Record) => {
  editingId.value = row.ID
  editForm.value = {
    person_name: row.person_name,
    amount: row.amount,
    address: row.address || '',
    gift_note: row.gift_note || '',
      card_id: row.card_id || 0,
      gone_at: row.gone_at || '',
  }
  editDialogVisible.value = true
}

const submitEdit = async () => {
  try {
    await axios.post(
      `/admin/giftrecord/${route.params.id}/records/${editingId.value}/edit`,
      editForm.value,
    )
    ElMessage.success('编辑成功')
    editDialogVisible.value = false
    fetchRecords()
  } catch (err: any) {
    const msg = err.response?.data?.error || '编辑失败'
    ElMessage.error(msg)
  }
}

//删除记录
const deleteRecord = async(id:string,rid:number)=>{
    try{
      await axios.post(`/admin/giftrecord/${id}/records/${rid}`)
      ElMessage.success('删除成功')
      fetchRecords()
    }catch(err:any){
      const msg = err.response?.data?.error||'删除失败'
      ElMessage.error(msg)
    }
}
const handleDelete = async(id:string ,rid:number) => {
  try{
    await ElMessageBox.confirm('确定要删除吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    deleteRecord(id,rid)
    
  }catch{
    
  }
  
}
//获取Record数据

const tableData=ref<Record[]>([])
const fetchRecords = async ()=>{
  try{
    const res = await axios.get(`/giftrecord/${route.params.id}/records`)
    
    tableData.value = res.data.gift_records
    tableData.value.forEach(record=>{
      if(record.gone_at){
        record.gone_at = record.gone_at.split('T')[0]||''
      }
    })
  }catch(err:any){
    const msg = err.response?.data?.error||'获取记录失败'
    ElMessage.error(msg)
  }
}  
defineExpose({
  fetchRecords
})

onMounted(()=>{
  fetchRecords()
  
})

</script>
<style scoped>

.edit-dialog{
    width:450px;
}
@media (max-width: 768px) {
  .edit-dialog {
   min-width:unset;
   
  }
}
</style>