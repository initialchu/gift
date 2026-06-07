<template>
  <div class="login-container">
    <el-card class="login-card">
    <el-form class="login-form">
        <h1 class="login-title">登录</h1>
        <el-form-item  class="label"  label="账号">
            <el-input v-model="user.username" type="text" placeholder="请输入账号"></el-input>
        </el-form-item>
        <el-form-item class="label" label="密码">
            <el-input v-model="user.password" type="password" placeholder="请输入密码"></el-input>
        </el-form-item>
        <div class="captcha">
        <el-form-item class="label " label="验证码">
            <el-input v-model="user.captcha" type="text" placeholder="请输入验证码"></el-input>
            
        </el-form-item>
        <img :src="captchaImage" @click="getCaptcha" alt="验证码" class="captcha-image">
        </div>
        <el-button type="primary" @click="handleLogin" :loading="loading">登录</el-button>
    </el-form>
    </el-card>
  </div>
</template>

<script lang="ts" setup>
import{useAuthStore } from '../stores/auth'
import { ElMessage } from 'element-plus'
import { useRoute } from 'vue-router'
import {ref} from 'vue'
import {useRouter} from 'vue-router'
import axios from '../axios'
import { onMounted } from 'vue'
const authStore = useAuthStore()
const router = useRouter()
const route = useRoute()
const user = ref({
    username:'',
    password:'',
    captcha:''
})
const loading = ref(false)
const redirect = (route.query.redirect as string) || '/home'
// 登录按钮点击事件
const handleLogin = async()=>{
    if (loading.value) return
    loading.value = true
    
    try{
        const success = await authStore.login(user.value.username, user.value.password, captchaId.value, user.value.captcha)
        if(success){
            ElMessage.success('登录成功')

           
            router.push(redirect)
        }
    }catch(err:any){
        const msg = err.response?.data?.error||'登录失败'
        ElMessage.error(msg)

    }finally{
        loading.value = false
    }
    // 登录成功后会自动更新 authStore 的状态，组件会响应式地反映登录状态，无需额外操作
    
    
}
//页面加载时获取验证码
const captchaId=ref('')
const captchaImage = ref('')
const getCaptcha = async()=>{
    try{
        const res = await axios.get('/auth/captcha')
        captchaId.value=res.data.id
       
        // 更新验证码图片
       captchaImage.value=  res.data.captcha_img
        user.value.captcha = ''
    }catch(err){
        ElMessage.error('获取验证码失败')
}
}
onMounted(()=>{
    getCaptcha()
})
</script>
<style scoped>

.login-title{
    font-size:2rem;
    margin-bottom:5rem;
}
.label:deep(.el-form-item__label){
    font-size:1.2rem;
    color:#333;
}
.el-form-item{
    width:80%;
    margin-bottom:2rem;
    
    size:2rem;
}
.el-input{
    width:100%;
    font-size:1.2rem;
    height:3rem;
    
}
.login-container{
    
    width:100vw;
    height:100vh;
    display:flex;
    flex-direction:column;
    justify-content:center;
    align-items:center;
    position:relative;
    
}
.login-card{
    max-width:400px;
    width:90%;
    height:60%;
}
.login-form{
    width:100%;
    height:100%;
    display:flex;
    flex-direction:column;
    justify-content:center;
    align-items:center;
}
.el-button{
    width:10rem;
    height:2.5rem;
    font-size:1.2rem;
    letter-spacing:0.5rem;

}
.captcha{
    display:flex;
    
   
    
    
    width:80%;
}
.captcha-image{

    border: #bebebe    solid 1px;
    height:3rem;
    margin-left:1rem;
    cursor:pointer;
    box-sizing: border-box;
}
</style>