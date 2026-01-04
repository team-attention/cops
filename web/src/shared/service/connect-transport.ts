import { Code, ConnectError, createClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'
import { create } from '@bufbuild/protobuf'
import type { Interceptor } from '@connectrpc/connect'
import type { TokenPair } from '@/gen/grpcstub/auth/v1/auth_pb'
import {
  AuthService,
  RefreshTokenReqSchema,
} from '@/gen/grpcstub/auth/v1/auth_pb'
import {
  getAccessToken,
  getRefreshToken,
  useAuthStore,
} from '@/shared/store/auth-store'

// RefreshState represents the current state of token refresh operation
interface RefreshState {
  // promise holds the ongoing refresh promise, null if no refresh in progress
  promise: Promise<TokenPair> | null
}

// refreshState holds the singleton state for refresh deduplication
const refreshState: RefreshState = {
  promise: null,
}

// AUTH_SERVICE_PREFIX is the URL path prefix for auth service endpoints that should not trigger token refresh
const AUTH_SERVICE_PREFIX = '/auth.v1.AuthService/'

// getBaseUrl returns the API base URL from environment or default
const getBaseUrl = (): string => {
  return import.meta.env.VITE_API_URL || 'http://localhost:8080'
}

// createBaseTransport creates a transport without auth interceptor for refresh calls
const createBaseTransport = () => {
  // Implementation outline:
  // 1. Call createConnectTransport with baseUrl from getBaseUrl()
  return createConnectTransport({
    baseUrl: getBaseUrl(),
  })
  // 2. Return the transport instance
}

// performTokenRefresh executes the actual refresh token RPC call
const performTokenRefresh = async (): Promise<TokenPair> => {
  // Implementation outline:
  // 1. Get refresh token using getRefreshToken() from auth-store
  const refreshToken = getRefreshToken()
  // 2. If refreshToken is null or empty string:
  //    a. Throw new Error('No refresh token available')
  if (!refreshToken || refreshToken.length === 0) {
    throw new Error('No refresh token available')
  }
  // 3. Create baseTransport using createBaseTransport()
  const baseTransport = createBaseTransport()
  // 4. Create AuthService client with baseTransport
  const client = createClient(AuthService, baseTransport)
  // 5. Create RefreshTokenReq using create(RefreshTokenReqSchema, { refreshToken })
  const request = create(RefreshTokenReqSchema, { refreshToken })
  // 6. Call client.refreshToken(request) and await response
  const response = await client.refreshToken(request)

  // 7. If response.tokens is undefined or null:
  //    a. Throw new Error('Invalid refresh response: no tokens returned')
  if (!response.tokens) {
    throw new Error('Invalid refresh response: no tokens returned')
  }
  // 8. Return response.tokens
  return response.tokens
}

// refreshTokenWithDeduplication handles token refresh with request deduplication
const refreshTokenWithDeduplication = async (): Promise<TokenPair> => {
  // Implementation outline:
  // 1. If refreshState.promise is not null:
  //    a. Return refreshState.promise (deduplicate concurrent refresh requests)
  if (refreshState.promise !== null) {
    return refreshState.promise
  }

  // 2. Set refreshState.promise = performTokenRefresh()
  refreshState.promise = performTokenRefresh()

  // 3. Try block:
  try {
    //    a. Await tokens from refreshState.promise
    const tokens = await refreshState.promise
    //    b. Call useAuthStore.getState().updateTokens(tokens) to update tokens
    useAuthStore.getState().updateTokens(tokens)
    //    c. Return tokens
    return tokens
  } catch (error) {
    // 4. Catch block (error):
    //    a. Call useAuthStore.getState().logout() to clear state and tokens
    useAuthStore.getState().logout()
    //    b. Throw error (let caller handle navigation)
    throw error
  } finally {
    // 5. Finally block:
    //    a. Set refreshState.promise = null (reset deduplication state)
    refreshState.promise = null
  }
}

// createAuthInterceptor creates an interceptor that adds JWT and handles token refresh
const createAuthInterceptor = (): Interceptor => {
  // Implementation outline:
  // 1. Return interceptor function: (next) => async (req) => { ... }
  return (next) => async (req) => {
    // 2. Inside interceptor:
    //    a. Get access token using getAccessToken() from auth-store
    const token = getAccessToken()
    //    b. If token exists and token.length > 0:
    //       i. Set Authorization header: req.header.set('Authorization', `Bearer ${token}`)
    if (token && token.length > 0) {
      req.header.set('Authorization', `Bearer ${token}`)
    }

    //    c. Try block:
    try {
      //       i. Await response from next(req)
      const response = await next(req)
      //       ii. Return response
      return response
    } catch (error) {
      //    d. Catch block (error):
      //       i. If error is ConnectError and error.code === Code.Unauthenticated:
      if (
        error instanceof ConnectError &&
        error.code === Code.Unauthenticated
      ) {
        // Guard 1: Skip token refresh for auth service endpoints
        // Auth endpoints (GoogleAuth, RefreshToken, DeviceCode, DevicePoll) don't require
        // authentication and should propagate their 401 errors directly
        if (req.url.includes(AUTH_SERVICE_PREFIX)) {
          throw error
        }

        // Guard 2: Skip token refresh if no refresh token is available
        // This handles the case where tokens have been cleared or user never logged in
        const refreshToken = getRefreshToken()
        if (!refreshToken || refreshToken.length === 0) {
          throw error
        }

        // Both guards passed - attempt token refresh
        //          - Await newTokens from refreshTokenWithDeduplication()
        const newTokens = await refreshTokenWithDeduplication()
        //          - Set Authorization header with new token: req.header.set('Authorization', `Bearer ${newTokens.accessToken}`)
        req.header.set('Authorization', `Bearer ${newTokens.accessToken}`)
        //          - Await and return response from next(req) (retry request)
        return await next(req)
      }
      //       ii. Otherwise:
      //          - Throw error (not an auth error)
      throw error
    }
  }
}

export const transport = createConnectTransport({
  baseUrl: getBaseUrl(),
  interceptors: [createAuthInterceptor()],
})
