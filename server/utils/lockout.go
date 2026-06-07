package utils

import (
	"sync"
	"time"
)

// 锁定信息结构体
type loginStatus struct {
	Attempts    int        //失败次数
	LockedUntil *time.Time //锁定截止时间
}

var (
	loginTracker = make(map[string]*loginStatus) //跟踪登录状态的map，key为用户名
	trackerMu    sync.Mutex                      //保护loginTracker的互斥锁,确保并发安全
)

const (
	maxAttempts     = 5                //最大失败次数
	lockoutDuration = 15 * time.Minute //锁定持续时间
)

// 记录登录失败
func RecordFailedAttempt(username string) {
	trackerMu.Lock()
	defer trackerMu.Unlock()
	status, exists := loginTracker[username]
	if !exists {
		status = &loginStatus{}
		loginTracker[username] = status
	}
	status.Attempts++
	if status.Attempts >= maxAttempts {
		t := time.Now().Add(lockoutDuration)
		status.LockedUntil = &t
	}
}

// 检查用户是否被锁定
func IsLocked(username string) bool {
	trackerMu.Lock()
	defer trackerMu.Unlock()
	status, exists := loginTracker[username]
	if !exists || status.LockedUntil == nil {
		return false
	}
	//检查当前时间是否超过锁定截止时间
	if time.Now().After(*status.LockedUntil) {
		//解锁用户
		delete(loginTracker, username)
		return false
	}
	return true
}

// 登录成功后重置登录状态
func ResetLoginStatus(username string) {
	trackerMu.Lock()
	defer trackerMu.Unlock()
	delete(loginTracker, username)
}

// 获取剩余锁定时间
func LockRemainingTime(username string) time.Duration {
	trackerMu.Lock()
	defer trackerMu.Unlock()
	status, exits := loginTracker[username]
	if !exits || status.LockedUntil == nil {
		return 0
	}
	return time.Until(*status.LockedUntil)
}
