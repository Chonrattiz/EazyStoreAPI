# Frontend Spec: Token Management & UX

## ภาพรวม
Backend ได้ implement `Access Token` + `Refresh Token` pattern แล้ว
Frontend ต้องจัดการ token lifecycle และให้ user experience ที่ดี

---

## 1. Token Storage Strategy

### ✅ Recommended
```javascript
// LocalStorage/SessionStorage
localStorage.setItem('access_token', accessToken);
localStorage.setItem('refresh_token', refreshToken);
localStorage.setItem('token_expires_at', expiresAtTimestamp); // Unix timestamp
```

### ⚠️ Secure (แต่ยากกว่า)
- เก็บ `access_token` ใน Memory
- เก็บ `refresh_token` ใน HttpOnly Cookie (ต้องให้ Backend set ด้วย)
- ปัจจุบันถ้า Backend ส่ง token เป็น JSON ก็เก็บใน localStorage ได้ก่อน

---

## 2. Login Flow

### 2.1 ตอนกดปุ่ม Login
```javascript
// Response จาก Backend
{
  "message": "เข้าสู่ระบบสำเร็จ",
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 900,  // วินาที (15 นาที)
  "user": { ... }
}

// Frontend ต้องทำ
const now = Math.floor(Date.now() / 1000);
const expiresAt = now + 900; // Unix timestamp

localStorage.setItem('access_token', response.access_token);
localStorage.setItem('refresh_token', response.refresh_token);
localStorage.setItem('token_expires_at', expiresAt);

// บันทึก user info
localStorage.setItem('user', JSON.stringify(response.user));
```

---

## 3. Access Token Check Before API Call

### 3.1 ก่อนทุกครั้งที่เรียก API
```javascript
function checkAndRefreshToken() {
  const expiresAt = parseInt(localStorage.getItem('token_expires_at'));
  const now = Math.floor(Date.now() / 1000);
  
  // ถ้า token หมดอายุ หรือ ใกล้หมดแล้ว (เหลือ 1 นาที)
  if (now > expiresAt - 60) {
    return refreshAccessToken();
  }
  return Promise.resolve();
}

// ใช้ใน API interceptor (axios/fetch)
async function apiCall(url, options) {
  await checkAndRefreshToken();
  
  const accessToken = localStorage.getItem('access_token');
  const headers = {
    ...options.headers,
    'Authorization': `Bearer ${accessToken}`
  };
  
  return fetch(url, { ...options, headers });
}
```

### 3.2 ตัวอย่าง Axios Interceptor
```javascript
import axios from 'axios';

const api = axios.create({
  baseURL: 'http://localhost:8080/api'
});

// Request Interceptor
api.interceptors.request.use(async (config) => {
  // ตรวจสอบ token ก่อนส่ง request
  const expiresAt = parseInt(localStorage.getItem('token_expires_at'));
  const now = Math.floor(Date.now() / 1000);
  
  if (now > expiresAt - 60) {
    await refreshAccessToken();
  }
  
  const accessToken = localStorage.getItem('access_token');
  if (accessToken) {
    config.headers.Authorization = `Bearer ${accessToken}`;
  }
  
  return config;
});

// Response Interceptor
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;
    
    // ถ้า error 401 และยังไม่เคยลอง refresh
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;
      
      try {
        await refreshAccessToken();
        return api(originalRequest); // ลองเรียก API เดิมอีกครั้ง
      } catch (refreshError) {
        // Refresh ล้มเหลว = ต้อง login ใหม่
        redirectToLogin();
      }
    }
    
    return Promise.reject(error);
  }
);
```

---

## 4. Refresh Token Function

### 4.1 ฟังก์ชัน Refresh
```javascript
async function refreshAccessToken() {
  try {
    const refreshToken = localStorage.getItem('refresh_token');
    
    if (!refreshToken) {
      throw new Error('No refresh token found');
    }
    
    const response = await fetch('http://localhost:8080/api/auth/refresh', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ refresh_token: refreshToken })
    });
    
    if (!response.ok) {
      throw new Error('Refresh failed');
    }
    
    const data = await response.json();
    
    // อัปเดต token ใหม่
    const now = Math.floor(Date.now() / 1000);
    const expiresAt = now + data.expires_in; // expires_in เป็นวินาที
    
    localStorage.setItem('access_token', data.access_token);
    localStorage.setItem('token_expires_at', expiresAt);
    
    return data.access_token;
  } catch (error) {
    console.error('Token refresh failed:', error);
    // ล้มเหลว = ต้อง login ใหม่
    redirectToLogin();
    throw error;
  }
}
```

---

## 5. Logout Flow

### 5.1 ตอนกดปุ่ม Logout
```javascript
async function logout() {
  try {
    const accessToken = localStorage.getItem('access_token');
    
    // ส่ง request ไป backend เพื่อยกเลิก refresh token
    await fetch('http://localhost:8080/api/auth/logout', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    });
    
  } catch (error) {
    console.error('Logout error:', error);
    // ยังคงต้องลบ token ไม่ว่าจะ fail หรือไม่
  } finally {
    // ลบ token ออกจาก storage
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('token_expires_at');
    localStorage.removeItem('user');
    
    // Redirect ไปหน้า login
    window.location.href = '/login';
  }
}
```

---

## 6. Token Expired Detection & UX

### 6.1 ตรวจสอบ Token ตอน App Load
```javascript
// ใน useEffect หรือ componentDidMount
useEffect(() => {
  const accessToken = localStorage.getItem('access_token');
  const expiresAt = parseInt(localStorage.getItem('token_expires_at'));
  
  if (!accessToken) {
    // ไม่มี token = ต้อง login
    redirectToLogin();
    return;
  }
  
  const now = Math.floor(Date.now() / 1000);
  if (now > expiresAt) {
    // Token หมดอายุแล้ว
    logout();
    return;
  }
  
  // Token ยังใช้ได้ = ทำต่อไป
}, []);
```

### 6.2 Progress Bar / Countdown (Optional)
```javascript
// แสดงให้ user รู้ว่า token จะหมดอายุเมื่อไหร่
function getTokenExpiryCountdown() {
  const expiresAt = parseInt(localStorage.getItem('token_expires_at'));
  const now = Math.floor(Date.now() / 1000);
  
  const remainingSeconds = expiresAt - now;
  
  if (remainingSeconds <= 0) return 'Token Expired';
  if (remainingSeconds < 60) return `Expires in ${remainingSeconds}s`;
  
  const remainingMinutes = Math.floor(remainingSeconds / 60);
  return `Expires in ${remainingMinutes}m`;
}
```

---

## 7. Error Handling

### 7.1 Status Code ที่ต้องรู้
| Code | Meaning | Action |
|------|---------|--------|
| 401 | Unauthorized (token invalid/expired) | Refresh token → Retry หรือ Login ใหม่ |
| 403 | Forbidden (permission denied) | แสดง error message ให้ user |
| 500 | Server error | Show generic error & retry later |

### 7.2 ตัวอย่าง Error Handling
```javascript
async function apiCall(url, options) {
  try {
    await checkAndRefreshToken();
    
    const response = await fetch(url, {
      ...options,
      headers: {
        ...options.headers,
        'Authorization': `Bearer ${localStorage.getItem('access_token')}`
      }
    });
    
    if (response.status === 401) {
      // Token ไม่ถูกต้อง = logout
      logout();
      throw new Error('Session expired');
    }
    
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'API Error');
    }
    
    return response.json();
  } catch (error) {
    console.error('API Call failed:', error);
    // แสดง error toast/modal ให้ user
    showErrorMessage(error.message);
    throw error;
  }
}
```

---

## 8. สรุป Checklist

- [ ] เก็บ `access_token`, `refresh_token`, `token_expires_at` ใน localStorage
- [ ] ตรวจสอบ token ก่อนเรียก API (ใน Request Interceptor)
- [ ] Implement `refreshAccessToken()` function
- [ ] Handle 401 error → Refresh → Retry
- [ ] Implement `logout()` function
- [ ] ตรวจสอบ token ตอน App Load
- [ ] Handle token expiry gracefully (ไม่ให้ error ปรากฏต่อ user)
- [ ] (Optional) แสดง countdown เมื่อ token ใกล้หมดอายุ
- [ ] (Optional) Auto-lock UI หรือ modal warning เมื่อ token ใกล้หมด

---

## 9. Code Example (React)

```javascript
// hooks/useAuth.js
import { useCallback } from 'react';

export function useAuth() {
  const refreshAccessToken = useCallback(async () => {
    const refreshToken = localStorage.getItem('refresh_token');
    if (!refreshToken) throw new Error('No refresh token');
    
    const res = await fetch('http://localhost:8080/api/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken })
    });
    
    if (!res.ok) throw new Error('Refresh failed');
    
    const data = await res.json();
    const expiresAt = Math.floor(Date.now() / 1000) + data.expires_in;
    
    localStorage.setItem('access_token', data.access_token);
    localStorage.setItem('token_expires_at', expiresAt);
    
    return data.access_token;
  }, []);
  
  const logout = useCallback(async () => {
    const token = localStorage.getItem('access_token');
    try {
      await fetch('http://localhost:8080/api/auth/logout', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      });
    } finally {
      localStorage.clear();
      window.location.href = '/login';
    }
  }, []);
  
  return { refreshAccessToken, logout };
}
```

---

## Notes สำหรับ POS System
- Token expiry ควรให้ POS เรียก refresh อัตโนมัติ ไม่ให้ user กดปุ่มใหม่
- ถ้า refresh ล้มเหลว 3 ครั้ง เก็บ transaction ที่ pending ไว้ แล้ว redirect ไป login
- ควร log transaction timestamp = ถ้า session ขาดก็รู้ว่าต้อง sync ไหม
- ถ้าใช้ qrcode payment ควรให้ user re-scan หรือ verify ใหม่เมื่อ session หมด
