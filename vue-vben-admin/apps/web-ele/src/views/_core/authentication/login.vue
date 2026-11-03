<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';

import { computed, h, reactive, ref } from 'vue';

import { AuthenticationLogin, z } from '@vben/common-ui';
import { $t } from '@vben/locales';

import { ElButton, ElDialog, ElInput, ElMessage } from 'element-plus';

import { changePasswordApi, getCaptchaApi } from '#/api';
import { useAuthStore } from '#/store';

defineOptions({ name: 'Login' });

const authStore = useAuthStore();

// 图形验证码: 仅在失败次数达到后端阈值后展示, 未触发时保持原来的账号密码登录
const captchaRequired = ref(false);
const captchaId = ref('');
const captchaImage = ref('');

// 首次登录/弱密码强制改密弹窗
const changeDialogVisible = ref(false);
const changeLoading = ref(false);
const changeForm = reactive({ oldPassword: '', newPassword: '' });

function errorCode(error: any): number {
  return Number(error?.response?.data?.code ?? 0);
}

async function loadCaptcha() {
  try {
    const res = await getCaptchaApi();
    captchaId.value = res.captchaId;
    captchaImage.value = res.image;
  } catch {
    // 通用错误拦截器已提示
  }
}

const formSchema = computed((): VbenFormSchema[] => {
  const schema: VbenFormSchema[] = [
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: $t('authentication.usernameTip'),
      },
      fieldName: 'username',
      label: $t('authentication.username'),
      rules: z.string().min(1, { message: $t('authentication.usernameTip') }),
    },
    {
      component: 'VbenInputPassword',
      componentProps: {
        placeholder: $t('authentication.password'),
      },
      fieldName: 'password',
      label: $t('authentication.password'),
      rules: z.string().min(1, { message: $t('authentication.passwordTip') }),
    },
  ];
  if (captchaRequired.value) {
    schema.push({
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入图形验证码',
      },
      fieldName: 'captchaCode',
      label: '验证码',
      rules: z.string().min(1, { message: '请输入图形验证码' }),
      // 验证码图片作为输入框后缀, 点击可刷新
      suffix: (() =>
        h('img', {
          alt: '验证码',
          src: captchaImage.value,
          style: 'height:32px;cursor:pointer;border-radius:4px;margin-left:8px',
          title: '点击刷新验证码',
          onClick: () => loadCaptcha(),
        })) as any,
    });
  }
  return schema;
});

async function handleLogin(values: Record<string, any>) {
  try {
    const res = await authStore.authLogin({
      ...values,
      ...(captchaRequired.value
        ? { captchaCode: values.captchaCode, captchaId: captchaId.value }
        : {}),
    });
    if (res?.mustChangePassword) {
      changeForm.oldPassword = String(values.password ?? '');
      changeForm.newPassword = '';
      changeDialogVisible.value = true;
    }
  } catch (error: any) {
    const code = errorCode(error);
    if (code === 4028 || code === 4027) {
      // 后端要求验证码(4028)或验证码错误(4027): 展示验证码并刷新
      captchaRequired.value = true;
      await loadCaptcha();
    } else if (captchaRequired.value) {
      // 验证码一次性使用, 其他登录失败后同样需要刷新, 避免重放旧 captchaId
      await loadCaptcha();
    }
  }
}

async function submitChangePassword() {
  if (!changeForm.newPassword) {
    ElMessage.warning('请输入新密码');
    return;
  }
  changeLoading.value = true;
  try {
    await changePasswordApi({
      newPassword: changeForm.newPassword,
      oldPassword: changeForm.oldPassword,
    });
    changeDialogVisible.value = false;
    // 后端改密成功后会作废全部旧会话(失效重登), 这里清空本地登录态并回到登录页
    ElMessage.success('密码修改成功, 请重新登录');
    await authStore.logout(false);
  } catch {
    // 原密码错误/新密码不符合策略等由通用错误拦截器提示
  } finally {
    changeLoading.value = false;
  }
}

async function cancelChangePassword() {
  changeDialogVisible.value = false;
  await authStore.logout(false);
}
</script>

<template>
  <!-- 保持单一根节点: 登录页容器会把 side-content/w-full/md:max-w-md 等 class 透传给路由组件,
       多根节点(如直接并列 ElDialog)会导致属性无法继承, 登录面板因此失去宽度限制而变形 -->
  <div>
    <AuthenticationLogin
      :form-schema="formSchema"
      :loading="authStore.loginLoading"
      @submit="handleLogin"
    />

    <ElDialog
      v-model="changeDialogVisible"
      append-to-body
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      :show-close="false"
      title="首次登录需修改密码"
      width="420px"
    >
      <div class="space-y-3">
        <p class="text-muted-foreground text-sm">
          为保障账号安全, 请先设置新密码后再使用系统。新密码需满足系统密码策略(默认
          6-32 位且至少包含 1 个数字)。
        </p>
        <ElInput
          v-model="changeForm.newPassword"
          placeholder="请输入新密码"
          show-password
          type="password"
          @keyup.enter="submitChangePassword"
        />
      </div>
      <template #footer>
        <ElButton :disabled="changeLoading" @click="cancelChangePassword">
          返回登录
        </ElButton>
        <ElButton
          :loading="changeLoading"
          type="primary"
          @click="submitChangePassword"
        >
          确认修改
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>
