import axios from 'axios'

const instance = axios.create({
    baseURL:'http://localhost:8080/api',
})
// 添加请求拦截器
instance.interceptors.request.use(config=>{
    const token = localStorage.getItem('token')
    if(token){
        config.headers.Authorization = `Bearer ${token}`

    }
    return config
})
// 添加响应拦截器
instance.interceptors.response.use((response)=>response,(error)=>{
    if(error.response?.status ===401){
        localStorage.removeItem(`token`)
        window.location.href = '/login'
    }
    return Promise.reject(error)

})

export default instance