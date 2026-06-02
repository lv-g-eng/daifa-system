// API 基础配置
const API_BASE = '/api';

// 存储Token
function setToken(token) {
  localStorage.setItem('token', token);
}

function getToken() {
  return localStorage.getItem('token');
}

function removeToken() {
  localStorage.removeItem('token');
}

// 存储用户信息
function setUser(user) {
  localStorage.setItem('user', JSON.stringify(user));
}

function getUser() {
  const user = localStorage.getItem('user');
  return user ? JSON.parse(user) : null;
}

function removeUser() {
  localStorage.removeItem('user');
}

// 检查登录状态
function isLoggedIn() {
  return !!getToken();
}

// 退出登录
function logout() {
  removeToken();
  removeUser();
  // 统一跳转到登录页（'/' 是后端注册的路由，避免 /xxx/index.html 404）
  window.location.href = '/';
}

// 通用请求方法
async function request(url, options = {}) {
  const token = getToken();

  const headers = {
    'Content-Type': 'application/json',
    ...options.headers,
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  try {
    const response = await fetch(API_BASE + url, {
      ...options,
      headers,
    });

    const data = await response.json();

    if (response.status === 401) {
      logout();
      return;
    }

    return data;
  } catch (error) {
    console.error('请求失败:', error);
    throw error;
  }
}

// GET 请求
function get(url, params = {}) {
  const queryString = new URLSearchParams(params).toString();
  const fullUrl = queryString ? `${url}?${queryString}` : url;
  return request(fullUrl, { method: 'GET' });
}

// POST 请求
function post(url, data = {}) {
  return request(url, {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

// PUT 请求
function put(url, data = {}) {
  return request(url, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

// DELETE 请求
function del(url) {
  return request(url, { method: 'DELETE' });
}

// 文件上传
async function uploadFile(file) {
  const formData = new FormData();
  formData.append('file', file);

  const token = getToken();

  const response = await fetch(API_BASE + '/upload', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    body: formData,
  });

  return response.json();
}

// 批量文件上传
async function uploadFiles(files) {
  const formData = new FormData();
  for (const file of files) {
    formData.append('files', file);
  }

  const token = getToken();

  const response = await fetch(API_BASE + '/upload/batch', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    body: formData,
  });

  return response.json();
}

// API 接口

// 认证
const authAPI = {
  login: (username, password) => post('/login', { username, password }),
  register: (data) => post('/register', data),
  getUserInfo: () => get('/user/info'),
};

// 总后台
const adminAPI = {
  getStats: () => get('/admin/stats'),
  listMerchants: () => get('/admin/merchants'),
  createMerchant: (data) => post('/admin/merchants', data),
  updateMerchant: (id, data) => put(`/admin/merchants/${id}`, data),
  deleteMerchant: (id) => del(`/admin/merchants/${id}`),
  verifyPayment: (id, verified) => put(`/admin/users/${id}/verify-payment`, { verified }),
  // 用户管理
  listUsers: () => get('/admin/users'),
  blockUser: (id, block) => put(`/admin/users/${id}/block`, { block }),
  deleteUser: (id) => del(`/admin/users/${id}`),
  // 提现管理
  listWithdrawals: (status) => get('/admin/withdrawals', { status }),
  processWithdrawal: (id, status) => put(`/admin/withdrawals/${id}`, { status }),
  // 收款码审核
  listPayments: () => get('/admin/payments'),
};

// 商家后台
const merchantAPI = {
  getStats: () => get('/merchant/stats'),

  // 素材
  listMaterials: (params) => get('/merchant/materials', params),
  getMaterial: (id) => get(`/merchant/materials/${id}`),
  createMaterial: (data) => post('/merchant/materials', data),
  updateMaterial: (id, data) => put(`/merchant/materials/${id}`, data),
  deleteMaterial: (id) => del(`/merchant/materials/${id}`),
  batchCreateMaterials: (materials) => post('/merchant/materials/batch', { materials }),

  // 发布码
  listPublishCodes: () => get('/merchant/publish-codes'),
  createPublishCode: (data) => post('/merchant/publish-codes', data),

  // 任务
  listTasks: () => get('/merchant/tasks'),
  createTask: (data) => post('/merchant/tasks', data),

  // 审核
  listSubmissions: (params) => get('/merchant/submissions', params),
  reviewSubmission: (id, data) => put(`/merchant/submissions/${id}`, data),
  checkLink: (id) => post(`/merchant/submissions/${id}/check`),

  // 用户
  listUsers: () => get('/merchant/users'),
  blockUser: (id, block) => put(`/merchant/users/${id}/block`, { block }),

  // 提现
  listWithdrawals: (params) => get('/merchant/withdrawals', params),
  processWithdrawal: (id, data) => put(`/merchant/withdrawals/${id}`, data),
};

// 用户端
const userAPI = {
  getHome: () => get('/user/home'),

  // 任务
  listTasks: (params) => get('/user/tasks', params),
  getTaskDetail: (id) => get(`/user/tasks/${id}`),

  // 素材
  listMaterials: (params) => get('/user/materials', params),
  getMaterialDetail: (id) => get(`/user/materials/${id}`),
  listPublishCodes: (params) => get('/user/publish-codes', params),

  // 提交
  submitTask: (data) => post('/user/submissions', data),
  listSubmissions: (params) => get('/user/submissions', params),

  // 个人中心
  getProfile: () => get('/user/profile'),
  updateProfile: (data) => put('/user/profile', data),
  changePassword: (oldPassword, newPassword) => put('/user/password', { old_password: oldPassword, new_password: newPassword }),

  // 提现
  requestWithdraw: (data) => post('/user/withdraw', data),
  requestWithdrawal: (data) => post('/user/withdraw', data), // alias
  listWithdrawals: () => get('/user/withdrawals'),
  getWithdrawalHistory: () => get('/user/withdrawals'), // alias

  // 订单
  listOrders: () => get('/user/orders'),

  // 裂变
  getReferralInfo: () => get('/user/referrals'),
};

// 公共API
const publicAPI = {
  listPlatforms: () => get('/platforms'),
  usePublishCode: (code) => get(`/publish-codes/${code}`),
  getCategories: () => get('/categories'),
};

// Toast 提示
function showToast(message, type = 'success') {
  const container = document.querySelector('.toast-container') || createToastContainer();

  const toast = document.createElement('div');
  toast.className = `toast toast-${type}`;
  toast.innerHTML = `
    <i class="fas ${type === 'success' ? 'fa-check-circle' : type === 'error' ? 'fa-times-circle' : 'fa-exclamation-circle'}"></i>
    <span>${message}</span>
  `;

  container.appendChild(toast);

  setTimeout(() => {
    toast.remove();
  }, 3000);
}

function createToastContainer() {
  const container = document.createElement('div');
  container.className = 'toast-container';
  document.body.appendChild(container);
  return container;
}

// 格式化日期
function formatDate(dateStr) {
  if (!dateStr) return '-';
  const date = new Date(dateStr);
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

// 格式化金额
function formatMoney(amount) {
  return '¥' + Number(amount || 0).toFixed(2);
}

// 状态文本映射
const statusText = {
  active: '正常',
  blocked: '已禁用',
  expired: '已过期',
  pending: '待审核',
  approved: '已通过',
  rejected: '已驳回',
  invalid: '已失效',
  paid: '已打款',
};

const statusClass = {
  active: 'badge-success',
  blocked: 'badge-danger',
  expired: 'badge-warning',
  pending: 'badge-warning',
  approved: 'badge-success',
  rejected: 'badge-danger',
  invalid: 'badge-secondary',
  paid: 'badge-success',
};

function getStatusBadge(status) {
  return `<span class="badge ${statusClass[status] || 'badge-secondary'}">${statusText[status] || status}</span>`;
}

// 跳转到对应页面
function navigateByRole() {
  const user = getUser();
  if (!user) {
    window.location.href = '/';
    return;
  }

  switch (user.role) {
    case 'admin':
      window.location.href = '/admin';
      break;
    case 'merchant':
      window.location.href = '/merchant';
      break;
    default:
      window.location.href = '/user';
  }
}

// 检查权限
function checkRole(allowedRoles) {
  const user = getUser();
  if (!user || !allowedRoles.includes(user.role)) {
    window.location.href = '/';
    return false;
  }
  return true;
}
