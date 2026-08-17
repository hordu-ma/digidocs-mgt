<script setup lang="ts">
import { Connection } from "@element-plus/icons-vue";
</script>

<template>
  <section class="page-card empty-state no-project-state">
    <ElIcon :size="28"><Connection /></ElIcon>
    <p class="empty-title">你还不属于任何课题</p>
    <p class="empty-hint">
      注册只创建登录账号，不会自动加入课题组。请向课题负责人申请加入，或打开对方发来的邀请链接。
    </p>
    <p class="empty-hint">负责人可在「成员」页生成邀请链接，管理员也可在系统管理中分配课题。</p>
  </section>
</template>

<style scoped>
.no-project-state {
  min-height: 280px;
  gap: 4px;
}
</style>
