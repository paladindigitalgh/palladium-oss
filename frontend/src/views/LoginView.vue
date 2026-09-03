<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import BaseCard from '@/components/base/BaseCard.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import { useAuth } from '@/composables/useAuth'
import { ApiError } from '@/services/api/httpClient'

/**
 * The one public route (see router/index.ts's guard). This is
 * intentionally a plain, standalone screen -- no AppShell, no sidebar,
 * no search -- since there is no session yet for any of that global
 * chrome to act on.
 */
const email = ref('')
const password = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

const { login } = useAuth()
const router = useRouter()
const route = useRoute()

async function handleSubmit() {
  error.value = null
  submitting.value = true
  try {
    await login(email.value, password.value)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard'
    await router.push(redirect)
  } catch (err) {
    error.value = err instanceof ApiError ? err.message : 'Unable to reach the server.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="login">
    <BaseCard class="login__card" elevated>
      <div class="login__header">
        <h1 class="login__title">Palladium OSS</h1>
        <p class="login__subtitle">Sign in to continue</p>
      </div>

      <form class="login__form" @submit.prevent="handleSubmit">
        <label class="login__field">
          <span class="login__label">Email</span>
          <input
            v-model="email"
            type="email"
            required
            autocomplete="username"
            class="login__input"
            :disabled="submitting"
          />
        </label>

        <label class="login__field">
          <span class="login__label">Password</span>
          <input
            v-model="password"
            type="password"
            required
            autocomplete="current-password"
            class="login__input"
            :disabled="submitting"
          />
        </label>

        <p v-if="error" class="login__error" role="alert">{{ error }}</p>

        <BaseButton type="submit" variant="primary" :disabled="submitting" class="login__submit">
          {{ submitting ? 'Signing in…' : 'Sign in' }}
        </BaseButton>
      </form>
    </BaseCard>
  </div>
</template>

<style scoped>
.login {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: var(--color-bg);
  padding: var(--space-4);
}

.login__card {
  width: 100%;
  max-width: 360px;
}

.login__header {
  margin-bottom: var(--space-5);
  text-align: center;
}

.login__title {
  font-size: var(--font-size-xl);
  font-weight: var(--font-weight-semibold);
  color: var(--color-text-primary);
}

.login__subtitle {
  margin-top: var(--space-1);
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.login__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.login__field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.login__label {
  font-size: var(--font-size-xs);
  font-weight: var(--font-weight-medium);
  color: var(--color-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.login__input {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border-strong);
  border-radius: var(--radius-sm);
  background-color: var(--color-surface);
  color: var(--color-text-primary);
  font: inherit;
  font-size: var(--font-size-sm);
}

.login__input:focus-visible {
  outline: 2px solid var(--color-brand);
  outline-offset: 2px;
}

.login__error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
}

.login__submit {
  width: 100%;
}
</style>
