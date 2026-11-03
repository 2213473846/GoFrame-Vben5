<script lang="ts" setup>
import type { DictFallbackItem } from '#/composables/useDict';

import { computed } from 'vue';

import { ElTag } from 'element-plus';

import { useDict } from '#/composables/useDict';

defineOptions({ name: 'DictTag' });

const props = withDefaults(
  defineProps<{
    /** 兜底文案与标签色, key 为字典值; 字典未命中时使用, 避免页面白屏 */
    fallback?: Record<string, DictFallbackItem>;
    /** 字典类型编码, 如 sys_user_status */
    type: string;
    /** 字典值 */
    value?: number | string;
  }>(),
  {
    fallback: () => ({}),
    value: '',
  },
);

const { labelOf, tagOf } = useDict(props.type, props.fallback);

// 字典未加载/无匹配时 labelOf 会原样回显 value
const text = computed(() => labelOf(props.value as number | string));

const tagType = computed(() => tagOf(props.value as number | string));
</script>

<template>
  <ElTag :type="tagType">{{ text }}</ElTag>
</template>
