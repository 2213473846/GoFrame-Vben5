<script lang="ts" setup>
import type { SystemMenuItem, SystemMenuPayload } from '#/api/system/menu';
import type { SystemUserListItem } from '#/api/system/user';

import { computed, nextTick, onMounted, reactive, ref } from 'vue';

import { IconifyIcon } from '@vben/icons';
import { useUserStore } from '@vben/stores';

import {
  ElAlert,
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElMessage,
  ElMessageBox,
  ElOption,
  ElRadio,
  ElRadioGroup,
  ElSelect,
  ElSwitch,
  ElTable,
  ElTableColumn,
  ElTag,
  ElTree,
} from 'element-plus';
import type { TableInstance } from 'element-plus';

import {
  createMenuApi,
  deleteMenuApi,
  getMenuGrantInfoApi,
  getMenuListApi,
  grantMenuApi,
  updateMenuApi,
} from '#/api/system/menu';
import { getUserListApi } from '#/api/system/user';
import IconPicker from '#/components/icon-picker/index.vue';

defineOptions({ name: 'SystemMenu' });

const userStore = useUserStore();
const isSuperAdmin = computed(
  () => userStore.userInfo?.roles?.includes('super_admin') ?? false,
);

// ---------------- 列表 ----------------
const loading = ref(false);
const items = ref<SystemMenuItem[]>([]);
const keyword = ref('');

async function fetchList() {
  loading.value = true;
  try {
    const res = await getMenuListApi();
    items.value = res.items ?? [];
    applySavedExpansion();
  } finally {
    loading.value = false;
  }
}

function handleSearch() {
  // 前端按标题/路由名/路径做树过滤
  keyword.value = searchInput.value.trim();
}

const searchInput = ref('');

const visibleItems = computed(() => {
  const kw = keyword.value.toLowerCase();
  if (!kw) return items.value;
  const filter = (nodes: SystemMenuItem[]): SystemMenuItem[] => {
    const result: SystemMenuItem[] = [];
    for (const node of nodes) {
      const children = filter(node.children ?? []);
      const matched
        = node.title.toLowerCase().includes(kw)
        || node.name.toLowerCase().includes(kw)
        || node.path.toLowerCase().includes(kw);
      if (matched || children.length > 0) {
        result.push({ ...node, children });
      }
    }
    return result;
  };
  return filter(items.value);
});

// ---------------- 树表展开状态持久化 ----------------
// 记住上次的展开状态, 无记录默认收起
const EXPAND_STORAGE_KEY = 'vben-system-menu-expanded';

function loadExpandedKeys(): string[] {
  try {
    const raw = localStorage.getItem(EXPAND_STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed.map(String) : [];
  } catch {
    return [];
  }
}

const expandedKeys = ref<string[]>(loadExpandedKeys());
// 非受控模式: 展开由 Element Plus 内部管理, 仅用 localStorage 记录 + 数据加载后恢复
const tableRef = ref<TableInstance>();

// Element Plus 树形表格 expand-change 第二参数为布尔值(是否展开),
// 展开型(type=expand)表格才是数组, 用 rest 参数兼容两种签名
function handleExpandChange(...args: any[]) {
  const row: any = args[0] ?? {};
  const second: unknown = args[1];
  let expanded: boolean;
  if (typeof second === 'boolean') {
    expanded = second;
  } else if (Array.isArray(second)) {
    expanded = second.some((r: any) => String(r?.id) === String(row.id));
  } else {
    expanded = !expandedKeys.value.includes(String(row.id));
  }
  const id = String(row.id);
  const rest = expandedKeys.value.filter((key) => key !== id);
  expandedKeys.value = expanded ? [...rest, id] : rest;
  localStorage.setItem(EXPAND_STORAGE_KEY, JSON.stringify(expandedKeys.value));
}

// 数据加载后恢复上次的展开状态
function applySavedExpansion() {
  if (expandedKeys.value.length === 0) return;
  const rowMap = new Map<string, SystemMenuItem>();
  const walk = (nodes: SystemMenuItem[]) => {
    for (const node of nodes ?? []) {
      rowMap.set(String(node.id), node);
      walk(node.children ?? []);
    }
  };
  walk(visibleItems.value ?? []);
  nextTick(() => {
    for (const key of expandedKeys.value) {
      const row = rowMap.get(key);
      if (row) tableRef.value?.toggleRowExpansion(row, true);
    }
  });
}

// ---------------- 新增/编辑 ----------------
const dialogVisible = ref(false);
const saving = ref(false);
const isEdit = ref(false);
const formRef = ref();
const form = reactive<SystemMenuPayload & { id: number }>({
  id: 0,
  affixTab: false,
  component: '',
  hideInMenu: false,
  icon: '',
  isPublic: false,
  keepAlive: false,
  name: '',
  path: '',
  pid: 0,
  redirect: '',
  remark: '',
  sort: 0,
  status: 1,
  title: '',
  type: 'menu',
});

const rules = {
  name: [{ required: true, message: '请输入路由名称', trigger: 'blur' }],
  path: [{ required: true, message: '请输入路由路径', trigger: 'blur' }],
  title: [{ required: true, message: '请输入菜单标题', trigger: 'blur' }],
};

// 上级菜单候选(目录节点 + 顶级), 编辑时排除自身及其子孙
const parentOptions = computed(() => {
  const flat: { id: number; label: string }[] = [
    { id: 0, label: '顶级菜单' },
  ];
  // 列出所有节点(目录/菜单均可作为上级);
  // 编辑时排除自己及自己的整个子树, 防止上级成环
  const walk = (nodes: SystemMenuItem[], depth: number, skip: boolean) => {
    for (const node of nodes) {
      const self = node.id === form.id;
      if (!skip && !self) {
        flat.push({
          id: node.id,
          label: `${'　'.repeat(depth)}${node.title}`,
        });
      }
      if (node.children?.length) walk(node.children, depth + 1, skip || self);
    }
  };
  walk(items.value, 0, false);
  return flat;
});

function openCreate(parent?: SystemMenuItem) {
  isEdit.value = false;
  Object.assign(form, {
    id: 0,
    affixTab: false,
    component: '',
    hideInMenu: false,
    icon: '',
    isPublic: false,
    keepAlive: false,
    name: '',
    path: '',
    pid: parent?.id ?? 0,
    redirect: '',
    remark: '',
    sort: 0,
    status: 1,
    title: '',
    type: parent ? 'menu' : 'catalog',
  });
  dialogVisible.value = true;
}

function openEdit(row: SystemMenuItem) {
  isEdit.value = true;
  Object.assign(form, {
    id: row.id,
    affixTab: row.affixTab,
    component: row.component,
    hideInMenu: row.hideInMenu,
    icon: row.icon,
    isPublic: row.isPublic,
    keepAlive: row.keepAlive,
    name: row.name,
    path: row.path,
    pid: row.pid,
    redirect: row.redirect,
    remark: row.remark,
    sort: row.sort,
    status: row.status,
    title: row.title,
    type: row.type,
  });
  dialogVisible.value = true;
}

async function handleSave() {
  const valid = await formRef.value?.validate().catch(() => false);
  if (!valid) return;
  if (form.type === 'menu' && !form.component) {
    ElMessage.warning('菜单类型必须填写组件路径');
    return;
  }
  saving.value = true;
  try {
    const payload: SystemMenuPayload = { ...form };
    if (form.pid === 0) {
      payload.pid = 0;
    }
    if (isEdit.value) {
      await updateMenuApi({ ...payload, id: form.id });
      ElMessage.success('更新成功');
    } else {
      await createMenuApi(payload);
      ElMessage.success('新增成功');
    }
    dialogVisible.value = false;
    fetchList();
  } finally {
    saving.value = false;
  }
}

async function handleDelete(row: SystemMenuItem) {
  const hasChildren = (row.children?.length ?? 0) > 0;
  const confirmed = await ElMessageBox.confirm(
    hasChildren
      ? `「${row.title}」下还有子菜单, 请先删除子菜单`
      : `确认删除「${row.title}」吗? 已授权该菜单的用户将失去此页面`,
    '删除确认',
    { type: hasChildren ? 'info' : 'warning' },
  ).catch(() => false);
  if (!confirmed || hasChildren) return;
  await deleteMenuApi(row.id);
  ElMessage.success('删除成功');
  fetchList();
}

// ---------------- 菜单授权 ----------------
const grantVisible = ref(false);
const grantLoading = ref(false);
const grantSaving = ref(false);
const grantUsers = ref<SystemUserListItem[]>([]);
const grantUserId = ref<number>();
const grantTree = ref();
const grantMenus = ref<SystemMenuItem[]>([]);
const grantChecked = ref<number[]>([]);
const grantUsername = ref('');

async function openGrant() {
  grantVisible.value = true;
  grantUserId.value = undefined;
  grantUsername.value = '';
  grantMenus.value = [];
  grantChecked.value = [];
  grantLoading.value = true;
  try {
    const res = await getUserListApi({ page: 1, pageSize: 100 });
    grantUsers.value = (res.items ?? []).filter((u) =>
      isSuperAdmin.value ? u.role !== 'super_admin' : u.role === 'user',
    );
  } finally {
    grantLoading.value = false;
  }
}

async function handleGrantUserChange() {
  if (!grantUserId.value) return;
  grantLoading.value = true;
  try {
    const res = await getMenuGrantInfoApi(grantUserId.value);
    grantUsername.value = res.username;
    grantMenus.value = res.items ?? [];
    grantChecked.value = res.checkedIds ?? [];
    await nextTick();
    grantTree.value?.setCheckedKeys(grantChecked.value, false);
  } finally {
    grantLoading.value = false;
  }
}

async function handleGrantSave() {
  if (!grantUserId.value) {
    ElMessage.warning('请选择要授权的用户');
    return;
  }
  grantSaving.value = true;
  try {
    const checked: number[] = grantTree.value?.getCheckedKeys() ?? [];
    await grantMenuApi(grantUserId.value, checked);
    ElMessage.success(`已保存「${grantUsername.value}」的菜单授权`);
    grantVisible.value = false;
  } finally {
    grantSaving.value = false;
  }
}

const typeText = (type: string) => (type === 'catalog' ? '目录' : '菜单');

onMounted(fetchList);
</script>

<template>
  <div class="p-4">
    <ElAlert
      v-if="!isSuperAdmin"
      class="mb-3"
      type="info"
      :closable="false"
      show-icon
      title="只读视图: 路由由超级管理员统一维护, 你只能把已下发给自己的路由分配给自己的下级用户"
    />
    <div class="mb-4 flex flex-wrap items-center gap-2">
      <ElInput
        v-model="searchInput"
        placeholder="标题 / 路由名 / 路径"
        clearable
        class="w-52"
        @keyup.enter="handleSearch"
      />
      <ElButton type="primary" @click="handleSearch">查询</ElButton>
      <ElButton
        @click="
          () => {
            searchInput = '';
            handleSearch();
          }
        "
        >重置</ElButton
      >
      <div class="flex-1"></div>
      <ElButton @click="openGrant">菜单授权</ElButton>
      <ElButton v-if="isSuperAdmin" type="primary" @click="openCreate()"
        >新增路由</ElButton
      >
    </div>

    <ElTable
      ref="tableRef"
      v-loading="loading"
      :data="visibleItems"
      row-key="id"
      border
      stripe
      @expand-change="handleExpandChange"
    >
      <ElTableColumn prop="title" label="菜单标题" min-width="140">
        <template #default="{ row }">
          <span class="inline-flex items-center gap-1.5 align-middle">
            <IconifyIcon
              v-if="row.icon"
              :icon="row.icon"
              class="size-4 text-current"
            />
            <span>{{ row.title }}</span>
          </span>
          <ElTag v-if="row.isPublic" size="small" class="ml-1">公共</ElTag>
          <ElTag v-if="row.hideInMenu" type="info" size="small" class="ml-1"
            >隐藏</ElTag
          >
        </template>
      </ElTableColumn>
      <ElTableColumn v-if="!isSuperAdmin" label="备注" min-width="200">
        <template #default="{ row }">
          {{ (row as SystemMenuItem).remark || '—' }}
        </template>
      </ElTableColumn>
      <ElTableColumn
        v-if="!isSuperAdmin"
        prop="createdAt"
        label="创建时间"
        width="170"
      />
      <ElTableColumn v-if="isSuperAdmin" prop="name" label="路由名称" min-width="130" />
      <ElTableColumn v-if="isSuperAdmin" prop="path" label="路径" min-width="110" />
      <ElTableColumn
        v-if="isSuperAdmin"
        prop="component"
        label="组件"
        min-width="180"
        show-overflow-tooltip
      />
      <ElTableColumn v-if="isSuperAdmin" label="类型" width="80">
        <template #default="{ row }">
          <ElTag :type="row.type === 'catalog' ? 'warning' : 'success'" size="small">
            {{ typeText(row.type) }}
          </ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn v-if="isSuperAdmin" prop="sort" label="排序" width="70" />
      <ElTableColumn label="状态" width="80">
        <template #default="{ row }">
          <ElTag :type="row.status === 1 ? 'success' : 'danger'" size="small">
            {{ row.status === 1 ? '启用' : '停用' }}
          </ElTag>
        </template>
      </ElTableColumn>
      <ElTableColumn v-if="isSuperAdmin" label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <ElButton link type="primary" @click="openCreate(row as SystemMenuItem)">
            添加下级
          </ElButton>
          <ElButton link type="primary" @click="openEdit(row as SystemMenuItem)">
            编辑
          </ElButton>
          <ElButton link type="danger" @click="handleDelete(row as SystemMenuItem)">
            删除
          </ElButton>
        </template>
      </ElTableColumn>
    </ElTable>

    <!-- 新增/编辑 -->
    <ElDialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑路由' : '新增路由'"
      width="680px"
      destroy-on-close
    >
      <ElForm ref="formRef" :model="form" :rules="rules" label-width="100px">
        <ElFormItem label="上级菜单">
          <ElSelect v-model="form.pid" class="w-full">
            <ElOption
              v-for="opt in parentOptions"
              :key="opt.id"
              :label="opt.label"
              :value="opt.id"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="类型">
          <ElRadioGroup v-model="form.type">
            <ElRadio value="catalog">目录</ElRadio>
            <ElRadio value="menu">菜单</ElRadio>
          </ElRadioGroup>
        </ElFormItem>
        <ElFormItem label="菜单标题" prop="title">
          <ElInput v-model="form.title" placeholder="显示在侧边栏的名称" />
        </ElFormItem>
        <ElFormItem label="路由名称" prop="name">
          <ElInput v-model="form.name" placeholder="唯一, 如 SystemUser" />
        </ElFormItem>
        <ElFormItem label="路由路径" prop="path">
          <ElInput
            v-model="form.path"
            placeholder="顶级以 / 开头, 子级相对路径(如 user)"
          />
        </ElFormItem>
        <ElFormItem label="组件路径">
          <ElInput
            v-model="form.component"
            placeholder="叶子菜单必填, 如 /system/user/index; 目录留空"
          />
          <div class="mt-1 text-xs text-gray-400">
            多级路由: 目录节点组件留空, 仅最底层菜单页面需要组件路径
          </div>
        </ElFormItem>
        <ElFormItem label="重定向">
          <ElInput v-model="form.redirect" placeholder="可选, 如 /system/user" />
        </ElFormItem>
        <ElFormItem label="图标">
          <IconPicker v-model="form.icon" />
        </ElFormItem>
        <ElFormItem label="排序">
          <ElInputNumber v-model="form.sort" :min="1" :step="1" />
          <div class="mt-1 text-xs text-gray-400">
            同级内唯一, 数字越小越靠前, 1 为最前
          </div>
        </ElFormItem>
        <ElFormItem label="状态">
          <ElSwitch
            :model-value="form.status"
            :active-value="1"
            :inactive-value="0"
            @update:model-value="(v) => (form.status = Number(v))"
          />
        </ElFormItem>
        <ElFormItem label="隐藏菜单">
          <ElSwitch v-model="form.hideInMenu" />
        </ElFormItem>
        <ElFormItem label="页面缓存">
          <ElSwitch v-model="form.keepAlive" />
        </ElFormItem>
        <ElFormItem label="固定标签页">
          <ElSwitch v-model="form.affixTab" />
        </ElFormItem>
        <ElFormItem label="公共菜单">
          <ElSwitch v-model="form.isPublic" />
          <span class="ml-2 text-xs text-gray-400">所有登录用户可见, 无需授权</span>
        </ElFormItem>
        <ElFormItem label="备注">
          <ElInput v-model="form.remark" type="textarea" :rows="2" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="saving" @click="handleSave">保存</ElButton>
      </template>
    </ElDialog>

    <!-- 菜单授权 -->
    <ElDialog
      v-model="grantVisible"
      title="菜单授权"
      width="620px"
      destroy-on-close
    >
      <div class="mb-3">
        <ElSelect
          v-model="grantUserId"
          v-loading="grantLoading"
          placeholder="选择要授权的账号"
          filterable
          class="w-full"
          @change="handleGrantUserChange"
        >
          <ElOption
            v-for="u in grantUsers"
            :key="u.id"
            :label="`${u.username} (${u.role === 'admin' ? '管理员' : '用户'})`"
            :value="u.id"
          />
        </ElSelect>
        <div class="mt-1 text-xs text-gray-400">
          {{
            isSuperAdmin
              ? '超级管理员可将任意路由下发给管理员或用户'
              : '管理员只能将超管下发给自己的路由分配给自己的下级用户'
          }}
        </div>
      </div>
      <div v-loading="grantLoading" class="h-[360px] overflow-auto border rounded p-2">
        <ElTree
          v-if="grantMenus.length > 0"
          ref="grantTree"
          :data="grantMenus"
          node-key="id"
          show-checkbox
          default-expand-all
          :props="{ label: 'title', children: 'children' }"
        />
        <div v-else class="text-gray-400">请选择账号后加载可授权菜单</div>
      </div>
      <template #footer>
        <ElButton @click="grantVisible = false">取消</ElButton>
        <ElButton
          type="primary"
          :loading="grantSaving"
          :disabled="!grantUserId"
          @click="handleGrantSave"
          >保存授权</ElButton
        >
      </template>
    </ElDialog>
  </div>
</template>

<style scoped>
:deep(.el-table th.el-table__cell),
:deep(.el-table td.el-table__cell) {
  text-align: center;
}
/* 树表首列: 去掉层级缩进占位, 让展开箭头+图标+标题整体居中, 与其他列对齐 */
:deep(.el-table .el-table__cell .el-table__indent) {
  display: none;
}
/* 子行展开动画: 淡入 + 轻微下滑, 仅作用于层级 >= 1 的行(level-0 不含动画, 数据加载时不会整表闪动) */
:deep(.el-table__body tr[class~='el-table__row--level-1']),
:deep(.el-table__body tr[class~='el-table__row--level-2']),
:deep(.el-table__body tr[class~='el-table__row--level-3']),
:deep(.el-table__body tr[class~='el-table__row--level-4']),
:deep(.el-table__body tr[class~='el-table__row--level-5']) {
  animation: menu-sub-row-in 0.22s ease-out both;
}

@keyframes menu-sub-row-in {
  from {
    opacity: 0;
    transform: translateY(-6px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
