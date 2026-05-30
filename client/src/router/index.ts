import { createRouter, createWebHistory } from 'vue-router'


const routes = [
  {
    path:"/",
    redirect:"/home"

  },
  {
    path:"/login",
    name:"Login",
    component:()=>import("../components/Login.vue")
  },
  {
    path:"/home",
    name:"Home",
    component:()=>import("../views/Home.vue")
  },
  {
    path:"/card",
    name:"Card",
    component:()=>import("../views/Card.vue")
  },
  {
    path:"/giftbooks",
    name:"GiftBooks",
    component:()=>import("../views/GiftBooks.vue")
  }
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: routes
})




export default router
